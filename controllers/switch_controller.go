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

// SwitchReconciler reconciles a Switch object
type SwitchReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=switches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=switches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=switches/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Switch object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *SwitchReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	switchH := &k8sv1alpha1.Switch{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	switchCtx, switchCancel := context.WithTimeout(cntxt, contextTimeout)
	defer switchCancel()
	if err := r.Get(switchCtx, req.NamespacedName, switchH); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	switchMetaNamespaced := req.NamespacedName
	switchMetaNamespaced.Name = string(switchH.GetUID())
	switchMeta := &k8sv1alpha1.SwitchMeta{}
	metaFound := true

	switchMetaCtx, switchMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer switchMetaCancel()
	if err := r.Get(switchMetaCtx, switchMetaNamespaced, switchMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			switchMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if switchH.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteSwitch(switchH, switchMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteSwitch} %s", err), "")
			return u.patchSwitchStatus(switchH, "Failure", err.Error())
		}
		logger.Info("Switch deleted")
		return ctrl.Result{}, nil
	}

	if switchMustUpdateAnnotations(switchH) {
		debugLogger.Info("Setting default annotations")
		switchUpdateDefaultAnnotations(switchH)
		switchPatchCtx, switchPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer switchPatchCancel()
		err := r.Patch(switchPatchCtx, switchH.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Switch default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if switchCompareFieldsForNewMeta(switchH, switchMeta) {
			debugLogger.Info("Generating New Meta")
			switchID := switchMeta.Spec.ID
			newSwitchMeta, err := r.SwitchToSwitchMeta(switchH)
			if err != nil {
				logger.Error(fmt.Errorf("{SwitchToSwitchMeta} %s", err), "")
				return u.patchSwitchStatus(switchH, "Failure", err.Error())
			}
			switchMeta.Spec = newSwitchMeta.DeepCopy().Spec
			switchMeta.Spec.ID = switchID
			switchMeta.Spec.SwitchCRGeneration = switchH.GetGeneration()

			switchMetaUpdateCtx, switchMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer switchMetaUpdateCancel()
			err = r.Update(switchMetaUpdateCtx, switchMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{switchMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if switchH.GetFinalizers() == nil {
			switchH.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			switchPatchCtx, switchPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer switchPatchCancel()
			err := r.Patch(switchPatchCtx, switchH.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Switch Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		switchMeta, err := r.SwitchToSwitchMeta(switchH)
		if err != nil {
			logger.Error(fmt.Errorf("{SwitchToSwitchMeta} %s", err), "")
			return u.patchSwitchStatus(switchH, "Failure", err.Error())
		}

		switchMeta.Spec.SwitchCRGeneration = switchH.GetGeneration()

		switchMetaCreateCtx, switchMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer switchMetaCreateCancel()
		if err := r.Create(switchMetaCreateCtx, switchMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{switchMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *SwitchReconciler) deleteSwitch(switchH *k8sv1alpha1.Switch, switchMeta *k8sv1alpha1.SwitchMeta) (ctrl.Result, error) {
	if switchMeta != nil && switchMeta.Spec.ID > 0 && !switchMeta.Spec.Reclaim {
		reply, err := r.Cred.Inventory().Delete("switch", switchMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteSwitch} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess && resp.Meta.StatusCode != 400 {
			return ctrl.Result{}, fmt.Errorf("{deleteSwitch} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(switchH, switchMeta)
}

func (r *SwitchReconciler) deleteCRs(switchH *k8sv1alpha1.Switch, switchMeta *k8sv1alpha1.SwitchMeta) (ctrl.Result, error) {
	if switchMeta != nil {
		_, err := r.deleteSwitchMetaCR(switchMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteSwitchCR(switchH)
}

func (r *SwitchReconciler) deleteSwitchCR(switchH *k8sv1alpha1.Switch) (ctrl.Result, error) {
	switchH.ObjectMeta.SetFinalizers(nil)
	switchH.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, switchH.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteSwitchCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *SwitchReconciler) deleteSwitchMetaCR(switchMeta *k8sv1alpha1.SwitchMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, switchMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteSwitchMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwitchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Switch{}).
		Complete(r)
}
