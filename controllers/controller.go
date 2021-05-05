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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
)

type uniReconciler struct {
	client.Client
	Logger      logr.Logger
	DebugLogger logr.InfoLogger
}

func (u *uniReconciler) patchVNetStatus(vnet *k8sv1alpha1.VNet, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)
	state := "active"
	if len(vnet.Spec.State) > 0 {
		state = vnet.Spec.State
	}
	vnet.Status.Status = status
	vnet.Status.Message = message
	vnet.Status.State = state
	vnet.Status.Gateways = vnet.GatewaysString()
	vnet.Status.Sites = vnet.SitesString()

	err := u.Status().Patch(context.Background(), vnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchEBGPStatus(ebgp *k8sv1alpha1.EBGP, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	ebgp.Status.Status = status
	ebgp.Status.Message = message

	err := u.Status().Patch(context.Background(), ebgp.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchL4LBStatus(l4lb *k8sv1alpha1.L4LB, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	state := "active"
	if len(l4lb.Spec.State) > 0 {
		state = l4lb.Spec.State
	}

	l4lb.Status.Status = status
	l4lb.Status.State = state
	l4lb.Status.Message = message

	err := u.Status().Patch(context.Background(), l4lb.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchL4LB(l4lb *k8sv1alpha1.L4LB) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching")

	err := u.Patch(context.Background(), l4lb.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Patch()}", "error", err)
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}
