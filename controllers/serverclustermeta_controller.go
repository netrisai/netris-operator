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
	"github.com/netrisai/netriswebapi/v2/types/servercluster"
)

// ServerClusterMetaReconciler reconciles a ServerClusterMeta object
type ServerClusterMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustermeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustermeta/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=serverclustermeta/finalizers,verbs=update

// Reconcile .
func (r *ServerClusterMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	scMeta := &k8sv1alpha1.ServerClusterMeta{}
	scCR := &k8sv1alpha1.ServerCluster{}
	scMetaCtx, scMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer scMetaCancel()
	if err := r.Get(scMetaCtx, req.NamespacedName, scMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, scMeta.Spec.ServerClusterName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "Provisioning"

	scNN := req.NamespacedName
	scNN.Name = scMeta.Spec.ServerClusterName
	scNNCtx, scNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer scNNCancel()
	if err := r.Get(scNNCtx, scNN, scCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info("ServerCluster CR not found, deleting ServerClusterMeta")
			// ServerCluster CR was deleted, clean up ServerClusterMeta
			if scMeta.Spec.ID > 0 && !scMeta.Spec.Reclaim {
				debugLogger.Info("Deleting ServerCluster from Netris", "id", scMeta.Spec.ID)
				reply, err := r.Cred.ServerCluster().Delete(scMeta.Spec.ID)
				if err != nil {
					logger.Error(fmt.Errorf("{deleteServerCluster} %s", err), "")
				} else {
					resp, err := http.ParseAPIResponse(reply.Data)
					if err == nil && !resp.IsSuccess {
						if resp.Message != "Invalid ServerCluster ID" {
							debugLogger.Info("Failed to delete ServerCluster from Netris", "error", resp.Message)
						}
					}
				}
			}
			// Delete the ServerClusterMeta object
			scMetaDeleteCtx, scMetaDeleteCancel := context.WithTimeout(cntxt, contextTimeout)
			defer scMetaDeleteCancel()
			if err := r.Delete(scMetaDeleteCtx, scMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
				if !errors.IsNotFound(err) {
					logger.Error(fmt.Errorf("{deleteServerClusterMeta} %s", err), "")
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
			}
			debugLogger.Info("ServerClusterMeta deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if scMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if scMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		// First, try to find existing ServerCluster by name (for both import and non-import cases)
		debugLogger.Info("Checking if ServerCluster exists in Netris by name")
		clusters, err := r.Cred.ServerCluster().Get()
		if err != nil {
			logger.Error(fmt.Errorf("{Get ServerClusters} %s", err), "")
			return u.patchServerClusterStatus(scCR, "Failure", err.Error())
		}
		for _, cluster := range clusters {
			if cluster.Name == scMeta.Spec.ServerClusterName {
				debugLogger.Info("ServerCluster found in Netris by name, importing")
				scMeta.Spec.ID = cluster.ID
				scMeta.Spec.AdminID = cluster.Admin.ID
				scMeta.Spec.Admin = cluster.Admin.Name
				scMeta.Spec.SiteID = cluster.Site.ID
				scMeta.Spec.Site = cluster.Site.Name
				scMeta.Spec.VPCID = cluster.VPC.ID
				scMeta.Spec.VPC = cluster.VPC.Name
				scMeta.Spec.TemplateID = cluster.SrvClusterTemplate.ID
				scMeta.Spec.Template = cluster.SrvClusterTemplate.Name
				scMeta.Spec.Tags = cluster.Tags
				scCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(cluster.ModifiedDate/1000), 0))
				scMetaPatchCtx, scMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer scMetaPatchCancel()
				err := r.Patch(scMetaPatchCtx, scMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch scmeta.Spec.ID} %s", err), "")
					return u.patchServerClusterStatus(scCR, "Failure", err.Error())
				}
				debugLogger.Info("ServerCluster ID patched from existing Netris resource")
				if scMeta.Spec.Imported {
					logger.Info("ServerCluster imported")
				} else {
					logger.Info("ServerCluster found in Netris and linked")
				}
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
		debugLogger.Info("ServerCluster not found in Netris, will create new one")

		logger.Info("Creating ServerCluster")
		if _, err, errMsg := r.createServerCluster(scMeta); err != nil {
			logger.Error(fmt.Errorf("{createServerCluster} %s", err), "")
			return u.patchServerClusterStatus(scCR, "Failure", errMsg.Error())
		}
		logger.Info("ServerCluster Created")
	} else {
		apiSC, err := r.Cred.ServerCluster().GetByID(scMeta.Spec.ID)
		if err != nil || apiSC == nil {
			debugLogger.Info("ServerCluster not found in Netris")
			debugLogger.Info("Going to create ServerCluster")
			logger.Info("Creating ServerCluster")
			if _, err, errMsg := r.createServerCluster(scMeta); err != nil {
				logger.Error(fmt.Errorf("{createServerCluster} %s", err), "")
				return u.patchServerClusterStatus(scCR, "Failure", errMsg.Error())
			}
			logger.Info("ServerCluster Created")
		} else {
			provisionState = "Active"
			scCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(apiSC.ModifiedDate/1000), 0))
			debugLogger.Info("Comparing ServerClusterMeta with Netris ServerCluster")
			if ok := compareServerClusterMetaAPIServerCluster(scMeta, apiSC); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Something changed")
				debugLogger.Info("Go to update ServerCluster in Netris")
				logger.Info("Updating ServerCluster")
				updateSC, err := ServerClusterMetaToNetrisUpdate(scMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{ServerClusterMetaToNetrisUpdate} %s", err), "")
					return u.patchServerClusterStatus(scCR, "Failure", err.Error())
				}
				_, err, errMsg := r.updateServerCluster(scMeta.Spec.ID, updateSC)
				if err != nil {
					logger.Error(fmt.Errorf("{updateServerCluster} %s", err), "")
					return u.patchServerClusterStatus(scCR, "Failure", errMsg.Error())
				}
				logger.Info("ServerCluster Updated")
			}
		}
	}
	return u.patchServerClusterStatus(scCR, provisionState, "Success")
}

// SetupWithManager .
func (r *ServerClusterMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.ServerClusterMeta{}).
		Complete(r)
}

func (r *ServerClusterMetaReconciler) createServerCluster(scMeta *k8sv1alpha1.ServerClusterMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", scMeta.Namespace, scMeta.Spec.ServerClusterName),
		"scName", scMeta.Spec.ServerClusterName,
	).V(int(zapcore.WarnLevel))

	scAdd, err := ServerClusterMetaToNetris(scMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	reply, err := r.Cred.ServerCluster().Add(scAdd)
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

	debugLogger.Info("ServerCluster Created", "id", idStruct.ID)

	scMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, scMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func (r *ServerClusterMetaReconciler) updateServerCluster(id int, sc *servercluster.ServerClusterU) (ctrl.Result, error, error) {
	reply, err := r.Cred.ServerCluster().Update(id, sc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateServerCluster} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateServerCluster} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

