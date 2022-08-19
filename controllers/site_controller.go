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
	"github.com/netrisai/netriswebapi/http"
	api "github.com/netrisai/netriswebapi/v2"
)

// SiteReconciler reconciles a Site object
type SiteReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=sites,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=sites/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=sites/finalizers,verbs=update

// Reconcile is the main reconciler for the appropriate resource type
func (r *SiteReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	site := &k8sv1alpha1.Site{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	siteCtx, siteCancel := context.WithTimeout(cntxt, contextTimeout)
	defer siteCancel()
	if err := r.Get(siteCtx, req.NamespacedName, site); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	siteMetaNamespaced := req.NamespacedName
	siteMetaNamespaced.Name = string(site.GetUID())
	siteMeta := &k8sv1alpha1.SiteMeta{}
	metaFound := true

	siteMetaCtx, siteMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer siteMetaCancel()
	if err := r.Get(siteMetaCtx, siteMetaNamespaced, siteMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			siteMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if site.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteSite(site, siteMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteSite} %s", err), "")
			return u.patchSiteStatus(site, "Failure", err.Error())
		}
		logger.Info("Site deleted")
		return ctrl.Result{}, nil
	}

	if siteMustUpdateAnnotations(site) {
		debugLogger.Info("Setting default annotations")
		siteUpdateDefaultAnnotations(site)
		sitePatchCtx, sitePatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer sitePatchCancel()
		err := r.Patch(sitePatchCtx, site.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Site default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if siteCompareFieldsForNewMeta(site, siteMeta) {
			debugLogger.Info("Generating New Meta")
			siteID := siteMeta.Spec.ID
			newVnetMeta, err := r.SiteToSiteMeta(site)
			if err != nil {
				logger.Error(fmt.Errorf("{SiteToSiteMeta} %s", err), "")
				return u.patchSiteStatus(site, "Failure", err.Error())
			}
			siteMeta.Spec = newVnetMeta.DeepCopy().Spec
			siteMeta.Spec.ID = siteID
			siteMeta.Spec.SiteCRGeneration = site.GetGeneration()

			siteMetaUpdateCtx, siteMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer siteMetaUpdateCancel()
			err = r.Update(siteMetaUpdateCtx, siteMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{siteMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if site.GetFinalizers() == nil {
			site.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			sitePatchCtx, sitePatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer sitePatchCancel()
			err := r.Patch(sitePatchCtx, site.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Site Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		siteMeta, err := r.SiteToSiteMeta(site)
		if err != nil {
			logger.Error(fmt.Errorf("{SiteToSiteMeta} %s", err), "")
			return u.patchSiteStatus(site, "Failure", err.Error())
		}

		siteMeta.Spec.SiteCRGeneration = site.GetGeneration()

		siteMetaCreateCtx, siteMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer siteMetaCreateCancel()
		if err := r.Create(siteMetaCreateCtx, siteMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{siteMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *SiteReconciler) deleteSite(site *k8sv1alpha1.Site, siteMeta *k8sv1alpha1.SiteMeta) (ctrl.Result, error) {
	if siteMeta != nil && siteMeta.Spec.ID > 0 && !siteMeta.Spec.Reclaim {
		reply, err := r.Cred.Site().Delete(siteMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteSite} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess && resp.Meta.StatusCode != 404 {
			return ctrl.Result{}, fmt.Errorf("{deleteSite} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(site, siteMeta)
}

func (r *SiteReconciler) deleteCRs(site *k8sv1alpha1.Site, siteMeta *k8sv1alpha1.SiteMeta) (ctrl.Result, error) {
	if siteMeta != nil {
		_, err := r.deleteSiteMetaCR(siteMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteSiteCR(site)
}

func (r *SiteReconciler) deleteSiteCR(site *k8sv1alpha1.Site) (ctrl.Result, error) {
	site.ObjectMeta.SetFinalizers(nil)
	site.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, site.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteSiteCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *SiteReconciler) deleteSiteMetaCR(siteMeta *k8sv1alpha1.SiteMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, siteMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteSiteMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager .
func (r *SiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Site{}).
		Complete(r)
}
