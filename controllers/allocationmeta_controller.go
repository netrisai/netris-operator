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
	"github.com/netrisai/netriswebapi/v2/types/ipam"
)

// AllocationMetaReconciler reconciles a AllocationMeta object
type AllocationMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=allocationmeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=allocationmeta/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=allocationmeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AllocationMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *AllocationMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	allocationMeta := &k8sv1alpha1.AllocationMeta{}
	allocationCR := &k8sv1alpha1.Allocation{}
	allocationMetaCtx, allocationMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer allocationMetaCancel()
	if err := r.Get(allocationMetaCtx, req.NamespacedName, allocationMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, allocationMeta.Spec.AllocationName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "Provisioning"

	allocationNN := req.NamespacedName
	allocationNN.Name = allocationMeta.Spec.AllocationName
	allocationNNCtx, allocationNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer allocationNNCancel()
	if err := r.Get(allocationNNCtx, allocationNN, allocationCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if allocationMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if allocationMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if allocationMeta.Spec.Imported {
			logger.Info("Importing allocation")
			debugLogger.Info("Imported yaml mode. Finding Allocation by name")
			if allocation, ok := r.NStorage.SubnetsStorage.FindByName(allocationMeta.Spec.AllocationName); ok {
				debugLogger.Info("Imported yaml mode. Allocation found")
				allocationMeta.Spec.ID = allocation.ID

				allocationMetaPatchCtx, allocationMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer allocationMetaPatchCancel()
				err := r.Patch(allocationMetaPatchCtx, allocationMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch allocationmeta.Spec.ID} %s", err), "")
					return u.patchAllocationStatus(allocationCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Allocation imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Allocation not found for import")
			debugLogger.Info("Imported yaml mode. Allocation not found")
		}

		logger.Info("Creating Allocation")
		if _, err, errMsg := r.createAllocation(allocationMeta); err != nil {
			logger.Error(fmt.Errorf("{createAllocation} %s", err), "")
			return u.patchAllocationStatus(allocationCR, "Failure", errMsg.Error())
		}
		logger.Info("Allocation Created")
	} else {
		if apiAllocation, ok := r.NStorage.SubnetsStorage.FindByID(allocationMeta.Spec.ID); ok {

			debugLogger.Info("Comparing AllocationMeta with Netris Allocation")
			if ok := compareAllocationMetaAPIEAllocation(allocationMeta, apiAllocation, u); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Go to update Allocation in Netris")
				logger.Info("Updating Allocation")
				allocationUpdate, err := AllocationMetaToNetrisUpdate(allocationMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{AllocationMetaToNetrisUpdate} %s", err), "")
					return u.patchAllocationStatus(allocationCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(allocationUpdate)
				debugLogger.Info("allocationUpdate", "payload", string(js))

				_, err, errMsg := updateAllocation(allocationMeta.Spec.ID, allocationUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateAllocation} %s", err), "")
					return u.patchAllocationStatus(allocationCR, "Failure", errMsg.Error())
				}
				logger.Info("Allocation Updated")
			}
		} else {
			debugLogger.Info("Allocation not found in Netris")
			debugLogger.Info("Going to create Allocation")
			logger.Info("Creating Allocation")
			if _, err, errMsg := r.createAllocation(allocationMeta); err != nil {
				logger.Error(fmt.Errorf("{createAllocation} %s", err), "")
				return u.patchAllocationStatus(allocationCR, "Failure", errMsg.Error())
			}
			logger.Info("Allocation Created")
		}
	}
	return u.patchAllocationStatus(allocationCR, provisionState, "Success")
}

func (r *AllocationMetaReconciler) createAllocation(allocationMeta *k8sv1alpha1.AllocationMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", allocationMeta.Namespace, allocationMeta.Spec.AllocationName),
		"allocationName", allocationMeta.Spec.AllocationCRGeneration,
	).V(int(zapcore.WarnLevel))

	allocationAdd, err := AllocationMetaToNetris(allocationMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(allocationAdd)
	debugLogger.Info("allocationToAdd", "payload", string(js))

	reply, err := r.Cred.IPAM().AddAllocation(allocationAdd)
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

	debugLogger.Info("Allocation Created", "id", idStruct.ID)

	allocationMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, allocationMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateAllocation(id int, allocation *ipam.Allocation, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.IPAM().UpdateAllocation(id, allocation)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateAllocation} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateAllocation} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AllocationMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.AllocationMeta{}).
		Complete(r)
}
