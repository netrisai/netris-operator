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
	"github.com/netrisai/netriswebapi/v2/types/vpc"
)

// VPCMetaReconciler reconciles a VPCMeta object
type VPCMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vpcmeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vpcmeta/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vpcmeta/finalizers,verbs=update

// Reconcile .
func (r *VPCMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	vpcMeta := &k8sv1alpha1.VPCMeta{}
	vpcCR := &k8sv1alpha1.VPC{}
	vpcMetaCtx, vpcMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer vpcMetaCancel()
	if err := r.Get(vpcMetaCtx, req.NamespacedName, vpcMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, vpcMeta.Spec.VPCName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "Active"

	vpcNN := req.NamespacedName
	vpcNN.Name = vpcMeta.Spec.VPCName
	vpcNNCtx, vpcNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer vpcNNCancel()
	if err := r.Get(vpcNNCtx, vpcNN, vpcCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if vpcMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if vpcMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if vpcMeta.Spec.Imported {
			logger.Info("Importing vpc")
			debugLogger.Info("Imported yaml mode. Finding VPC by name")
			if vpc, ok := r.NStorage.VPCStorage.FindByName(vpcMeta.Spec.VPCName); ok {
				debugLogger.Info("Imported yaml mode. VPC found")
				vpcMeta.Spec.ID = vpc.ID
				vpcMeta.Spec.AdminTenantID = vpc.AdminTenant.ID
				vpcMeta.Spec.AdminTenant = vpc.AdminTenant.Name
				guestTenantIDs := []int{}
				guestTenantNames := []string{}
				for _, tenant := range vpc.GuestTenant {
					guestTenantIDs = append(guestTenantIDs, tenant.ID)
					guestTenantNames = append(guestTenantNames, tenant.Name)
				}
				vpcMeta.Spec.GuestTenantIDs = guestTenantIDs
				vpcMeta.Spec.GuestTenants = guestTenantNames
				vpcMeta.Spec.Tags = vpc.Tags
				vpcMeta.Spec.IsSystem = false // VPC API doesn't expose IsSystem in response
				vpcMeta.Spec.IsDefault = vpc.IsDefault
				vpcMeta.Spec.VNI = 0 // VPC API doesn't expose VNI in response
				vpcCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(vpc.ModifiedDate/1000), 0))
				vpcMetaPatchCtx, vpcMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer vpcMetaPatchCancel()
				err := r.Patch(vpcMetaPatchCtx, vpcMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch vpcmeta.Spec.ID} %s", err), "")
					return u.patchVPCStatus(vpcCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("VPC imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("VPC not found for import")
			debugLogger.Info("Imported yaml mode. VPC not found")
		}

		logger.Info("Creating VPC")
		if _, err, errMsg := r.createVPC(vpcMeta); err != nil {
			logger.Error(fmt.Errorf("{createVPC} %s", err), "")
			return u.patchVPCStatus(vpcCR, "Failure", errMsg.Error())
		}
		logger.Info("VPC Created")
	} else {
		apiVPC, _ := r.Cred.VPC().GetByID(vpcMeta.Spec.ID)
		if apiVPC == nil {
			debugLogger.Info("VPC not found in Netris")
			debugLogger.Info("Going to create VPC")
			logger.Info("Creating VPC")
			if _, err, errMsg := r.createVPC(vpcMeta); err != nil {
				logger.Error(fmt.Errorf("{createVPC} %s", err), "")
				return u.patchVPCStatus(vpcCR, "Failure", errMsg.Error())
			}
			logger.Info("VPC Created")
		} else {
			vpcCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(apiVPC.ModifiedDate/1000), 0))
			debugLogger.Info("Comparing VPCMeta with Netris VPC")
			if ok := compareVPCMetaAPIVPC(vpcMeta, apiVPC); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Something changed")
				debugLogger.Info("Go to update VPC in Netris")
				logger.Info("Updating VPC")
				updateVPC, err := VPCMetaToNetrisUpdate(vpcMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{VPCMetaToNetrisUpdate} %s", err), "")
					return u.patchVPCStatus(vpcCR, "Failure", err.Error())
				}
				_, err, errMsg := r.updateVPC(vpcMeta.Spec.ID, updateVPC)
				if err != nil {
					logger.Error(fmt.Errorf("{updateVPC} %s", err), "")
					return u.patchVPCStatus(vpcCR, "Failure", errMsg.Error())
				}
				logger.Info("VPC Updated")
			}
		}
	}
	return u.patchVPCStatus(vpcCR, provisionState, "Success")
}

// SetupWithManager .
func (r *VPCMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VPCMeta{}).
		Complete(r)
}

func (r *VPCMetaReconciler) createVPC(vpcMeta *k8sv1alpha1.VPCMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", vpcMeta.Namespace, vpcMeta.Spec.VPCName),
		"vpcName", vpcMeta.Spec.VPCName,
	).V(int(zapcore.WarnLevel))

	vpcAdd, err := r.VPCMetaToNetris(vpcMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	reply, err := r.Cred.VPC().Add(vpcAdd)
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

	debugLogger.Info("VPC Created", "id", idStruct.ID)

	vpcMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, vpcMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func (r *VPCMetaReconciler) updateVPC(id int, vpc *vpc.VPCw) (ctrl.Result, error, error) {
	reply, err := r.Cred.VPC().Update(id, vpc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateVPC} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateVPC} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

