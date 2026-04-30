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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	"github.com/netrisai/netriswebapi/http"
	api "github.com/netrisai/netriswebapi/v2"
)

// ServerClusterReconciler reconciles a ServerCluster object
type ServerClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclusters/finalizers,verbs=update

// Reconcile servercluster events
func (r *ServerClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	scCR := &k8sv1alpha1.ServerCluster{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	scCtx, scCancel := context.WithTimeout(cntxt, contextTimeout)
	defer scCancel()
	if err := r.Get(scCtx, req.NamespacedName, scCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	scMetaNamespaced := req.NamespacedName
	scMetaNamespaced.Name = string(scCR.GetUID())
	scMeta := &k8sv1alpha1.ServerClusterMeta{}
	metaFound := true
	scMetaCtx, scMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer scMetaCancel()
	if err := r.Get(scMetaCtx, scMetaNamespaced, scMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			scMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if scCR.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteServerCluster(scCR, scMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteServerCluster} %s", err), "")
			return u.patchServerClusterStatus(scCR, "Failure", err.Error())
		}
		logger.Info("ServerCluster deleted")
		return ctrl.Result{}, nil
	}

	if serverClusterMustUpdateAnnotations(scCR) {
		debugLogger.Info("Setting default annotations")
		serverClusterUpdateDefaultAnnotations(scCR)
		scUpdateCtx, scUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer scUpdateCancel()
		err := r.Patch(scUpdateCtx, scCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch ServerCluster default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if serverClusterCompareFieldsForNewMeta(scCR, scMeta) {
			debugLogger.Info("Generating New Meta")
			scID := scMeta.Spec.ID
			newScMeta, err := r.ServerClusterToServerClusterMeta(scCR)
			if err != nil {
				logger.Error(fmt.Errorf("{ServerClusterToServerClusterMeta} %s", err), "")
				return u.patchServerClusterStatus(scCR, "Failure", err.Error())
			}
			scMeta.Spec = newScMeta.DeepCopy().Spec
			scMeta.Spec.ID = scID
			scMeta.Spec.ServerClusterCRGeneration = scCR.GetGeneration()

			scMetaUpdateCtx, scMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer scMetaUpdateCancel()
			err = r.Update(scMetaUpdateCtx, scMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{scMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if scCR.GetFinalizers() == nil {
			scCR.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})
			scPatchCtx, scPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer scPatchCancel()
			err := r.Patch(scPatchCtx, scCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch ServerCluster Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		scMeta, err := r.ServerClusterToServerClusterMeta(scCR)
		if err != nil {
			logger.Error(fmt.Errorf("{ServerClusterToServerClusterMeta} %s", err), "")
			return u.patchServerClusterStatus(scCR, "Failure", err.Error())
		}

		scMeta.Spec.ServerClusterCRGeneration = scCR.GetGeneration()

		scMetaCreateCtx, scMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer scMetaCreateCancel()
		if err := r.Create(scMetaCreateCtx, scMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{scMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *ServerClusterReconciler) deleteServerCluster(scCR *k8sv1alpha1.ServerCluster, scMeta *k8sv1alpha1.ServerClusterMeta) (ctrl.Result, error) {
	if scMeta != nil && scMeta.Spec.ID > 0 && !scMeta.Spec.Reclaim {
		reply, err := r.Cred.ServerCluster().Delete(scMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteServerCluster} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess {
			if resp.Message != "Invalid ServerCluster ID" {
				return ctrl.Result{}, fmt.Errorf("{deleteServerCluster} %s", fmt.Errorf(resp.Message))
			}
		}
	}
	return r.deleteCRs(scCR, scMeta)
}

func (r *ServerClusterReconciler) deleteCRs(scCR *k8sv1alpha1.ServerCluster, scMeta *k8sv1alpha1.ServerClusterMeta) (ctrl.Result, error) {
	if scMeta != nil {
		_, err := r.deleteServerClusterMetaCR(scMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteServerClusterCR(scCR)
}

func (r *ServerClusterReconciler) deleteServerClusterCR(scCR *k8sv1alpha1.ServerCluster) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	scCR.ObjectMeta.SetFinalizers(nil)
	scCR.SetFinalizers(nil)
	if err := r.Update(ctx, scCR.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteServerClusterCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *ServerClusterReconciler) deleteServerClusterMetaCR(scMeta *k8sv1alpha1.ServerClusterMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, scMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteServerClusterMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager Resources
func (r *ServerClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.ServerCluster{}).
		Complete(r)
}

