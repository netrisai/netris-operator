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

// InventoryProfileReconciler reconciles a InventoryProfile object
type InventoryProfileReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=inventoryprofiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=inventoryprofiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=inventoryprofiles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InventoryProfile object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *InventoryProfileReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	inventoryProfile := &k8sv1alpha1.InventoryProfile{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	inventoryProfileCtx, inventoryProfileCancel := context.WithTimeout(cntxt, contextTimeout)
	defer inventoryProfileCancel()
	if err := r.Get(inventoryProfileCtx, req.NamespacedName, inventoryProfile); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	inventoryProfileMetaNamespaced := req.NamespacedName
	inventoryProfileMetaNamespaced.Name = string(inventoryProfile.GetUID())
	inventoryProfileMeta := &k8sv1alpha1.InventoryProfileMeta{}
	metaFound := true

	inventoryProfileMetaCtx, inventoryProfileMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer inventoryProfileMetaCancel()
	if err := r.Get(inventoryProfileMetaCtx, inventoryProfileMetaNamespaced, inventoryProfileMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			inventoryProfileMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if inventoryProfile.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteInventoryProfile(inventoryProfile, inventoryProfileMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteInventoryProfile} %s", err), "")
			return u.patchInventoryProfileStatus(inventoryProfile, "Failure", err.Error())
		}
		logger.Info("InventoryProfile deleted")
		return ctrl.Result{}, nil
	}

	if inventoryProfileMustUpdateAnnotations(inventoryProfile) {
		debugLogger.Info("Setting default annotations")
		inventoryProfileUpdateDefaultAnnotations(inventoryProfile)
		inventoryProfilePatchCtx, inventoryProfilePatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer inventoryProfilePatchCancel()
		err := r.Patch(inventoryProfilePatchCtx, inventoryProfile.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch InventoryProfile default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if inventoryProfileCompareFieldsForNewMeta(inventoryProfile, inventoryProfileMeta) {
			debugLogger.Info("Generating New Meta")
			inventoryProfileID := inventoryProfileMeta.Spec.ID
			newVnetMeta, err := r.InventoryProfileToInventoryProfileMeta(inventoryProfile)
			if err != nil {
				logger.Error(fmt.Errorf("{InventoryProfileToInventoryProfileMeta} %s", err), "")
				return u.patchInventoryProfileStatus(inventoryProfile, "Failure", err.Error())
			}
			inventoryProfileMeta.Spec = newVnetMeta.DeepCopy().Spec
			inventoryProfileMeta.Spec.ID = inventoryProfileID
			inventoryProfileMeta.Spec.InventoryProfileCRGeneration = inventoryProfile.GetGeneration()

			inventoryProfileMetaUpdateCtx, inventoryProfileMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer inventoryProfileMetaUpdateCancel()
			err = r.Update(inventoryProfileMetaUpdateCtx, inventoryProfileMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{inventoryProfileMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if inventoryProfile.GetFinalizers() == nil {
			inventoryProfile.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			inventoryProfilePatchCtx, inventoryProfilePatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer inventoryProfilePatchCancel()
			err := r.Patch(inventoryProfilePatchCtx, inventoryProfile.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch InventoryProfile Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		inventoryProfileMeta, err := r.InventoryProfileToInventoryProfileMeta(inventoryProfile)
		if err != nil {
			logger.Error(fmt.Errorf("{InventoryProfileToInventoryProfileMeta} %s", err), "")
			return u.patchInventoryProfileStatus(inventoryProfile, "Failure", err.Error())
		}

		inventoryProfileMeta.Spec.InventoryProfileCRGeneration = inventoryProfile.GetGeneration()

		inventoryProfileMetaCreateCtx, inventoryProfileMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer inventoryProfileMetaCreateCancel()
		if err := r.Create(inventoryProfileMetaCreateCtx, inventoryProfileMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{inventoryProfileMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *InventoryProfileReconciler) deleteInventoryProfile(inventoryProfile *k8sv1alpha1.InventoryProfile, inventoryProfileMeta *k8sv1alpha1.InventoryProfileMeta) (ctrl.Result, error) {
	if inventoryProfileMeta != nil && inventoryProfileMeta.Spec.ID > 0 && !inventoryProfileMeta.Spec.Reclaim {
		reply, err := r.Cred.InventoryProfile().Delete(inventoryProfileMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteInventoryProfile} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess {
			return ctrl.Result{}, fmt.Errorf("{deleteInventoryProfile} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(inventoryProfile, inventoryProfileMeta)
}

func (r *InventoryProfileReconciler) deleteCRs(inventoryProfile *k8sv1alpha1.InventoryProfile, inventoryProfileMeta *k8sv1alpha1.InventoryProfileMeta) (ctrl.Result, error) {
	if inventoryProfileMeta != nil {
		_, err := r.deleteInventoryProfileMetaCR(inventoryProfileMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteInventoryProfileCR(inventoryProfile)
}

func (r *InventoryProfileReconciler) deleteInventoryProfileCR(inventoryProfile *k8sv1alpha1.InventoryProfile) (ctrl.Result, error) {
	inventoryProfile.ObjectMeta.SetFinalizers(nil)
	inventoryProfile.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, inventoryProfile.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteInventoryProfileCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *InventoryProfileReconciler) deleteInventoryProfileMetaCR(inventoryProfileMeta *k8sv1alpha1.InventoryProfileMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, inventoryProfileMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteInventoryProfileMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InventoryProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.InventoryProfile{}).
		Complete(r)
}
