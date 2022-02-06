/*
Copyright 2021. Netris, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/netrisai/netris-operator/api/v1alpha1"
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	"github.com/netrisai/netriswebapi/http"
	api "github.com/netrisai/netriswebapi/v2"
	"github.com/netrisai/netriswebapi/v2/types/inventory"
)

// ControllerMetaReconciler reconciles a ControllerMeta object
type ControllerMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=controllermeta,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=controllermeta/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=controllermeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ControllerMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *ControllerMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	controllerMeta := &k8sv1alpha1.ControllerMeta{}
	controllerCR := &k8sv1alpha1.Controller{}
	controllerMetaCtx, controllerMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer controllerMetaCancel()
	if err := r.Get(controllerMetaCtx, req.NamespacedName, controllerMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, controllerMeta.Spec.ControllerName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "OK"

	controllerNN := req.NamespacedName
	controllerNN.Name = controllerMeta.Spec.ControllerName
	controllerNNCtx, controllerNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer controllerNNCancel()
	if err := r.Get(controllerNNCtx, controllerNN, controllerCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if controllerMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if controllerMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if controllerMeta.Spec.Imported {
			logger.Info("Importing controller")
			debugLogger.Info("Imported yaml mode. Finding Controller by name")
			if controller, ok := r.NStorage.HWsStorage.FindControllerByName(controllerMeta.Spec.ControllerName); ok {
				debugLogger.Info("Imported yaml mode. Controller found")
				controllerMeta.Spec.ID = controller.ID
				controllerMeta.Spec.MainIP = controller.MainIP.Address

				controllerMetaPatchCtx, controllerMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer controllerMetaPatchCancel()
				err := r.Patch(controllerMetaPatchCtx, controllerMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch controllermeta.Spec.ID} %s", err), "")
					return u.patchControllerStatus(controllerCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Controller imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Controller not found for import")
			debugLogger.Info("Imported yaml mode. Controller not found")
		}

		logger.Info("Creating Controller")
		if _, err, errMsg := r.createController(controllerMeta); err != nil {
			logger.Error(fmt.Errorf("{createController} %s", err), "")
			return u.patchControllerStatus(controllerCR, "Failure", errMsg.Error())
		}
		logger.Info("Controller Created")
	} else {
		if apiController, ok := r.NStorage.HWsStorage.FindControllerByID(controllerMeta.Spec.ID); ok {
			debugLogger.Info("Comparing ControllerMeta with Netris Controller")

			if ok := compareControllerMetaAPIEController(controllerMeta, apiController, u); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Go to update Controller in Netris")
				logger.Info("Updating Controller")
				controllerUpdate, err := ControllerMetaToNetrisUpdate(controllerMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{ControllerMetaToNetrisUpdate} %s", err), "")
					return u.patchControllerStatus(controllerCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(controllerUpdate)
				debugLogger.Info("controllerUpdate", "payload", string(js))

				_, err, errMsg := updateController(controllerMeta.Spec.ID, controllerUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateController} %s", err), "")
					return u.patchControllerStatus(controllerCR, "Failure", errMsg.Error())
				}
				logger.Info("Controller Updated")
			}
			controllerMeta.Spec.MainIP = apiController.MainIP.Address
		} else {
			debugLogger.Info("Controller not found in Netris")
			debugLogger.Info("Going to create Controller")
			logger.Info("Creating Controller")
			if _, err, errMsg := r.createController(controllerMeta); err != nil {
				logger.Error(fmt.Errorf("{createController} %s", err), "")
				return u.patchControllerStatus(controllerCR, "Failure", errMsg.Error())
			}
			logger.Info("Controller Created")
		}
	}

	if _, err := u.updateControllerIfNeccesarry(controllerCR, *controllerMeta); err != nil {
		logger.Error(fmt.Errorf("{updateControllerIfNeccesarry} %s", err), "")
		return u.patchControllerStatus(controllerCR, "Failure", err.Error())
	}

	return u.patchControllerStatus(controllerCR, provisionState, "Success")
}

func (r *ControllerMetaReconciler) createController(controllerMeta *k8sv1alpha1.ControllerMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", controllerMeta.Namespace, controllerMeta.Spec.ControllerName),
		"controllerName", controllerMeta.Spec.ControllerCRGeneration,
	).V(int(zapcore.WarnLevel))

	controllerAdd, err := ControllerMetaToNetris(controllerMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(controllerAdd)
	debugLogger.Info("controllerToAdd", "payload", string(js))

	reply, err := r.Cred.Inventory().AddController(controllerAdd)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	idStruct := struct {
		ID int `json:"id"`
	}{}

	data, err := reply.Parse()
	if err != nil {
		return ctrl.Result{}, err, err
	}

	if reply.StatusCode != 200 {
		return ctrl.Result{}, fmt.Errorf(data.Message), fmt.Errorf(data.Message)
	}

	idStruct.ID = int(data.Data.(map[string]interface{})["id"].(float64))

	debugLogger.Info("Controller Created", "id", idStruct.ID)

	controllerMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, controllerMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateController(id int, controller *inventory.HWControllerUpdate, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.Inventory().UpdateController(id, controller)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateController} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateController} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ControllerMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.ControllerMeta{}).
		Complete(r)
}

func (u *uniReconciler) updateControllerIfNeccesarry(controllerCR *v1alpha1.Controller, controllerMeta v1alpha1.ControllerMeta) (ctrl.Result, error) {
	shouldUpdateCR := false
	if controllerCR.Spec.MainIP == "" && controllerCR.Spec.MainIP != controllerMeta.Spec.MainIP {
		controllerCR.Spec.MainIP = controllerMeta.Spec.MainIP
		shouldUpdateCR = true
	}
	if shouldUpdateCR {
		u.DebugLogger.Info("Updating Controller CR")
		if _, err := u.patchController(controllerCR); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
