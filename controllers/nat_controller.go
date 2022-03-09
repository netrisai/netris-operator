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

// NatReconciler reconciles a Nat object
type NatReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=nats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=nats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=nats/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Nat object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *NatReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	nat := &k8sv1alpha1.Nat{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	natCtx, natCancel := context.WithTimeout(cntxt, contextTimeout)
	defer natCancel()
	if err := r.Get(natCtx, req.NamespacedName, nat); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	natMetaNamespaced := req.NamespacedName
	natMetaNamespaced.Name = string(nat.GetUID())
	natMeta := &k8sv1alpha1.NatMeta{}
	metaFound := true

	natMetaCtx, natMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer natMetaCancel()
	if err := r.Get(natMetaCtx, natMetaNamespaced, natMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			natMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if nat.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteNat(nat, natMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteNat} %s", err), "")
			return u.patchNatStatus(nat, "Failure", err.Error())
		}
		logger.Info("Nat deleted")
		return ctrl.Result{}, nil
	}

	if natMustUpdateAnnotations(nat) {
		debugLogger.Info("Setting default annotations")
		natUpdateDefaultAnnotations(nat)
		natPatchCtx, natPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer natPatchCancel()
		err := r.Patch(natPatchCtx, nat.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Nat default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if natCompareFieldsForNewMeta(nat, natMeta) {
			debugLogger.Info("Generating New Meta")
			natID := natMeta.Spec.ID
			newVnetMeta, err := r.NatToNatMeta(nat)
			if err != nil {
				logger.Error(fmt.Errorf("{NatToNatMeta} %s", err), "")
				return u.patchNatStatus(nat, "Failure", err.Error())
			}
			natMeta.Spec = newVnetMeta.DeepCopy().Spec
			natMeta.Spec.ID = natID
			natMeta.Spec.NatCRGeneration = nat.GetGeneration()

			natMetaUpdateCtx, natMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer natMetaUpdateCancel()
			err = r.Update(natMetaUpdateCtx, natMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{natMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if nat.GetFinalizers() == nil {
			nat.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			natPatchCtx, natPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer natPatchCancel()
			err := r.Patch(natPatchCtx, nat.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Nat Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		natMeta, err := r.NatToNatMeta(nat)
		if err != nil {
			logger.Error(fmt.Errorf("{NatToNatMeta} %s", err), "")
			return u.patchNatStatus(nat, "Failure", err.Error())
		}

		natMeta.Spec.NatCRGeneration = nat.GetGeneration()

		natMetaCreateCtx, natMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer natMetaCreateCancel()
		if err := r.Create(natMetaCreateCtx, natMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{natMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *NatReconciler) deleteNat(nat *k8sv1alpha1.Nat, natMeta *k8sv1alpha1.NatMeta) (ctrl.Result, error) {
	if natMeta != nil && natMeta.Spec.ID > 0 && !natMeta.Spec.Reclaim {
		reply, err := r.Cred.NAT().Delete(natMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteNat} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess {
			return ctrl.Result{}, fmt.Errorf("{deleteNat} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(nat, natMeta)
}

func (r *NatReconciler) deleteCRs(nat *k8sv1alpha1.Nat, natMeta *k8sv1alpha1.NatMeta) (ctrl.Result, error) {
	if natMeta != nil {
		_, err := r.deleteNatMetaCR(natMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteNatCR(nat)
}

func (r *NatReconciler) deleteNatCR(nat *k8sv1alpha1.Nat) (ctrl.Result, error) {
	nat.ObjectMeta.SetFinalizers(nil)
	nat.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, nat.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteNatCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *NatReconciler) deleteNatMetaCR(natMeta *k8sv1alpha1.NatMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, natMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteNatMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NatReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Nat{}).
		Complete(r)
}
