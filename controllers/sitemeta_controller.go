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

	"github.com/go-logr/logr"
	"github.com/netrisai/netriswebapi/http"
	api "github.com/netrisai/netriswebapi/v2"
	"github.com/netrisai/netriswebapi/v2/types/site"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
)

// SiteMetaReconciler reconciles a SiteMeta object
type SiteMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=sitemeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=sitemeta/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=sitemeta/finalizers,verbs=update

// Reconcile is the main reconciler for the appropriate resource type
func (r *SiteMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	siteMeta := &k8sv1alpha1.SiteMeta{}
	siteCR := &k8sv1alpha1.Site{}
	siteMetaCtx, siteMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer siteMetaCancel()
	if err := r.Get(siteMetaCtx, req.NamespacedName, siteMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, siteMeta.Spec.SiteName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "OK"

	siteNN := req.NamespacedName
	siteNN.Name = siteMeta.Spec.SiteName
	siteNNCtx, siteNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer siteNNCancel()
	if err := r.Get(siteNNCtx, siteNN, siteCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if siteMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if siteMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if siteMeta.Spec.Imported {
			logger.Info("Importing site")
			debugLogger.Info("Imported yaml mode. Finding Site by name")
			if site, ok := r.NStorage.SitesStorage.FindByName(siteMeta.Spec.SiteName); ok {
				debugLogger.Info("Imported yaml mode. Site found")
				siteMeta.Spec.ID = site.ID

				siteMetaPatchCtx, siteMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer siteMetaPatchCancel()
				err := r.Patch(siteMetaPatchCtx, siteMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch sitemeta.Spec.ID} %s", err), "")
					return u.patchSiteStatus(siteCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Site imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Site not found for import")
			debugLogger.Info("Imported yaml mode. Site not found")
		}

		logger.Info("Creating Site")
		if _, err, errMsg := r.createSite(siteMeta); err != nil {
			logger.Error(fmt.Errorf("{createSite} %s", err), "")
			return u.patchSiteStatus(siteCR, "Failure", errMsg.Error())
		}
		logger.Info("Site Created")
	} else {
		if apiSite, ok := r.NStorage.SitesStorage.FindByID(siteMeta.Spec.ID); ok {

			debugLogger.Info("Comparing SiteMeta with Netris Site")
			if ok := compareSiteMetaAPIESite(siteMeta, apiSite, u); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Go to update Site in Netris")
				logger.Info("Updating Site")
				siteUpdate, err := SiteMetaToNetrisUpdate(siteMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{SiteMetaToNetrisUpdate} %s", err), "")
					return u.patchSiteStatus(siteCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(siteUpdate)
				debugLogger.Info("siteUpdate", "payload", string(js))

				_, err, errMsg := updateSite(siteMeta.Spec.ID, siteUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateSite} %s", err), "")
					return u.patchSiteStatus(siteCR, "Failure", errMsg.Error())
				}
				logger.Info("Site Updated")
			}
		} else {
			debugLogger.Info("Site not found in Netris")
			debugLogger.Info("Going to create Site")
			logger.Info("Creating Site")
			if _, err, errMsg := r.createSite(siteMeta); err != nil {
				logger.Error(fmt.Errorf("{createSite} %s", err), "")
				return u.patchSiteStatus(siteCR, "Failure", errMsg.Error())
			}
			logger.Info("Site Created")
		}
	}
	return u.patchSiteStatus(siteCR, provisionState, "Success")
}

func (r *SiteMetaReconciler) createSite(siteMeta *k8sv1alpha1.SiteMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", siteMeta.Namespace, siteMeta.Spec.SiteName),
		"siteName", siteMeta.Spec.SiteCRGeneration,
	).V(int(zapcore.WarnLevel))

	siteAdd, err := SiteMetaToNetris(siteMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(siteAdd)
	debugLogger.Info("siteToAdd", "payload", string(js))

	reply, err := r.Cred.Site().Add(siteAdd)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf(resp.Message), fmt.Errorf(resp.Message)
	}

	idStruct := struct {
		ID int `json:"id"`
	}{}
	debugLogger.Info("response Data", "payload", resp.Data)
	err = http.Decode(resp.Data, &idStruct)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("Site Created", "id", idStruct.ID)

	siteMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, siteMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateSite(id int, site *site.Site, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.Site().Update(id, site)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateSite} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateSite} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager .
func (r *SiteMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.SiteMeta{}).
		Complete(r)
}
