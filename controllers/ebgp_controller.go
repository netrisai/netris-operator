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

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
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

	_ = uniReconciler{
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
		// Delete Logic
		logger.Info("EBGP deleted")
		return ctrl.Result{}, nil
	}

	if metaFound {
	} else {
	}

	return ctrl.Result{}, nil
}

func (r *EBGPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.EBGP{}).
		Complete(r)
}
