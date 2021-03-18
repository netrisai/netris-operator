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
	api "github.com/netrisai/netrisapi"
)

// EBGPReconciler reconciles a EBGP object
type EBGPReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=ebgps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=ebgps/status,verbs=get;update;patch

func (r *EBGPReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	ebgp := &k8sv1alpha1.EBGP{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
	}

	if err := r.Get(context.Background(), req.NamespacedName, ebgp); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	ebgpMetaNamespaced := req.NamespacedName
	ebgpMetaNamespaced.Name = string(ebgp.GetUID())
	ebgpMeta := &k8sv1alpha1.EBGPMeta{}
	metaFound := true

	if err := r.Get(context.Background(), ebgpMetaNamespaced, ebgpMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			ebgpMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if ebgp.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteEBGP(ebgp, ebgpMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteEBGP} %s", err), "")
			return u.patchEBGPStatus(ebgp, "Failure", err.Error())
		}
		logger.Info("EBGP deleted")
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if ebgp.GetGeneration() != ebgpMeta.Spec.EBGPCRGeneration {
			debugLogger.Info("Generating New Meta")
			ebgpID := ebgpMeta.Spec.ID
			newVnetMeta, err := r.EBGPToEBGPMeta(ebgp)
			if err != nil {
				logger.Error(fmt.Errorf("{EBGPToEBGPMeta} %s", err), "")
				return u.patchEBGPStatus(ebgp, "Failure", err.Error())
			}
			ebgpMeta.Spec = newVnetMeta.DeepCopy().Spec
			ebgpMeta.Spec.ID = ebgpID
			ebgpMeta.Spec.EBGPCRGeneration = ebgp.GetGeneration()

			err = r.Update(context.Background(), ebgpMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{ebgpMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if ebgp.GetFinalizers() == nil {
			ebgp.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})
			err := r.Patch(context.Background(), ebgp.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch EBGP Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		ebgpMeta, err := r.EBGPToEBGPMeta(ebgp)
		if err != nil {
			logger.Error(fmt.Errorf("{EBGPToEBGPMeta} %s", err), "")
			return u.patchEBGPStatus(ebgp, "Failure", err.Error())
		}

		ebgpMeta.Spec.EBGPCRGeneration = ebgp.GetGeneration()

		if err := r.Create(context.Background(), ebgpMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{ebgpMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *EBGPReconciler) deleteEBGP(ebgp *k8sv1alpha1.EBGP, ebgpMeta *k8sv1alpha1.EBGPMeta) (ctrl.Result, error) {
	if ebgpMeta != nil && ebgpMeta.Spec.ID > 0 {
		reply, err := Cred.DeleteEBGP(ebgpMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteEBGP} %s", err)
		}
		resp, err := api.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess {
			return ctrl.Result{}, fmt.Errorf("{deleteEBGP} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(ebgp, ebgpMeta)
}

func (r *EBGPReconciler) deleteCRs(ebgp *k8sv1alpha1.EBGP, ebgpMeta *k8sv1alpha1.EBGPMeta) (ctrl.Result, error) {
	if ebgpMeta != nil {
		_, err := r.deleteEBGPMetaCR(ebgpMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteEBGPCR(ebgp)
}

func (r *EBGPReconciler) deleteEBGPCR(ebgp *k8sv1alpha1.EBGP) (ctrl.Result, error) {
	ebgp.ObjectMeta.SetFinalizers(nil)
	ebgp.SetFinalizers(nil)
	if err := r.Update(context.Background(), ebgp.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteEBGPCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *EBGPReconciler) deleteEBGPMetaCR(ebgpMeta *k8sv1alpha1.EBGPMeta) (ctrl.Result, error) {
	if err := r.Delete(context.Background(), ebgpMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteEBGPMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *EBGPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.EBGP{}).
		Complete(r)
}
