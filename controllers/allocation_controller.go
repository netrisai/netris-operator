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
)

// AllocationReconciler reconciles a Allocation object
type AllocationReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=allocations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=allocations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=allocations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Allocation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *AllocationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	allocation := &k8sv1alpha1.Allocation{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	allocationCtx, allocationCancel := context.WithTimeout(cntxt, contextTimeout)
	defer allocationCancel()
	if err := r.Get(allocationCtx, req.NamespacedName, allocation); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	allocationMetaNamespaced := req.NamespacedName
	allocationMetaNamespaced.Name = string(allocation.GetUID())
	allocationMeta := &k8sv1alpha1.AllocationMeta{}
	metaFound := true

	allocationMetaCtx, allocationMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer allocationMetaCancel()
	if err := r.Get(allocationMetaCtx, allocationMetaNamespaced, allocationMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			allocationMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if allocation.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteAllocation(allocation, allocationMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteAllocation} %s", err), "")
			return u.patchAllocationStatus(allocation, "Failure", err.Error())
		}
		logger.Info("Allocation deleted")
		return ctrl.Result{}, nil
	}

	if allocationMustUpdateAnnotations(allocation) {
		debugLogger.Info("Setting default annotations")
		allocationUpdateDefaultAnnotations(allocation)
		allocationPatchCtx, allocationPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer allocationPatchCancel()
		err := r.Patch(allocationPatchCtx, allocation.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Allocation default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if allocationCompareFieldsForNewMeta(allocation, allocationMeta) {
			debugLogger.Info("Generating New Meta")
			allocationID := allocationMeta.Spec.ID
			newVnetMeta, err := r.AllocationToAllocationMeta(allocation)
			if err != nil {
				logger.Error(fmt.Errorf("{AllocationToAllocationMeta} %s", err), "")
				return u.patchAllocationStatus(allocation, "Failure", err.Error())
			}
			allocationMeta.Spec = newVnetMeta.DeepCopy().Spec
			allocationMeta.Spec.ID = allocationID
			allocationMeta.Spec.AllocationCRGeneration = allocation.GetGeneration()

			allocationMetaUpdateCtx, allocationMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer allocationMetaUpdateCancel()
			err = r.Update(allocationMetaUpdateCtx, allocationMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{allocationMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if allocation.GetFinalizers() == nil {
			allocation.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			allocationPatchCtx, allocationPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer allocationPatchCancel()
			err := r.Patch(allocationPatchCtx, allocation.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Allocation Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		allocationMeta, err := r.AllocationToAllocationMeta(allocation)
		if err != nil {
			logger.Error(fmt.Errorf("{AllocationToAllocationMeta} %s", err), "")
			return u.patchAllocationStatus(allocation, "Failure", err.Error())
		}

		allocationMeta.Spec.AllocationCRGeneration = allocation.GetGeneration()

		allocationMetaCreateCtx, allocationMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer allocationMetaCreateCancel()
		if err := r.Create(allocationMetaCreateCtx, allocationMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{allocationMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *AllocationReconciler) deleteAllocation(allocation *k8sv1alpha1.Allocation, allocationMeta *k8sv1alpha1.AllocationMeta) (ctrl.Result, error) {
	if allocationMeta != nil && allocationMeta.Spec.ID > 0 && !allocationMeta.Spec.Reclaim {
		reply, err := r.Cred.IPAM().Delete("allocation", allocationMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteAllocation} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess && resp.Meta.StatusCode != 400 {
			return ctrl.Result{}, fmt.Errorf("{deleteAllocation} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(allocation, allocationMeta)
}

func (r *AllocationReconciler) deleteCRs(allocation *k8sv1alpha1.Allocation, allocationMeta *k8sv1alpha1.AllocationMeta) (ctrl.Result, error) {
	if allocationMeta != nil {
		_, err := r.deleteAllocationMetaCR(allocationMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteAllocationCR(allocation)
}

func (r *AllocationReconciler) deleteAllocationCR(allocation *k8sv1alpha1.Allocation) (ctrl.Result, error) {
	allocation.ObjectMeta.SetFinalizers(nil)
	allocation.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, allocation.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteAllocationCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *AllocationReconciler) deleteAllocationMetaCR(allocationMeta *k8sv1alpha1.AllocationMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, allocationMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteAllocationMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AllocationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Allocation{}).
		Complete(r)
}
