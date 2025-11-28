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

// ServerClusterTemplateReconciler reconciles a ServerClusterTemplate object
type ServerClusterTemplateReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustertemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustertemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustertemplates/finalizers,verbs=update

// Reconcile serverclustertemplate events
func (r *ServerClusterTemplateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	sctCR := &k8sv1alpha1.ServerClusterTemplate{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	sctCtx, sctCancel := context.WithTimeout(cntxt, contextTimeout)
	defer sctCancel()
	if err := r.Get(sctCtx, req.NamespacedName, sctCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	sctMetaNamespaced := req.NamespacedName
	sctMetaNamespaced.Name = string(sctCR.GetUID())
	sctMeta := &k8sv1alpha1.ServerClusterTemplateMeta{}
	metaFound := true
	sctMetaCtx, sctMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer sctMetaCancel()
	if err := r.Get(sctMetaCtx, sctMetaNamespaced, sctMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			sctMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if sctCR.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteServerClusterTemplate(sctCR, sctMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteServerClusterTemplate} %s", err), "")
			return u.patchServerClusterTemplateStatus(sctCR, "Failure", err.Error())
		}
		logger.Info("ServerClusterTemplate deleted")
		return ctrl.Result{}, nil
	}

	if serverClusterTemplateMustUpdateAnnotations(sctCR) {
		debugLogger.Info("Setting default annotations")
		serverClusterTemplateUpdateDefaultAnnotations(sctCR)
		sctUpdateCtx, sctUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer sctUpdateCancel()
		err := r.Patch(sctUpdateCtx, sctCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch ServerClusterTemplate default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if serverClusterTemplateCompareFieldsForNewMeta(sctCR, sctMeta) {
			debugLogger.Info("Generating New Meta")
			sctID := sctMeta.Spec.ID
			newSctMeta, err := r.ServerClusterTemplateToServerClusterTemplateMeta(sctCR)
			if err != nil {
				logger.Error(fmt.Errorf("{ServerClusterTemplateToServerClusterTemplateMeta} %s", err), "")
				return u.patchServerClusterTemplateStatus(sctCR, "Failure", err.Error())
			}
			sctMeta.Spec = newSctMeta.DeepCopy().Spec
			sctMeta.Spec.ID = sctID
			sctMeta.Spec.ServerClusterTemplateCRGeneration = sctCR.GetGeneration()

			sctMetaUpdateCtx, sctMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer sctMetaUpdateCancel()
			err = r.Update(sctMetaUpdateCtx, sctMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{sctMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if sctCR.GetFinalizers() == nil {
			sctCR.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})
			sctPatchCtx, sctPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer sctPatchCancel()
			err := r.Patch(sctPatchCtx, sctCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch ServerClusterTemplate Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		sctMeta, err := r.ServerClusterTemplateToServerClusterTemplateMeta(sctCR)
		if err != nil {
			logger.Error(fmt.Errorf("{ServerClusterTemplateToServerClusterTemplateMeta} %s", err), "")
			return u.patchServerClusterTemplateStatus(sctCR, "Failure", err.Error())
		}

		sctMeta.Spec.ServerClusterTemplateCRGeneration = sctCR.GetGeneration()

		sctMetaCreateCtx, sctMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer sctMetaCreateCancel()
		if err := r.Create(sctMetaCreateCtx, sctMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{sctMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *ServerClusterTemplateReconciler) deleteServerClusterTemplate(sctCR *k8sv1alpha1.ServerClusterTemplate, sctMeta *k8sv1alpha1.ServerClusterTemplateMeta) (ctrl.Result, error) {
	if sctMeta != nil && sctMeta.Spec.ID > 0 && !sctMeta.Spec.Reclaim {
		reply, err := r.Cred.ServerClusterTemplate().Delete(sctMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteServerClusterTemplate} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess {
			if resp.Message != "Invalid ServerClusterTemplate ID" {
				return ctrl.Result{}, fmt.Errorf("{deleteServerClusterTemplate} %s", fmt.Errorf(resp.Message))
			}
		}
	}
	return r.deleteCRs(sctCR, sctMeta)
}

func (r *ServerClusterTemplateReconciler) deleteCRs(sctCR *k8sv1alpha1.ServerClusterTemplate, sctMeta *k8sv1alpha1.ServerClusterTemplateMeta) (ctrl.Result, error) {
	if sctMeta != nil {
		_, err := r.deleteServerClusterTemplateMetaCR(sctMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteServerClusterTemplateCR(sctCR)
}

func (r *ServerClusterTemplateReconciler) deleteServerClusterTemplateCR(sctCR *k8sv1alpha1.ServerClusterTemplate) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	sctCR.ObjectMeta.SetFinalizers(nil)
	sctCR.SetFinalizers(nil)
	if err := r.Update(ctx, sctCR.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteServerClusterTemplateCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *ServerClusterTemplateReconciler) deleteServerClusterTemplateMetaCR(sctMeta *k8sv1alpha1.ServerClusterTemplateMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, sctMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteServerClusterTemplateMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager Resources
func (r *ServerClusterTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.ServerClusterTemplate{}).
		Complete(r)
}

