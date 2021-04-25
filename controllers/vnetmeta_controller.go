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
	"strconv"
	"time"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	vnetMeta := &k8sv1alpha1.VNetMeta{}
	vnetCR := &k8sv1alpha1.VNet{}
	if err := r.Get(context.Background(), req.NamespacedName, vnetMeta); err != nil {
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
	}

	provisionState := "Provisioning"

	vnetNN := req.NamespacedName
	vnetNN.Name = vnetMeta.Spec.VnetName
	if err := r.Get(context.Background(), vnetNN, vnetCR); err != nil {
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
				vnetCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(vnet.ModifiedDate/1000), 0))
				err = r.Patch(context.Background(), vnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
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
		vnets, err := Cred.GetVNetsByID(vnetMeta.Spec.ID)
		if err != nil {
			logger.Error(fmt.Errorf("{GetVNetsByID} %s", err), "")
			return u.patchVNetStatus(vnetCR, "Failure", err.Error())
		}
		if len(vnets) == 0 {
			debugLogger.Info("VNet not found in Netris")
			debugLogger.Info("Going to create VNet")
			logger.Info("Creating VNet")
			if _, err, errMsg := r.createVNet(vnetMeta); err != nil {
				logger.Error(fmt.Errorf("{createVNet} %s", err), "")
				return u.patchVNetStatus(vnetCR, "Failure", errMsg.Error())
			}
			logger.Info("VNet Created")
		} else {
			apiVnet := vnets[0]
			if apiVnet.Provisioning == 0 {
				provisionState = "Active"
			}
			if apiVnet.State == "disabled" {
				provisionState = "Disabled"
			}
			vnetCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(apiVnet.ModifiedDate/1000), 0))
			debugLogger.Info("Comparing VnetMeta with Netris Vnet")
			if ok := compareVNetMetaAPIVnet(vnetMeta, apiVnet); ok {
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
				_, err, errMsg := updateVNet(updateVnet)
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

	vnetAdd, err := VnetMetaToNetris(vnetMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	reply, err := Cred.AddVNet(vnetAdd)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf(resp.Message), fmt.Errorf(resp.Message)
	}

	idStruct := api.APIVNetAddReply{}
	err = api.CustomDecode(resp.Data, &idStruct)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("VNet Created", "id", idStruct.CircuitID)

	vnetMeta.Spec.ID = idStruct.CircuitID

	err = r.Patch(context.Background(), vnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.CircuitID)
	return ctrl.Result{}, nil, nil
}
