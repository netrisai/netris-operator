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
)

// VPCReconciler reconciles a VPC object
type VPCReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vpcs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vpcs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vpcs/finalizers,verbs=update

// Reconcile vpc events
func (r *VPCReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	vpcCR := &k8sv1alpha1.VPC{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	vpcCtx, vpcCancel := context.WithTimeout(cntxt, contextTimeout)
	defer vpcCancel()
	if err := r.Get(vpcCtx, req.NamespacedName, vpcCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	vpcMetaNamespaced := req.NamespacedName
	vpcMetaNamespaced.Name = string(vpcCR.GetUID())
	vpcMeta := &k8sv1alpha1.VPCMeta{}
	metaFound := true
	vpcMetaCtx, vpcMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer vpcMetaCancel()
	if err := r.Get(vpcMetaCtx, vpcMetaNamespaced, vpcMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			vpcMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if vpcCR.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteVPC(vpcCR, vpcMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteVPC} %s", err), "")
			return u.patchVPCStatus(vpcCR, "Failure", err.Error())
		}
		logger.Info("VPC deleted")
		return ctrl.Result{}, nil
	}

	if vpcMustUpdateAnnotations(vpcCR) {
		debugLogger.Info("Setting default annotations")
		vpcUpdateDefaultAnnotations(vpcCR)
		vpcUpdateCtx, vpcUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer vpcUpdateCancel()
		err := r.Patch(vpcUpdateCtx, vpcCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch VPC default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if vpcCompareFieldsForNewMeta(vpcCR, vpcMeta) {
			debugLogger.Info("Generating New Meta")
			vpcID := vpcMeta.Spec.ID
			newVpcMeta, err := r.VPCToVPCMeta(vpcCR)
			if err != nil {
				logger.Error(fmt.Errorf("{VPCToVPCMeta} %s", err), "")
				return u.patchVPCStatus(vpcCR, "Failure", err.Error())
			}
			vpcMeta.Spec = newVpcMeta.DeepCopy().Spec
			vpcMeta.Spec.ID = vpcID
			vpcMeta.Spec.VPCCRGeneration = vpcCR.GetGeneration()

			vpcMetaUpdateCtx, vpcMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer vpcMetaUpdateCancel()
			err = r.Update(vpcMetaUpdateCtx, vpcMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{vpcMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if vpcCR.GetFinalizers() == nil {
			vpcCR.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})
			vpcPatchCtx, vpcPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer vpcPatchCancel()
			err := r.Patch(vpcPatchCtx, vpcCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch VPC Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		vpcMeta, err := r.VPCToVPCMeta(vpcCR)
		if err != nil {
			logger.Error(fmt.Errorf("{VPCToVPCMeta} %s", err), "")
			return u.patchVPCStatus(vpcCR, "Failure", err.Error())
		}

		vpcMeta.Spec.VPCCRGeneration = vpcCR.GetGeneration()

		vpcMetaCreateCtx, vpcMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer vpcMetaCreateCancel()
		if err := r.Create(vpcMetaCreateCtx, vpcMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{vpcMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *VPCReconciler) deleteVPC(vpcCR *k8sv1alpha1.VPC, vpcMeta *k8sv1alpha1.VPCMeta) (ctrl.Result, error) {
	if vpcMeta != nil && vpcMeta.Spec.ID > 0 && !vpcMeta.Spec.Reclaim {
		reply, err := r.Cred.VPC().Delete(vpcMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteVPC} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess {
			if resp.Message != "Invalid VPC ID" {
				return ctrl.Result{}, fmt.Errorf("{deleteVPC} %s", fmt.Errorf(resp.Message))
			}
		}
	}
	return r.deleteCRs(vpcCR, vpcMeta)
}

func (r *VPCReconciler) deleteCRs(vpcCR *k8sv1alpha1.VPC, vpcMeta *k8sv1alpha1.VPCMeta) (ctrl.Result, error) {
	if vpcMeta != nil {
		_, err := r.deleteVPCMetaCR(vpcMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteVPCCR(vpcCR)
}

func (r *VPCReconciler) deleteVPCCR(vpcCR *k8sv1alpha1.VPC) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	vpcCR.ObjectMeta.SetFinalizers(nil)
	vpcCR.SetFinalizers(nil)
	if err := r.Update(ctx, vpcCR.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteVPCCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *VPCReconciler) deleteVPCMetaCR(vpcMeta *k8sv1alpha1.VPCMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, vpcMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteVPCMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager Resources
func (r *VPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VPC{}).
		Complete(r)
}

