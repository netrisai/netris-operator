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
	api "github.com/netrisai/netrisapi"
)

// VNetReconciler reconciles a VNet object
type VNetReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnets/status,verbs=get;update;patch

// Reconcile vnet events
func (r *VNetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	vnet := &k8sv1alpha1.VNet{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
	}

	if err := r.Get(context.Background(), req.NamespacedName, vnet); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	vnetMetaNamespaced := req.NamespacedName
	vnetMetaNamespaced.Name = string(vnet.GetUID())
	vnetMeta := &k8sv1alpha1.VNetMeta{}
	metaFound := true
	if err := r.Get(context.Background(), vnetMetaNamespaced, vnetMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			vnetMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if vnet.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteVNet(vnet, vnetMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteVNet} %s", err), "")
			return u.patchVNetStatus(vnet, "Failure", err.Error())
		}
		logger.Info("Vnet deleted")
		return ctrl.Result{}, nil
	}

	for _, s := range vnet.Spec.Sites {
		if dup, found := findGatewayDuplicates(s.Gateways); found {
			errMsg := fmt.Sprintf("Found duplicate value '%s' in '%s' site gateways", dup, s.Name)
			return u.patchVNetStatus(vnet, "Failure", errMsg)
		}
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if vnet.GetGeneration() != vnetMeta.Spec.VnetCRGeneration {
			debugLogger.Info("Generating New Meta")
			vnetID := vnetMeta.Spec.ID
			newVnetMeta, err := r.VnetToVnetMeta(vnet)
			if err != nil {
				logger.Error(fmt.Errorf("{VnetToVnetMeta} %s", err), "")
				return u.patchVNetStatus(vnet, "Failure", err.Error())
			}
			vnetMeta.Spec = newVnetMeta.DeepCopy().Spec
			vnetMeta.Spec.ID = vnetID
			vnetMeta.Spec.VnetCRGeneration = vnet.GetGeneration()

			err = r.Update(context.Background(), vnetMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{vnetMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if vnet.GetFinalizers() == nil {
			vnet.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})
			err := r.Patch(context.Background(), vnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch VNet Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		vnetMeta, err := r.VnetToVnetMeta(vnet)
		if err != nil {
			logger.Error(fmt.Errorf("{VnetToVnetMeta} %s", err), "")
			return u.patchVNetStatus(vnet, "Failure", err.Error())
		}

		vnetMeta.Spec.VnetCRGeneration = vnet.GetGeneration()

		if err := r.Create(context.Background(), vnetMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{vnetMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func updateVNet(vnet *api.APIVNetUpdate) (ctrl.Result, error, error) {
	reply, err := Cred.ValidateVNet(vnet)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateVNet} %s", err), err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateVNet} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	reply, err = Cred.UpdateVNet(vnet)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateVNet} %s", err), err
	}
	resp, err = api.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateVNet} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

func (r *VNetReconciler) deleteVNet(vnet *k8sv1alpha1.VNet, vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	if vnetMeta != nil && vnetMeta.Spec.ID > 0 {
		reply, err := Cred.DeleteVNet(vnetMeta.Spec.ID, []int{1})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteVNet} %s", err)
		}
		resp, err := api.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess {
			if resp.Message != "Invalid circuit ID" {
				return ctrl.Result{}, fmt.Errorf("{deleteVNet} %s", fmt.Errorf(resp.Message))
			}
		}
	}
	return r.deleteCRs(vnet, vnetMeta)
}

func (r *VNetReconciler) deleteCRs(vnet *k8sv1alpha1.VNet, vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	if vnetMeta != nil {
		_, err := r.deleteVnetMetaCR(vnetMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteVnetCR(vnet)
}

func (r *VNetReconciler) deleteVnetCR(vnet *k8sv1alpha1.VNet) (ctrl.Result, error) {
	vnet.ObjectMeta.SetFinalizers(nil)
	vnet.SetFinalizers(nil)
	if err := r.Update(context.Background(), vnet.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteVnetCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *VNetReconciler) deleteVnetMetaCR(vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	if err := r.Delete(context.Background(), vnetMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteVnetMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager Resources
func (r *VNetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VNet{}).
		// WithEventFilter(ignoreDeletionPredicate()).
		Complete(r)
}
