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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/netrisai/netris-operator/api/v1alpha1"
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
)

type uniReconciler struct {
	client.Client
	Logger      logr.Logger
	DebugLogger logr.InfoLogger
}

func (u *uniReconciler) patchVNetStatus(vnet *k8sv1alpha1.VNet, status, message string) (ctrl.Result, error) {

	u.DebugLogger.Info("Patching Status", "status", status, "message", message)
	vnet.Status.Status = status
	vnet.Status.Message = message
	err := u.Status().Patch(context.Background(), vnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.Logger.Error(fmt.Errorf("{r.Status().Patch} %s", err), "")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) updateVNetStatus(vnet *k8sv1alpha1.VNet, status, message string) (ctrl.Result, error) {

	u.DebugLogger.Info("Updating Status", "status", status, "message", message)
	state := "active"
	if len(vnet.Spec.State) > 0 {
		state = vnet.Spec.State
	}
	vnet.Status = v1alpha1.VNetStatus{
		Status:  status,
		Message: message,
		State:   state,
	}
	err := u.Status().Update(context.Background(), vnet.DeepCopyObject(), &client.UpdateOptions{})
	if err != nil {
		u.Logger.Error(fmt.Errorf("{r.Status().Update} %s", err), "")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}
