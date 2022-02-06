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

// SoftgateToSoftgateMeta converts the Softgate resource to SoftgateMeta type and used for add the Softgate for Netris API.
func (r *SoftgateReconciler) SoftgateToSoftgateMeta(softgate *k8sv1alpha1.Softgate) (*k8sv1alpha1.SoftgateMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := softgate.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := softgate.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	siteID := 0
	if site, ok := r.NStorage.SitesStorage.FindByName(softgate.Spec.Site); ok {
		siteID = site.ID
	} else {
		return nil, fmt.Errorf("Invalid site '%s'", softgate.Spec.Site)
	}

	tenantID := 0
	if tenant, ok := r.NStorage.TenantsStorage.FindByName(softgate.Spec.Tenant); ok {
		tenantID = tenant.ID
	} else {
		return nil, fmt.Errorf("Invalid tenant '%s'", softgate.Spec.Tenant)
	}

	profileID := 0
	profiles, err := r.Cred.InventoryProfile().Get()
	if err != nil {
		return nil, err
	}

	for _, p := range profiles {
		if p.Name == softgate.Spec.Profile {
			profileID = p.ID
		}
	}

	if profileID == 0 {
		return nil, fmt.Errorf("Invalid profile '%s'", softgate.Spec.Profile)
	}

	softgateMeta := &k8sv1alpha1.SoftgateMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(softgate.GetUID()),
			Namespace: softgate.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.SoftgateMetaSpec{
			Imported:     imported,
			Reclaim:      reclaim,
			SoftgateName: softgate.Name,
			Description:  softgate.Spec.Description,
			TenantID:     tenantID,
			SiteID:       siteID,
			ProfileID:    profileID,
			MainIP:       softgate.Spec.MainIP,
			MgmtIP:       softgate.Spec.MgmtIP,
		},
	}

	return softgateMeta, nil
}

func softgateCompareFieldsForNewMeta(softgate *k8sv1alpha1.Softgate, softgateMeta *k8sv1alpha1.SoftgateMeta) bool {
	imported := false
	reclaim := false
	if i, ok := softgate.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := softgate.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return softgate.GetGeneration() != softgateMeta.Spec.SoftgateCRGeneration || imported != softgateMeta.Spec.Imported || reclaim != softgateMeta.Spec.Reclaim
}

func softgateMustUpdateAnnotations(softgate *k8sv1alpha1.Softgate) bool {
	update := false
	if i, ok := softgate.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := softgate.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func softgateUpdateDefaultAnnotations(softgate *k8sv1alpha1.Softgate) {
	imported := "false"
	reclaim := "delete"
	if i, ok := softgate.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := softgate.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := softgate.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	softgate.SetAnnotations(annotations)
}

// SoftgateMetaToNetris converts the k8s Softgate resource to Netris type and used for add the Softgate for Netris API.
func SoftgateMetaToNetris(softgateMeta *k8sv1alpha1.SoftgateMeta) (*inventory.HWSoftgate, error) {
	mainIP := softgateMeta.Spec.MainIP
	if softgateMeta.Spec.MainIP == "" {
		mainIP = "auto"
	}

	mgmtIP := softgateMeta.Spec.MgmtIP
	if softgateMeta.Spec.MgmtIP == "" {
		mgmtIP = "auto"
	}

	softgateAdd := &inventory.HWSoftgate{
		Name:        softgateMeta.Spec.SoftgateName,
		Description: softgateMeta.Spec.Description,
		Tenant:      inventory.IDName{ID: softgateMeta.Spec.TenantID},
		Site:        inventory.IDName{ID: softgateMeta.Spec.SiteID},
		Profile:     inventory.IDName{ID: softgateMeta.Spec.ProfileID},
		MainAddress: mainIP,
		MgmtAddress: mgmtIP,
		Links:       []inventory.HWLink{},
	}

	return softgateAdd, nil
}

// SoftgateMetaToNetrisUpdate converts the k8s Softgate resource to Netris type and used for update the Softgate for Netris API.
func SoftgateMetaToNetrisUpdate(softgateMeta *k8sv1alpha1.SoftgateMeta) (*inventory.HWSoftgateUpdate, error) {
	mainIP := softgateMeta.Spec.MainIP
	if softgateMeta.Spec.MainIP == "" {
		mainIP = "auto"
	}

	mgmtIP := softgateMeta.Spec.MgmtIP
	if softgateMeta.Spec.MgmtIP == "" {
		mgmtIP = "auto"
	}

	softgateUpdate := &inventory.HWSoftgateUpdate{
		Name:        softgateMeta.Spec.SoftgateName,
		Description: softgateMeta.Spec.Description,
		Tenant:      inventory.IDName{ID: softgateMeta.Spec.TenantID},
		Site:        inventory.IDName{ID: softgateMeta.Spec.SiteID},
		Profile:     inventory.IDName{ID: softgateMeta.Spec.ProfileID},
		MainAddress: mainIP,
		MgmtAddress: mgmtIP,
		Links:       []inventory.HWLink{},
	}

	return softgateUpdate, nil
}

func compareSoftgateMetaAPIESoftgate(softgateMeta *k8sv1alpha1.SoftgateMeta, apiSoftgate *inventory.HW, u uniReconciler) bool {
	if apiSoftgate.Name != softgateMeta.Spec.SoftgateName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiSoftgate.Name, "k8sValue", softgateMeta.Spec.SoftgateName)
		return false
	}

	if apiSoftgate.Description != softgateMeta.Spec.Description {
		u.DebugLogger.Info("Description changed", "netrisValue", apiSoftgate.Description, "k8sValue", softgateMeta.Spec.Description)
		return false
	}

	if apiSoftgate.Tenant.ID != softgateMeta.Spec.TenantID {
		u.DebugLogger.Info("Tenant changed", "netrisValue", apiSoftgate.Tenant.ID, "k8sValue", softgateMeta.Spec.TenantID)
		return false
	}

	if apiSoftgate.Site.ID != softgateMeta.Spec.SiteID {
		u.DebugLogger.Info("Site changed", "netrisValue", apiSoftgate.Site.ID, "k8sValue", softgateMeta.Spec.SiteID)
		return false
	}

	if apiSoftgate.Profile.ID != softgateMeta.Spec.ProfileID {
		u.DebugLogger.Info("Profile changed", "netrisValue", apiSoftgate.Profile.ID, "k8sValue", softgateMeta.Spec.ProfileID)
		return false
	}

	if apiSoftgate.MainIP.Address != softgateMeta.Spec.MainIP {
		u.DebugLogger.Info("MainIP changed", "netrisValue", apiSoftgate.MainIP.Address, "k8sValue", softgateMeta.Spec.MainIP)
		return false
	}

	if apiSoftgate.MgmtIP.Address != softgateMeta.Spec.MgmtIP {
		u.DebugLogger.Info("MgmtIP changed", "netrisValue", apiSoftgate.MgmtIP.Address, "k8sValue", softgateMeta.Spec.MgmtIP)
		return false
	}

	return true
}
