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
	"time"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	"github.com/netrisai/netriswebapi/http"
	api "github.com/netrisai/netriswebapi/v2"
	"github.com/netrisai/netriswebapi/v2/types/serverclustertemplate"
)

// ServerClusterTemplateMetaReconciler reconciles a ServerClusterTemplateMeta object
type ServerClusterTemplateMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustertemplatemeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustertemplatemeta/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustertemplatemeta/finalizers,verbs=update

// Reconcile .
func (r *ServerClusterTemplateMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	sctMeta := &k8sv1alpha1.ServerClusterTemplateMeta{}
	sctCR := &k8sv1alpha1.ServerClusterTemplate{}
	sctMetaCtx, sctMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer sctMetaCancel()
	if err := r.Get(sctMetaCtx, req.NamespacedName, sctMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, sctMeta.Spec.ServerClusterTemplateName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "Provisioning"

	sctNN := req.NamespacedName
	sctNN.Name = sctMeta.Spec.ServerClusterTemplateName
	sctNNCtx, sctNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer sctNNCancel()
	if err := r.Get(sctNNCtx, sctNN, sctCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if sctMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if sctMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if sctMeta.Spec.Imported {
			logger.Info("Importing serverclustertemplate")
			debugLogger.Info("Imported yaml mode. Finding ServerClusterTemplate by name")
			// Note: ServerClusterTemplate doesn't have a storage, so we need to fetch from API
			templates, err := r.Cred.ServerClusterTemplate().Get()
			if err != nil {
				logger.Error(fmt.Errorf("{Get ServerClusterTemplates} %s", err), "")
				return u.patchServerClusterTemplateStatus(sctCR, "Failure", err.Error())
			}
			for _, template := range templates {
				if template.Name == sctMeta.Spec.ServerClusterTemplateName {
					debugLogger.Info("Imported yaml mode. ServerClusterTemplate found")
					sctMeta.Spec.ID = template.ID
					sctMeta.Spec.Name = template.Name
					sctMeta.Spec.Vnets = template.Vnets
					sctCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(template.ModifiedDate/1000), 0))
					sctMetaPatchCtx, sctMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
					defer sctMetaPatchCancel()
					err := r.Patch(sctMetaPatchCtx, sctMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
					if err != nil {
						logger.Error(fmt.Errorf("{patch sctmeta.Spec.ID} %s", err), "")
						return u.patchServerClusterTemplateStatus(sctCR, "Failure", err.Error())
					}
					debugLogger.Info("Imported yaml mode. ID patched")
					logger.Info("ServerClusterTemplate imported")
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
			}
			logger.Info("ServerClusterTemplate not found for import")
			debugLogger.Info("Imported yaml mode. ServerClusterTemplate not found")
		}

		logger.Info("Creating ServerClusterTemplate")
		if _, err, errMsg := r.createServerClusterTemplate(sctMeta); err != nil {
			logger.Error(fmt.Errorf("{createServerClusterTemplate} %s", err), "")
			return u.patchServerClusterTemplateStatus(sctCR, "Failure", errMsg.Error())
		}
		logger.Info("ServerClusterTemplate Created")
	} else {
		apiSCT, err := r.Cred.ServerClusterTemplate().GetByID(sctMeta.Spec.ID)
		if err != nil || apiSCT == nil {
			debugLogger.Info("ServerClusterTemplate not found in Netris")
			debugLogger.Info("Going to create ServerClusterTemplate")
			logger.Info("Creating ServerClusterTemplate")
			if _, err, errMsg := r.createServerClusterTemplate(sctMeta); err != nil {
				logger.Error(fmt.Errorf("{createServerClusterTemplate} %s", err), "")
				return u.patchServerClusterTemplateStatus(sctCR, "Failure", errMsg.Error())
			}
			logger.Info("ServerClusterTemplate Created")
		} else {
			provisionState = "Active"
			sctCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(apiSCT.ModifiedDate/1000), 0))
			debugLogger.Info("Comparing ServerClusterTemplateMeta with Netris ServerClusterTemplate",
				"metaName", sctMeta.Spec.ServerClusterTemplateName,
				"apiName", apiSCT.Name)
			if ok := compareServerClusterTemplateMetaAPIServerClusterTemplate(sctMeta, apiSCT, debugLogger); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				// Check if template is in use by any ServerCluster
				serverClusterList := &k8sv1alpha1.ServerClusterList{}
				serverClusterListCtx, serverClusterListCancel := context.WithTimeout(cntxt, contextTimeout)
				defer serverClusterListCancel()
				if err := r.List(serverClusterListCtx, serverClusterList, &client.ListOptions{}); err != nil {
					debugLogger.Info("Failed to list ServerClusters", "error", err)
					// Continue with update attempt if we can't check
				} else {
					for _, sc := range serverClusterList.Items {
						if sc.Spec.Template == sctMeta.Spec.ServerClusterTemplateName {
							logger.Info("ServerClusterTemplate is in use by ServerCluster, skipping update",
								"serverCluster", fmt.Sprintf("%s/%s", sc.Namespace, sc.Name))
							return u.patchServerClusterTemplateStatus(sctCR, "Active",
								fmt.Sprintf("Template is in use by ServerCluster %s/%s and cannot be updated", sc.Namespace, sc.Name))
						}
					}
				}

				debugLogger.Info("Something changed - see previous debug logs for details")
				debugLogger.Info("Go to update ServerClusterTemplate in Netris")
				logger.Info("Updating ServerClusterTemplate")
				updateSCT, err := ServerClusterTemplateMetaToNetrisUpdate(sctMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{ServerClusterTemplateMetaToNetrisUpdate} %s", err), "")
					return u.patchServerClusterTemplateStatus(sctCR, "Failure", err.Error())
				}
				_, err, errMsg := r.updateServerClusterTemplate(sctMeta.Spec.ID, updateSCT)
				if err != nil {
					logger.Error(fmt.Errorf("{updateServerClusterTemplate} %s", err), "")
					return u.patchServerClusterTemplateStatus(sctCR, "Failure", errMsg.Error())
				}
				logger.Info("ServerClusterTemplate Updated")
			}
		}
	}
	return u.patchServerClusterTemplateStatus(sctCR, provisionState, "Success")
}

// SetupWithManager .
func (r *ServerClusterTemplateMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.ServerClusterTemplateMeta{}).
		Complete(r)
}

func (r *ServerClusterTemplateMetaReconciler) createServerClusterTemplate(sctMeta *k8sv1alpha1.ServerClusterTemplateMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", sctMeta.Namespace, sctMeta.Spec.ServerClusterTemplateName),
		"sctName", sctMeta.Spec.ServerClusterTemplateName,
	).V(int(zapcore.WarnLevel))

	sctAdd, err := ServerClusterTemplateMetaToNetris(sctMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	reply, err := r.Cred.ServerClusterTemplate().Add(sctAdd)
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
	err = http.Decode(resp.Data, &idStruct)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ServerClusterTemplate Created", "id", idStruct.ID)

	sctMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, sctMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func (r *ServerClusterTemplateMetaReconciler) updateServerClusterTemplate(id int, sct *serverclustertemplate.ServerClusterTemplateW) (ctrl.Result, error, error) {
	reply, err := r.Cred.ServerClusterTemplate().Update(id, sct)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateServerClusterTemplate} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateServerClusterTemplate} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

