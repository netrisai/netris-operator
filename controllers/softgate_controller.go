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

// SoftgateReconciler reconciles a Softgate object
type SoftgateReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=softgates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=softgates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=softgates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Softgate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *SoftgateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	softgate := &k8sv1alpha1.Softgate{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	softgateCtx, softgateCancel := context.WithTimeout(cntxt, contextTimeout)
	defer softgateCancel()
	if err := r.Get(softgateCtx, req.NamespacedName, softgate); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	softgateMetaNamespaced := req.NamespacedName
	softgateMetaNamespaced.Name = string(softgate.GetUID())
	softgateMeta := &k8sv1alpha1.SoftgateMeta{}
	metaFound := true

	softgateMetaCtx, softgateMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer softgateMetaCancel()
	if err := r.Get(softgateMetaCtx, softgateMetaNamespaced, softgateMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			softgateMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if softgate.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteSoftgate(softgate, softgateMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteSoftgate} %s", err), "")
			return u.patchSoftgateStatus(softgate, "Failure", err.Error())
		}
		logger.Info("Softgate deleted")
		return ctrl.Result{}, nil
	}

	if softgateMustUpdateAnnotations(softgate) {
		debugLogger.Info("Setting default annotations")
		softgateUpdateDefaultAnnotations(softgate)
		softgatePatchCtx, softgatePatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer softgatePatchCancel()
		err := r.Patch(softgatePatchCtx, softgate.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Softgate default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if softgateCompareFieldsForNewMeta(softgate, softgateMeta) {
			debugLogger.Info("Generating New Meta")
			softgateID := softgateMeta.Spec.ID
			newSoftgateMeta, err := r.SoftgateToSoftgateMeta(softgate)
			if err != nil {
				logger.Error(fmt.Errorf("{SoftgateToSoftgateMeta} %s", err), "")
				return u.patchSoftgateStatus(softgate, "Failure", err.Error())
			}
			softgateMeta.Spec = newSoftgateMeta.DeepCopy().Spec
			softgateMeta.Spec.ID = softgateID
			softgateMeta.Spec.SoftgateCRGeneration = softgate.GetGeneration()

			softgateMetaUpdateCtx, softgateMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer softgateMetaUpdateCancel()
			err = r.Update(softgateMetaUpdateCtx, softgateMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{softgateMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if softgate.GetFinalizers() == nil {
			softgate.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			softgatePatchCtx, softgatePatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer softgatePatchCancel()
			err := r.Patch(softgatePatchCtx, softgate.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Softgate Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		softgateMeta, err := r.SoftgateToSoftgateMeta(softgate)
		if err != nil {
			logger.Error(fmt.Errorf("{SoftgateToSoftgateMeta} %s", err), "")
			return u.patchSoftgateStatus(softgate, "Failure", err.Error())
		}

		softgateMeta.Spec.SoftgateCRGeneration = softgate.GetGeneration()

		softgateMetaCreateCtx, softgateMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer softgateMetaCreateCancel()
		if err := r.Create(softgateMetaCreateCtx, softgateMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{softgateMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *SoftgateReconciler) deleteSoftgate(softgate *k8sv1alpha1.Softgate, softgateMeta *k8sv1alpha1.SoftgateMeta) (ctrl.Result, error) {
	if softgateMeta != nil && softgateMeta.Spec.ID > 0 && !softgateMeta.Spec.Reclaim {
		reply, err := r.Cred.Inventory().Delete("softgate", softgateMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteSoftgate} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess && resp.Meta.StatusCode != 400 {
			return ctrl.Result{}, fmt.Errorf("{deleteSoftgate} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(softgate, softgateMeta)
}

func (r *SoftgateReconciler) deleteCRs(softgate *k8sv1alpha1.Softgate, softgateMeta *k8sv1alpha1.SoftgateMeta) (ctrl.Result, error) {
	if softgateMeta != nil {
		_, err := r.deleteSoftgateMetaCR(softgateMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteSoftgateCR(softgate)
}

func (r *SoftgateReconciler) deleteSoftgateCR(softgate *k8sv1alpha1.Softgate) (ctrl.Result, error) {
	softgate.ObjectMeta.SetFinalizers(nil)
	softgate.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, softgate.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteSoftgateCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *SoftgateReconciler) deleteSoftgateMetaCR(softgateMeta *k8sv1alpha1.SoftgateMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, softgateMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteSoftgateMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SoftgateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Softgate{}).
		Complete(r)
}
