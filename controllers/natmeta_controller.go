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
	api "github.com/netrisai/netriswebapi/v2"
	"github.com/netrisai/netriswebapi/v2/types/nat"
)

// NatMetaReconciler reconciles a NatMeta object
type NatMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=natmeta,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=natmeta/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=natmeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NatMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *NatMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	natMeta := &k8sv1alpha1.NatMeta{}
	natCR := &k8sv1alpha1.Nat{}
	natMetaCtx, natMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer natMetaCancel()
	if err := r.Get(natMetaCtx, req.NamespacedName, natMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, natMeta.Spec.NatName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "OK"

	natNN := req.NamespacedName
	natNN.Name = natMeta.Spec.NatName
	natNNCtx, natNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer natNNCancel()
	if err := r.Get(natNNCtx, natNN, natCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if natMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if natMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if natMeta.Spec.Imported {
			logger.Info("Importing nat")
			debugLogger.Info("Imported yaml mode. Finding Nat by name")
			if nat, ok := r.NStorage.NATStorage.FindByName(natMeta.Spec.NatName); ok {
				debugLogger.Info("Imported yaml mode. Nat found")
				natMeta.Spec.ID = nat.ID

				natMetaPatchCtx, natMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer natMetaPatchCancel()
				err := r.Patch(natMetaPatchCtx, natMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch natmeta.Spec.ID} %s", err), "")
					return u.patchNatStatus(natCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Nat imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Nat not found for import")
			debugLogger.Info("Imported yaml mode. Nat not found")
		}

		logger.Info("Creating Nat")
		if _, err, errMsg := r.createNat(natMeta); err != nil {
			logger.Error(fmt.Errorf("{createNat} %s", err), "")
			return u.patchNatStatus(natCR, "Failure", errMsg.Error())
		}
		logger.Info("Nat Created")
	} else {
		if apiNat, ok := r.NStorage.NATStorage.FindByID(natMeta.Spec.ID); ok {

			debugLogger.Info("Comparing NatMeta with Netris Nat")
			if ok := compareNatMetaAPIENat(natMeta, apiNat, u); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Go to update Nat in Netris")
				logger.Info("Updating Nat")
				natUpdate, err := NatMetaToNetrisUpdate(natMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{NatMetaToNetrisUpdate} %s", err), "")
					return u.patchNatStatus(natCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(natUpdate)
				debugLogger.Info("natUpdate", "payload", string(js))

				_, err, errMsg := updateNat(natMeta.Spec.ID, natUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateNat} %s", err), "")
					return u.patchNatStatus(natCR, "Failure", errMsg.Error())
				}
				logger.Info("Nat Updated")
			}
		} else {
			debugLogger.Info("Nat not found in Netris")
			debugLogger.Info("Going to create Nat")
			logger.Info("Creating Nat")
			if _, err, errMsg := r.createNat(natMeta); err != nil {
				logger.Error(fmt.Errorf("{createNat} %s", err), "")
				return u.patchNatStatus(natCR, "Failure", errMsg.Error())
			}
			logger.Info("Nat Created")
		}
	}
	return u.patchNatStatus(natCR, provisionState, "Success")
}

func (r *NatMetaReconciler) createNat(natMeta *k8sv1alpha1.NatMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", natMeta.Namespace, natMeta.Spec.NatName),
		"natName", natMeta.Spec.NatCRGeneration,
	).V(int(zapcore.WarnLevel))

	natAdd, err := NatMetaToNetris(natMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(natAdd)
	debugLogger.Info("natToAdd", "payload", string(js))

	reply, err := r.Cred.NAT().Add(natAdd)
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

	debugLogger.Info("Nat Created", "id", idStruct.ID)

	natMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, natMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateNat(id int, nat *nat.NATw, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.NAT().Update(id, nat)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateNat} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateNat} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NatMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.NatMeta{}).
		Complete(r)
}
