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
	"fmt"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netriswebapi/v2/types/inventory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ControllerToControllerMeta converts the Controller resource to ControllerMeta type and used for add the Controller for Netris API.
func (r *ControllerReconciler) ControllerToControllerMeta(controller *k8sv1alpha1.Controller) (*k8sv1alpha1.ControllerMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := controller.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := controller.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	siteID := 0
	if site, ok := r.NStorage.SitesStorage.FindByName(controller.Spec.Site); ok {
		siteID = site.ID
	} else {
		return nil, fmt.Errorf("invalid site '%s'", controller.Spec.Site)
	}

	tenantID := 0
	if tenant, ok := r.NStorage.TenantsStorage.FindByName(controller.Spec.Tenant); ok {
		tenantID = tenant.ID
	} else {
		return nil, fmt.Errorf("invalid tenant '%s'", controller.Spec.Tenant)
	}

	controllerMeta := &k8sv1alpha1.ControllerMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(controller.GetUID()),
			Namespace: controller.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.ControllerMetaSpec{
			Imported:       imported,
			Reclaim:        reclaim,
			ControllerName: controller.Name,
			Description:    controller.Spec.Description,
			TenantID:       tenantID,
			SiteID:         siteID,
			MainIP:         controller.Spec.MainIP,
		},
	}

	return controllerMeta, nil
}

func controllerCompareFieldsForNewMeta(controller *k8sv1alpha1.Controller, controllerMeta *k8sv1alpha1.ControllerMeta) bool {
	imported := false
	reclaim := false
	if i, ok := controller.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := controller.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return controller.GetGeneration() != controllerMeta.Spec.ControllerCRGeneration || imported != controllerMeta.Spec.Imported || reclaim != controllerMeta.Spec.Reclaim
}

func controllerMustUpdateAnnotations(controller *k8sv1alpha1.Controller) bool {
	update := false
	if i, ok := controller.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := controller.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func controllerUpdateDefaultAnnotations(controller *k8sv1alpha1.Controller) {
	imported := "false"
	reclaim := "delete"
	if i, ok := controller.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := controller.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := controller.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	controller.SetAnnotations(annotations)
}

// ControllerMetaToNetris converts the k8s Controller resource to Netris type and used for add the Controller for Netris API.
func ControllerMetaToNetris(controllerMeta *k8sv1alpha1.ControllerMeta) (*inventory.HWController, error) {
	mainIP := controllerMeta.Spec.MainIP
	if controllerMeta.Spec.MainIP == "" {
		mainIP = "auto"
	}

	controllerAdd := &inventory.HWController{
		Name:        controllerMeta.Spec.ControllerName,
		Description: controllerMeta.Spec.Description,
		Tenant:      inventory.IDName{ID: controllerMeta.Spec.TenantID},
		Site:        inventory.IDName{ID: controllerMeta.Spec.SiteID},
		MainAddress: mainIP,
	}

	return controllerAdd, nil
}

// ControllerMetaToNetrisUpdate converts the k8s Controller resource to Netris type and used for update the Controller for Netris API.
func ControllerMetaToNetrisUpdate(controllerMeta *k8sv1alpha1.ControllerMeta) (*inventory.HWControllerUpdate, error) {
	mainIP := controllerMeta.Spec.MainIP
	if controllerMeta.Spec.MainIP == "" {
		mainIP = "auto"
	}

	controllerUpdate := &inventory.HWControllerUpdate{
		Name:        controllerMeta.Spec.ControllerName,
		Description: controllerMeta.Spec.Description,
		MainAddress: mainIP,
	}

	return controllerUpdate, nil
}

func compareControllerMetaAPIEController(controllerMeta *k8sv1alpha1.ControllerMeta, apiController *inventory.HW, u uniReconciler) bool {
	if apiController.Name != controllerMeta.Spec.ControllerName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiController.Name, "k8sValue", controllerMeta.Spec.ControllerName)
		return false
	}

	if apiController.Description != controllerMeta.Spec.Description {
		u.DebugLogger.Info("Description changed", "netrisValue", apiController.Description, "k8sValue", controllerMeta.Spec.Description)
		return false
	}

	if apiController.MainIP.Address != controllerMeta.Spec.MainIP {
		u.DebugLogger.Info("MainIP changed", "netrisValue", apiController.MainIP.Address, "k8sValue", controllerMeta.Spec.MainIP)
		return false
	}

	return true
}
