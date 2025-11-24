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

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	api "github.com/netrisai/netriswebapi/v2"
)

// L4LBReconciler reconciles a L4LB object
type L4LBReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Cred       *api.Clientset
	NStorage   *netrisstorage.Storage
	L4LBTenant string
	VPCID      int
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=l4lbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=l4lbs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=l4lbs/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is the main reconciler for the appropriate resource type
func (r *L4LBReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	l4lb := &k8sv1alpha1.L4LB{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	l4lbCtx, l4lbCancel := context.WithTimeout(cntxt, contextTimeout)
	defer l4lbCancel()
	if err := r.Get(l4lbCtx, req.NamespacedName, l4lb); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	l4lbMetaNamespaced := req.NamespacedName
	l4lbMetaNamespaced.Name = string(l4lb.GetUID())
	l4lbMeta := &k8sv1alpha1.L4LBMeta{}
	metaFound := true
	l4lbMetaCtx, l4lbMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer l4lbMetaCancel()
	if err := r.Get(l4lbMetaCtx, l4lbMetaNamespaced, l4lbMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			l4lbMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if l4lb.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		result, err := r.deleteL4LB(l4lb, l4lbMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteL4LB} %s", err), "")
			return u.patchL4LBStatus(l4lb, "Failure", err.Error())
		}
		if result.IsZero() {
			logger.Info("L4LB deleted")
		}
		return result, nil
	}

	if l4lbMustUpdateAnnotations(l4lb) {
		debugLogger.Info("Setting default annotations")
		l4lbUpdateDefaultAnnotations(l4lb)
		l4lbPatchCtx, l4lbPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer l4lbPatchCancel()
		err := r.Patch(l4lbPatchCtx, l4lb.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch L4LB default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if l4lbCompareFieldsForNewMeta(l4lb, l4lbMeta) {
			debugLogger.Info("Generating New Meta")
			l4lbID := l4lbMeta.Spec.ID
			newL4LBMeta, err := r.L4LBToL4LBMeta(l4lb)
			if err != nil {
				logger.Error(fmt.Errorf("{L4LBToL4LBMeta} %s", err), "")
				return u.patchL4LBStatus(l4lb, "Failure", err.Error())
			}
			l4lbMeta.Spec = newL4LBMeta.DeepCopy().Spec
			l4lbMeta.Spec.ID = l4lbID
			l4lbMeta.Spec.L4LBCRGeneration = l4lb.GetGeneration()

			l4lbMetaUpdateCtx, l4lbMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer l4lbMetaUpdateCancel()
			err = r.Update(l4lbMetaUpdateCtx, l4lbMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{l4lbMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if l4lb.GetFinalizers() == nil {
			l4lb.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})
			l4lbCtx, l4lbCancel := context.WithTimeout(cntxt, contextTimeout)
			defer l4lbCancel()
			err := r.Patch(l4lbCtx, l4lb.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch L4LB Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		l4lbMeta, err := r.L4LBToL4LBMeta(l4lb)
		if err != nil {
			logger.Error(fmt.Errorf("{L4LBToL4LBMeta} %s", err), "")
			return u.patchL4LBStatus(l4lb, "Failure", err.Error())
		}

		l4lbMeta.Spec.L4LBCRGeneration = l4lb.GetGeneration()
		l4lbMeta.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

		l4lbCreateCtx, l4lbCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer l4lbCreateCancel()
		if err := r.Create(l4lbCreateCtx, l4lbMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{l4lbMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *L4LBReconciler) deleteL4LB(l4lb *k8sv1alpha1.L4LB, l4lbMeta *k8sv1alpha1.L4LBMeta) (ctrl.Result, error) {
	return r.deleteCRs(l4lb, l4lbMeta)
}

func (r *L4LBReconciler) deleteCRs(l4lb *k8sv1alpha1.L4LB, l4lbMeta *k8sv1alpha1.L4LBMeta) (ctrl.Result, error) {
	if l4lbMeta != nil {
		_, err := r.deleteL4LBMetaCR(l4lbMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	} else {
		return r.deleteL4LBCR(l4lb)
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *L4LBReconciler) deleteL4LBCR(l4lb *k8sv1alpha1.L4LB) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	l4lb.ObjectMeta.SetFinalizers(nil)
	l4lb.SetFinalizers(nil)
	if err := r.Update(ctx, l4lb.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteL4LBCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *L4LBReconciler) deleteL4LBMetaCR(l4lbMeta *k8sv1alpha1.L4LBMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, l4lbMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteL4LBMetaCR} %s", err)
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// SetupWithManager .
func (r *L4LBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.L4LB{}).
		Complete(r)
}
