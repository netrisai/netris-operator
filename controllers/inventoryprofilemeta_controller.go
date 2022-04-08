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
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	"github.com/netrisai/netriswebapi/http"
	"github.com/netrisai/netriswebapi/v1/types/inventoryprofile"
	api "github.com/netrisai/netriswebapi/v2"
)

// InventoryProfileMetaReconciler reconciles a InventoryProfileMeta object
type InventoryProfileMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=inventoryprofilemeta,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=inventoryprofilemeta/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=inventoryprofilemeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InventoryProfileMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *InventoryProfileMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	inventoryProfileMeta := &k8sv1alpha1.InventoryProfileMeta{}
	inventoryProfileCR := &k8sv1alpha1.InventoryProfile{}
	inventoryProfileMetaCtx, inventoryProfileMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer inventoryProfileMetaCancel()
	if err := r.Get(inventoryProfileMetaCtx, req.NamespacedName, inventoryProfileMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, inventoryProfileMeta.Spec.InventoryProfileName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "OK"

	inventoryProfileNN := req.NamespacedName
	inventoryProfileNN.Name = inventoryProfileMeta.Spec.InventoryProfileName
	inventoryProfileNNCtx, inventoryProfileNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer inventoryProfileNNCancel()
	if err := r.Get(inventoryProfileNNCtx, inventoryProfileNN, inventoryProfileCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if inventoryProfileMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if inventoryProfileMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if inventoryProfileMeta.Spec.Imported {
			logger.Info("Importing inventoryProfile")
			debugLogger.Info("Imported yaml mode. Finding InventoryProfile by name")
			if inventoryProfile, ok := r.NStorage.InventoryProfileStorage.FindByName(inventoryProfileMeta.Spec.InventoryProfileName); ok {
				debugLogger.Info("Imported yaml mode. InventoryProfile found")
				inventoryProfileMeta.Spec.ID = inventoryProfile.ID

				inventoryProfileMetaPatchCtx, inventoryProfileMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer inventoryProfileMetaPatchCancel()
				err := r.Patch(inventoryProfileMetaPatchCtx, inventoryProfileMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch inventoryProfilemeta.Spec.ID} %s", err), "")
					return u.patchInventoryProfileStatus(inventoryProfileCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("InventoryProfile imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("InventoryProfile not found for import")
			debugLogger.Info("Imported yaml mode. InventoryProfile not found")
		}

		logger.Info("Creating InventoryProfile")
		if _, err, errMsg := r.createInventoryProfile(inventoryProfileMeta); err != nil {
			logger.Error(fmt.Errorf("{createInventoryProfile} %s", err), "")
			return u.patchInventoryProfileStatus(inventoryProfileCR, "Failure", errMsg.Error())
		}
		logger.Info("InventoryProfile Created")
	} else {
		if apiInventoryProfile, ok := r.NStorage.InventoryProfileStorage.FindByID(inventoryProfileMeta.Spec.ID); ok {

			debugLogger.Info("Comparing InventoryProfileMeta with Netris InventoryProfile")
			if ok := compareInventoryProfileMetaAPIEInventoryProfile(inventoryProfileMeta, apiInventoryProfile, u); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Go to update InventoryProfile in Netris")
				logger.Info("Updating InventoryProfile")
				inventoryProfileUpdate, err := InventoryProfileMetaToNetrisUpdate(inventoryProfileMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{InventoryProfileMetaToNetrisUpdate} %s", err), "")
					return u.patchInventoryProfileStatus(inventoryProfileCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(inventoryProfileUpdate)
				debugLogger.Info("inventoryProfileUpdate", "payload", string(js))

				_, err, errMsg := updateInventoryProfile(inventoryProfileMeta.Spec.ID, inventoryProfileUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateInventoryProfile} %s", err), "")
					return u.patchInventoryProfileStatus(inventoryProfileCR, "Failure", errMsg.Error())
				}
				logger.Info("InventoryProfile Updated")
			}
		} else {
			debugLogger.Info("InventoryProfile not found in Netris")
			debugLogger.Info("Going to create InventoryProfile")
			logger.Info("Creating InventoryProfile")
			if _, err, errMsg := r.createInventoryProfile(inventoryProfileMeta); err != nil {
				logger.Error(fmt.Errorf("{createInventoryProfile} %s", err), "")
				return u.patchInventoryProfileStatus(inventoryProfileCR, "Failure", errMsg.Error())
			}
			logger.Info("InventoryProfile Created")
		}
	}
	return u.patchInventoryProfileStatus(inventoryProfileCR, provisionState, "Success")
}

func (r *InventoryProfileMetaReconciler) createInventoryProfile(inventoryProfileMeta *k8sv1alpha1.InventoryProfileMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", inventoryProfileMeta.Namespace, inventoryProfileMeta.Spec.InventoryProfileName),
		"inventoryProfileName", inventoryProfileMeta.Spec.InventoryProfileCRGeneration,
	).V(int(zapcore.WarnLevel))

	inventoryProfileAdd, err := InventoryProfileMetaToNetris(inventoryProfileMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(inventoryProfileAdd)
	debugLogger.Info("inventoryProfileToAdd", "payload", string(js))

	reply, err := r.Cred.InventoryProfile().Add(inventoryProfileAdd)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf(resp.Message), fmt.Errorf(resp.Message)
	}

	idStruct := struct {
		ID int `json:"id"`
	}{}
	debugLogger.Info("response Data", "payload", resp.Data)
	err = http.Decode(resp.Data, &idStruct)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("InventoryProfile Created", "id", idStruct.ID)

	inventoryProfileMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, inventoryProfileMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateInventoryProfile(id int, inventoryProfile *inventoryprofile.ProfileW, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.InventoryProfile().Update(inventoryProfile)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateInventoryProfile} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateInventoryProfile} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InventoryProfileMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.InventoryProfileMeta{}).
		Complete(r)
}
