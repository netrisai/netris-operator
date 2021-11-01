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
	"github.com/netrisai/netriswebapi/v2/types/vnet"
)

// VNetReconciler reconciles a VNet object
type VNetReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnets/status,verbs=get;update;patch

// Reconcile vnet events
func (r *VNetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	vnet := &k8sv1alpha1.VNet{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	vnetCtx, vnetCancel := context.WithTimeout(cntxt, contextTimeout)
	defer vnetCancel()
	if err := r.Get(vnetCtx, req.NamespacedName, vnet); err != nil {
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
	vnetMetaCtx, vnetMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer vnetMetaCancel()
	if err := r.Get(vnetMetaCtx, vnetMetaNamespaced, vnetMeta); err != nil {
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

	if vnetMustUpdateAnnotations(vnet) {
		debugLogger.Info("Setting default annotations")
		vnetUpdateDefaultAnnotations(vnet)
		vnetUpdateCtx, vnetUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer vnetUpdateCancel()
		err := r.Patch(vnetUpdateCtx, vnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch VNet default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
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
		if vnetCompareFieldsForNewMeta(vnet, vnetMeta) {
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

			vnetMetaUpdateCtx, vnetMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer vnetMetaUpdateCancel()
			err = r.Update(vnetMetaUpdateCtx, vnetMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{vnetMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if vnet.GetFinalizers() == nil {
			vnet.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})
			vnetPatchCtx, vnetPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer vnetPatchCancel()
			err := r.Patch(vnetPatchCtx, vnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
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

		vnetMetaCreateCtx, vnetMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer vnetMetaCreateCancel()
		if err := r.Create(vnetMetaCreateCtx, vnetMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{vnetMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *VNetMetaReconciler) updateVNet(id int, vnet *vnet.VNetUpdate) (ctrl.Result, error, error) {
	reply, err := r.Cred.VNet().Update(id, vnet)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateVNet} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateVNet} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

func (r *VNetReconciler) deleteVNet(vnet *k8sv1alpha1.VNet, vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	if vnetMeta != nil && vnetMeta.Spec.ID > 0 && !vnetMeta.Spec.Reclaim {
		reply, err := r.Cred.VNet().Delete(vnetMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteVNet} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
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
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	vnet.ObjectMeta.SetFinalizers(nil)
	vnet.SetFinalizers(nil)
	if err := r.Update(ctx, vnet.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteVnetCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *VNetReconciler) deleteVnetMetaCR(vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, vnetMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
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
