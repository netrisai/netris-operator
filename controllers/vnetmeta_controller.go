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
)

// VNetMetaReconciler reconciles a VNetMeta object
type VNetMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnetmeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnetmeta/status,verbs=get;update;patch

// Reconcile .
func (r *VNetMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	vnetMeta := &k8sv1alpha1.VNetMeta{}
	vnetCR := &k8sv1alpha1.VNet{}
	vnetMetaCtx, vnetMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer vnetMetaCancel()
	if err := r.Get(vnetMetaCtx, req.NamespacedName, vnetMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, vnetMeta.Spec.VnetName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "Provisioning"

	vnetNN := req.NamespacedName
	vnetNN.Name = vnetMeta.Spec.VnetName
	vnetNNCtx, vnetNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer vnetNNCancel()
	if err := r.Get(vnetNNCtx, vnetNN, vnetCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if vnetMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if vnetMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if vnetMeta.Spec.Imported {
			logger.Info("Importing vnet")
			debugLogger.Info("Imported yaml mode. Finding VNet by name")
			if vnet, ok := r.NStorage.VNetStorage.FindByName(vnetMeta.Spec.VnetName); ok {
				debugLogger.Info("Imported yaml mode. Vnet found")
				vnetMeta.Spec.ID = vnet.ID
				vnetCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(vnet.ModifiedDate/1000), 0))
				vnetMetaPatchCtx, vnetMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer vnetMetaPatchCancel()
				err := r.Patch(vnetMetaPatchCtx, vnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch vnetmeta.Spec.ID} %s", err), "")
					return u.patchVNetStatus(vnetCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("VNet imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("VNet not found for import")
			debugLogger.Info("Imported yaml mode. VNet not found")
		}

		logger.Info("Creating VNet")
		if _, err, errMsg := r.createVNet(vnetMeta); err != nil {
			logger.Error(fmt.Errorf("{createVNet} %s", err), "")
			return u.patchVNetStatus(vnetCR, "Failure", errMsg.Error())
		}
		logger.Info("VNet Created")
	} else {
		vnet, err := r.Cred.VNet().GetByID(vnetMeta.Spec.ID)
		if err != nil {
			logger.Error(fmt.Errorf("{GetVNetsByID} %s", err), "")
			return u.patchVNetStatus(vnetCR, "Failure", err.Error())
		}
		if vnet == nil {
			debugLogger.Info("VNet not found in Netris")
			debugLogger.Info("Going to create VNet")
			logger.Info("Creating VNet")
			if _, err, errMsg := r.createVNet(vnetMeta); err != nil {
				logger.Error(fmt.Errorf("{createVNet} %s", err), "")
				return u.patchVNetStatus(vnetCR, "Failure", errMsg.Error())
			}
			logger.Info("VNet Created")
		} else {
			if !vnet.Provisioning {
				provisionState = "Active"
			}
			if vnet.State == "disabled" {
				provisionState = "Disabled"
			}
			vnetCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(vnet.ModifiedDate/1000), 0))
			debugLogger.Info("Comparing VnetMeta with Netris Vnet")
			if ok := compareVNetMetaAPIVnet(vnetMeta, vnet); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Something changed")
				debugLogger.Info("Go to update Vnet in Netris")
				logger.Info("Updating VNet")
				updateVnet, err := VnetMetaToNetrisUpdate(vnetMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{VnetMetaToNetrisUpdate} %s", err), "")
					return u.patchVNetStatus(vnetCR, "Failure", err.Error())
				}
				_, err, errMsg := r.updateVNet(vnetMeta.Spec.ID, updateVnet)
				if err != nil {
					logger.Error(fmt.Errorf("{updateVNet} %s", err), "")
					return u.patchVNetStatus(vnetCR, "Failure", errMsg.Error())
				}
				logger.Info("VNet Updated")
			}
		}
	}
	return u.patchVNetStatus(vnetCR, provisionState, "Success")
}

// SetupWithManager .
func (r *VNetMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VNetMeta{}).
		Complete(r)
}

func (r *VNetMetaReconciler) createVNet(vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", vnetMeta.Namespace, vnetMeta.Spec.VnetName),
		"vnetName", vnetMeta.Spec.VnetName,
	).V(int(zapcore.WarnLevel))

	vnetAdd, err := r.VnetMetaToNetris(vnetMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	reply, err := r.Cred.VNet().Add(vnetAdd)
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

	debugLogger.Info("VNet Created", "id", idStruct.ID)

	vnetMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, vnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}
