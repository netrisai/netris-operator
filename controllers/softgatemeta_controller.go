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

// SoftgateMetaReconciler reconciles a SoftgateMeta object
type SoftgateMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=softgatemeta,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=softgatemeta/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=softgatemeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SoftgateMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *SoftgateMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	softgateMeta := &k8sv1alpha1.SoftgateMeta{}
	softgateCR := &k8sv1alpha1.Softgate{}
	softgateMetaCtx, softgateMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer softgateMetaCancel()
	if err := r.Get(softgateMetaCtx, req.NamespacedName, softgateMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, softgateMeta.Spec.SoftgateName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "OK"

	softgateNN := req.NamespacedName
	softgateNN.Name = softgateMeta.Spec.SoftgateName
	softgateNNCtx, softgateNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer softgateNNCancel()
	if err := r.Get(softgateNNCtx, softgateNN, softgateCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if softgateMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if softgateMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if softgateMeta.Spec.Imported {
			logger.Info("Importing softgate")
			debugLogger.Info("Imported yaml mode. Finding Softgate by name")
			if softgate, ok := r.NStorage.HWsStorage.FindSoftgateByName(softgateMeta.Spec.SoftgateName); ok {
				debugLogger.Info("Imported yaml mode. Softgate found")
				softgateMeta.Spec.ID = softgate.ID
				softgateMeta.Spec.MainIP = softgate.MainIP.Address
				softgateMeta.Spec.MgmtIP = softgate.MgmtIP.Address

				softgateMetaPatchCtx, softgateMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer softgateMetaPatchCancel()
				err := r.Patch(softgateMetaPatchCtx, softgateMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch softgatemeta.Spec.ID} %s", err), "")
					return u.patchSoftgateStatus(softgateCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Softgate imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Softgate not found for import")
			debugLogger.Info("Imported yaml mode. Softgate not found")
		}

		logger.Info("Creating Softgate")
		if _, err, errMsg := r.createSoftgate(softgateMeta); err != nil {
			logger.Error(fmt.Errorf("{createSoftgate} %s", err), "")
			return u.patchSoftgateStatus(softgateCR, "Failure", errMsg.Error())
		}
		logger.Info("Softgate Created")
	} else {
		if apiSoftgate, ok := r.NStorage.HWsStorage.FindSoftgateByID(softgateMeta.Spec.ID); ok {
			debugLogger.Info("Comparing SoftgateMeta with Netris Softgate")

			softgateMeta.Spec.MainIP = apiSoftgate.MainIP.Address
			softgateMeta.Spec.MgmtIP = apiSoftgate.MgmtIP.Address

			if ok := compareSoftgateMetaAPIESoftgate(softgateMeta, apiSoftgate, u); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Go to update Softgate in Netris")
				logger.Info("Updating Softgate")
				softgateUpdate, err := SoftgateMetaToNetrisUpdate(softgateMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{SoftgateMetaToNetrisUpdate} %s", err), "")
					return u.patchSoftgateStatus(softgateCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(softgateUpdate)
				debugLogger.Info("softgateUpdate", "payload", string(js))

				_, err, errMsg := updateSoftgate(softgateMeta.Spec.ID, softgateUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateSoftgate} %s", err), "")
					return u.patchSoftgateStatus(softgateCR, "Failure", errMsg.Error())
				}
				logger.Info("Softgate Updated")
			}
		} else {
			debugLogger.Info("Softgate not found in Netris")
			debugLogger.Info("Going to create Softgate")
			logger.Info("Creating Softgate")
			if _, err, errMsg := r.createSoftgate(softgateMeta); err != nil {
				logger.Error(fmt.Errorf("{createSoftgate} %s", err), "")
				return u.patchSoftgateStatus(softgateCR, "Failure", errMsg.Error())
			}
			logger.Info("Softgate Created")
		}
	}

	if _, err := u.updateSoftgateIfNeccesarry(softgateCR, *softgateMeta); err != nil {
		logger.Error(fmt.Errorf("{updateSoftgateIfNeccesarry} %s", err), "")
		return u.patchSoftgateStatus(softgateCR, "Failure", err.Error())
	}

	return u.patchSoftgateStatus(softgateCR, provisionState, "Success")
}

func (r *SoftgateMetaReconciler) createSoftgate(softgateMeta *k8sv1alpha1.SoftgateMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", softgateMeta.Namespace, softgateMeta.Spec.SoftgateName),
		"softgateName", softgateMeta.Spec.SoftgateCRGeneration,
	).V(int(zapcore.WarnLevel))

	softgateAdd, err := SoftgateMetaToNetris(softgateMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(softgateAdd)
	debugLogger.Info("softgateToAdd", "payload", string(js))

	reply, err := r.Cred.Inventory().AddSoftgate(softgateAdd)
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

	debugLogger.Info("Softgate Created", "id", idStruct.ID)

	softgateMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, softgateMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateSoftgate(id int, softgate *inventory.HWSoftgateUpdate, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.Inventory().UpdateSoftgate(id, softgate)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateSoftgate} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateSoftgate} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SoftgateMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.SoftgateMeta{}).
		Complete(r)
}

func (u *uniReconciler) updateSoftgateIfNeccesarry(softgateCR *v1alpha1.Softgate, softgateMeta v1alpha1.SoftgateMeta) (ctrl.Result, error) {
	shouldUpdateCR := false
	if softgateCR.Spec.MainIP == "" && softgateCR.Spec.MainIP != softgateMeta.Spec.MainIP {
		softgateCR.Spec.MainIP = softgateMeta.Spec.MainIP
		shouldUpdateCR = true
	}
	if softgateCR.Spec.MgmtIP == "" && softgateCR.Spec.MgmtIP != softgateMeta.Spec.MgmtIP {
		softgateCR.Spec.MgmtIP = softgateMeta.Spec.MgmtIP
		shouldUpdateCR = true
	}
	if shouldUpdateCR {
		u.DebugLogger.Info("Updating Softgate CR")
		if _, err := u.patchSoftgate(softgateCR); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
