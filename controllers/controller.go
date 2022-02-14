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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	api "github.com/netrisai/netriswebapi/v2"
)

var (
	requeueInterval = time.Duration(10 * time.Second)
	cntxt           = context.Background()
	contextTimeout  = requeueInterval
)

type uniReconciler struct {
	client.Client
	Logger      logr.Logger
	DebugLogger logr.InfoLogger
	Cred        *api.Clientset
	NStorage    *netrisstorage.Storage
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

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, vnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchBGPStatus(bgp *k8sv1alpha1.BGP, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	state := "enabled"
	if len(bgp.Spec.State) > 0 {
		state = bgp.Spec.State
	}

	bgp.Status.Status = status
	bgp.Status.State = state
	bgp.Status.Message = message

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, bgp.DeepCopyObject(), client.Merge, &client.PatchOptions{})
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

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, l4lb.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchL4LB(l4lb *k8sv1alpha1.L4LB) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching")
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Patch(ctx, l4lb.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Patch()}", "error", err)
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchSiteStatus(l4lb *k8sv1alpha1.Site, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	l4lb.Status.Status = status
	l4lb.Status.Message = message

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, l4lb.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchAllocationStatus(allocation *k8sv1alpha1.Allocation, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	allocation.Status.Status = status
	allocation.Status.Message = message

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, allocation.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchSubnetStatus(subnet *k8sv1alpha1.Subnet, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	subnet.Status.Status = status
	subnet.Status.Message = message

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, subnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchSoftgateStatus(softgate *k8sv1alpha1.Softgate, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	softgate.Status.Status = status
	softgate.Status.Message = message

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, softgate.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchSwitchStatus(switchH *k8sv1alpha1.Switch, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	switchH.Status.Status = status
	switchH.Status.Message = message

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, switchH.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchControllerStatus(controller *k8sv1alpha1.Controller, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	controller.Status.Status = status
	controller.Status.Message = message

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, controller.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchLinkStatus(link *k8sv1alpha1.Link, status, message string) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching Status", "status", status, "message", message)

	link.Status.Status = status
	link.Status.Message = message
	link.Status.Ports = fmt.Sprintf("%s, %s", link.Spec.Ports[0], link.Spec.Ports[1])

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Status().Patch(ctx, link.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Status().Patch}", "error", err, "action", "status update")
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchSoftgate(softgate *k8sv1alpha1.Softgate) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching")
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Patch(ctx, softgate.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Patch()}", "error", err)
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchSwitch(switchH *k8sv1alpha1.Switch) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching")
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Patch(ctx, switchH.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Patch()}", "error", err)
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (u *uniReconciler) patchController(controller *k8sv1alpha1.Controller) (ctrl.Result, error) {
	u.DebugLogger.Info("Patching")
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err := u.Patch(ctx, controller.DeepCopyObject(), client.Merge, &client.PatchOptions{})
	if err != nil {
		u.DebugLogger.Info("{r.Patch()}", "error", err)
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}
