/*
Copyright 2020.

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
	"strconv"
	"time"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	api "github.com/netrisai/netrisapi"
)

// VNetMetaReconciler reconciles a VNetMeta object
type VNetMetaReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnetmeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnetmeta/status,verbs=get;update;patch

// Reconcile .
func (r *VNetMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))

	vnetMeta := &k8sv1alpha1.VNetMeta{}
	if err := r.Get(context.Background(), req.NamespacedName, vnetMeta); err != nil {
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
			if vnet, ok := NStorage.VNetStorage.findByName(vnetMeta.Spec.VnetName); ok {
				debugLogger.Info("Imported yaml mode. Vnet found")
				vnetID, err := strconv.Atoi(vnet.ID)
				if err != nil {
					debugLogger.Info(err.Error())
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
				vnetMeta.Spec.ID = vnetID
				err = r.Patch(context.Background(), vnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch vnetmeta.Spec.ID} %s", err), "")
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("VNet imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			debugLogger.Info("Imported yaml mode. VNet not found")
		}

		logger.Info("Creating VNet")
		if _, err := r.createVNet(vnetMeta); err != nil {
			logger.Error(fmt.Errorf("{createVNet} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		logger.Info("VNet Created")
	} else {
		vnets, err := Cred.GetVNetsByID(vnetMeta.Spec.ID)
		if err != nil {
			logger.Error(fmt.Errorf("{GetVNetsByID} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		if len(vnets) == 0 {
			debugLogger.Info("VNet not found in Netris")
			debugLogger.Info("Going to create VNet")
			logger.Info("Creating VNet")
			if _, err := r.createVNet(vnetMeta); err != nil {
				logger.Error(fmt.Errorf("{createVNet} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("VNet Created")
		} else {
			apiVnet := vnets[0]
			debugLogger.Info("Comparing VnetMeta with Netris Vnet")
			if ok := compareVNetMetaAPIVnet(vnetMeta, apiVnet); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Something changed")
				debugLogger.Info("Go to update Vnet in Netris")
				logger.Info("Updating VNet")
				vnetMeta.Spec.State = apiVnet.State
				updateVnet, err := VnetMetaToNetrisUpdate(vnetMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{VnetMetaToNetrisUpdate} %s", err), "")
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
				_, err = updateVNet(updateVnet)
				if err != nil {
					logger.Error(fmt.Errorf("{updateVNet} %s", err), "")
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
				logger.Info("VNet Updated")
			}
		}
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager .
func (r *VNetMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VNetMeta{}).
		Complete(r)
}

func (r *VNetMetaReconciler) createVNet(vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", vnetMeta.Namespace, vnetMeta.Name),
		"vnetName", vnetMeta.Spec.VnetName,
	).V(int(zapcore.WarnLevel))

	vnetAdd, err := VnetMetaToNetris(vnetMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	reply, err := Cred.AddVNet(vnetAdd)
	if err != nil {
		return ctrl.Result{}, err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf(resp.Message)
	}

	idStruct := api.APIVNetAddReply{}
	api.CustomDecode(resp.Data, &idStruct)

	debugLogger.Info("VNet Created", "id", idStruct.CircuitID)

	vnetMeta.Spec.ID = idStruct.CircuitID

	err = r.Patch(context.Background(), vnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.CircuitID)
	return ctrl.Result{}, nil
}
