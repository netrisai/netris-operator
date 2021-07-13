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

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	api "github.com/netrisai/netrisapi"
)

// BGPReconciler reconciles a BGP object
type BGPReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.HTTPCred
	NStorage *netrisstorage.Storage
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=bgps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=bgps/status,verbs=get;update;patch

func (r *BGPReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	bgp := &k8sv1alpha1.BGP{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	bgpCtx, bgpCancel := context.WithTimeout(cntxt, contextTimeout)
	defer bgpCancel()
	if err := r.Get(bgpCtx, req.NamespacedName, bgp); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	bgpMetaNamespaced := req.NamespacedName
	bgpMetaNamespaced.Name = string(bgp.GetUID())
	bgpMeta := &k8sv1alpha1.BGPMeta{}
	metaFound := true

	bgpMetaCtx, bgpMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer bgpMetaCancel()
	if err := r.Get(bgpMetaCtx, bgpMetaNamespaced, bgpMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			bgpMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if bgp.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteBGP(bgp, bgpMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteBGP} %s", err), "")
			return u.patchBGPStatus(bgp, "Failure", err.Error())
		}
		logger.Info("BGP deleted")
		return ctrl.Result{}, nil
	}

	if bgpMustUpdateAnnotations(bgp) {
		debugLogger.Info("Setting default annotations")
		bgpUpdateDefaultAnnotations(bgp)
		bgpPatchCtx, bgpPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer bgpPatchCancel()
		err := r.Patch(bgpPatchCtx, bgp.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch BGP default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if bgpCompareFieldsForNewMeta(bgp, bgpMeta) {
			debugLogger.Info("Generating New Meta")
			bgpID := bgpMeta.Spec.ID
			newVnetMeta, err := r.BGPToBGPMeta(bgp)
			if err != nil {
				logger.Error(fmt.Errorf("{BGPToBGPMeta} %s", err), "")
				return u.patchBGPStatus(bgp, "Failure", err.Error())
			}
			bgpMeta.Spec = newVnetMeta.DeepCopy().Spec
			bgpMeta.Spec.ID = bgpID
			bgpMeta.Spec.BGPCRGeneration = bgp.GetGeneration()

			bgpMetaUpdateCtx, bgpMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer bgpMetaUpdateCancel()
			err = r.Update(bgpMetaUpdateCtx, bgpMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{bgpMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if bgp.GetFinalizers() == nil {
			bgp.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

			bgpPatchCtx, bgpPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer bgpPatchCancel()
			err := r.Patch(bgpPatchCtx, bgp.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch BGP Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		bgpMeta, err := r.BGPToBGPMeta(bgp)
		if err != nil {
			logger.Error(fmt.Errorf("{BGPToBGPMeta} %s", err), "")
			return u.patchBGPStatus(bgp, "Failure", err.Error())
		}

		bgpMeta.Spec.BGPCRGeneration = bgp.GetGeneration()

		bgpMetaCreateCtx, bgpMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer bgpMetaCreateCancel()
		if err := r.Create(bgpMetaCreateCtx, bgpMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{bgpMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *BGPReconciler) deleteBGP(bgp *k8sv1alpha1.BGP, bgpMeta *k8sv1alpha1.BGPMeta) (ctrl.Result, error) {
	if bgpMeta != nil && bgpMeta.Spec.ID > 0 && !bgpMeta.Spec.Reclaim {
		reply, err := r.Cred.DeleteEBGP(bgpMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteBGP} %s", err)
		}
		resp, err := api.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess {
			return ctrl.Result{}, fmt.Errorf("{deleteBGP} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(bgp, bgpMeta)
}

func (r *BGPReconciler) deleteCRs(bgp *k8sv1alpha1.BGP, bgpMeta *k8sv1alpha1.BGPMeta) (ctrl.Result, error) {
	if bgpMeta != nil {
		_, err := r.deleteBGPMetaCR(bgpMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteBGPCR(bgp)
}

func (r *BGPReconciler) deleteBGPCR(bgp *k8sv1alpha1.BGP) (ctrl.Result, error) {
	bgp.ObjectMeta.SetFinalizers(nil)
	bgp.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, bgp.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteBGPCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *BGPReconciler) deleteBGPMetaCR(bgpMeta *k8sv1alpha1.BGPMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, bgpMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteBGPMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *BGPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.BGP{}).
		Complete(r)
}
