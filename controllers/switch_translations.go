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

// SwitchToSwitchMeta converts the Switch resource to SwitchMeta type and used for add the Switch for Netris API.
func (r *SwitchReconciler) SwitchToSwitchMeta(switchH *k8sv1alpha1.Switch) (*k8sv1alpha1.SwitchMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	nosList, err := r.Cred.Inventory().GetNOS()
	if err != nil {
		return nil, err
	}

	nosMap := make(map[string]inventory.NOS)
	for _, nos := range nosList {
		nosMap[nos.Tag] = *nos
	}

	nos := nosMap[switchH.Spec.NOS]

	if i, ok := switchH.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := switchH.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	siteID := 0
	if site, ok := r.NStorage.SitesStorage.FindByName(switchH.Spec.Site); ok {
		siteID = site.ID
	} else {
		return nil, fmt.Errorf("invalid site '%s'", switchH.Spec.Site)
	}

	tenantID := 0
	if tenant, ok := r.NStorage.TenantsStorage.FindByName(switchH.Spec.Tenant); ok {
		tenantID = tenant.ID
	} else {
		return nil, fmt.Errorf("invalid tenant '%s'", switchH.Spec.Tenant)
	}

	profileID := 0
	profiles, err := r.Cred.InventoryProfile().Get()
	if err != nil {
		return nil, err
	}

	for _, p := range profiles {
		if p.Name == switchH.Spec.Profile {
			profileID = p.ID
		}
	}

	if profileID == 0 && switchH.Spec.Profile != "" {
		return nil, fmt.Errorf("invalid profile '%s'", switchH.Spec.Profile)
	}

	switchMeta := &k8sv1alpha1.SwitchMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(switchH.GetUID()),
			Namespace: switchH.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.SwitchMetaSpec{
			Imported:    imported,
			Reclaim:     reclaim,
			SwitchName:  switchH.Name,
			Description: switchH.Spec.Description,
			NOS:         nos,
			TenantID:    tenantID,
			SiteID:      siteID,
			ASN:         switchH.Spec.ASN,
			ProfileID:   profileID,
			MainIP:      switchH.Spec.MainIP,
			MgmtIP:      switchH.Spec.MgmtIP,
			PortsCount:  switchH.Spec.PortsCount,
			MacAddress:  switchH.Spec.MacAddress,
		},
	}

	return switchMeta, nil
}

func switchCompareFieldsForNewMeta(switchH *k8sv1alpha1.Switch, switchMeta *k8sv1alpha1.SwitchMeta) bool {
	imported := false
	reclaim := false
	if i, ok := switchH.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := switchH.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return switchH.GetGeneration() != switchMeta.Spec.SwitchCRGeneration || imported != switchMeta.Spec.Imported || reclaim != switchMeta.Spec.Reclaim
}

func switchMustUpdateAnnotations(switchH *k8sv1alpha1.Switch) bool {
	update := false
	if i, ok := switchH.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := switchH.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func switchUpdateDefaultAnnotations(switchH *k8sv1alpha1.Switch) {
	imported := "false"
	reclaim := "delete"
	if i, ok := switchH.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := switchH.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := switchH.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	switchH.SetAnnotations(annotations)
}

// SwitchMetaToNetris converts the k8s Switch resource to Netris type and used for add the Switch for Netris API.
func SwitchMetaToNetris(switchMeta *k8sv1alpha1.SwitchMeta) (*inventory.HWSwitchAdd, error) {
	mainIP := switchMeta.Spec.MainIP
	if switchMeta.Spec.MainIP == "" {
		mainIP = "auto"
	}

	mgmtIP := switchMeta.Spec.MgmtIP
	if switchMeta.Spec.MgmtIP == "" {
		mgmtIP = "auto"
	}

	var asn interface{} = switchMeta.Spec.ASN
	if switchMeta.Spec.ASN == 0 {
		asn = "auto"
	}

	switchAdd := &inventory.HWSwitchAdd{
		Name:        switchMeta.Spec.SwitchName,
		Description: switchMeta.Spec.Description,
		Tenant:      inventory.IDName{ID: switchMeta.Spec.TenantID},
		Nos:         switchMeta.Spec.NOS,
		Asn:         asn,
		Site:        inventory.IDName{ID: switchMeta.Spec.SiteID},
		Profile:     inventory.IDName{ID: switchMeta.Spec.ProfileID},
		MainAddress: mainIP,
		MgmtAddress: mgmtIP,
		PortCount:   switchMeta.Spec.PortsCount,
		MacAddress:  switchMeta.Spec.MacAddress,
		Links:       []inventory.HWLink{},
	}

	return switchAdd, nil
}

// SwitchMetaToNetrisUpdate converts the k8s Switch resource to Netris type and used for update the Switch for Netris API.
func SwitchMetaToNetrisUpdate(switchMeta *k8sv1alpha1.SwitchMeta) (*inventory.HWSwitchUpdate, error) {
	mainIP := switchMeta.Spec.MainIP
	if switchMeta.Spec.MainIP == "" {
		mainIP = "auto"
	}

	mgmtIP := switchMeta.Spec.MgmtIP
	if switchMeta.Spec.MgmtIP == "" {
		mgmtIP = "auto"
	}

	asn := switchMeta.Spec.ASN
	if switchMeta.Spec.ASN == 0 {
		mainIP = "auto"
	}

	switchUpdate := &inventory.HWSwitchUpdate{
		Name:        switchMeta.Spec.SwitchName,
		Description: switchMeta.Spec.Description,
		Tenant:      inventory.IDName{ID: switchMeta.Spec.TenantID},
		Nos:         switchMeta.Spec.NOS,
		Asn:         asn,
		Site:        inventory.IDName{ID: switchMeta.Spec.SiteID},
		Profile:     inventory.IDName{ID: switchMeta.Spec.ProfileID},
		MainAddress: mainIP,
		MgmtAddress: mgmtIP,
		PortCount:   switchMeta.Spec.PortsCount,
		MacAddress:  "",
		Links:       []inventory.HWLink{},
	}

	return switchUpdate, nil
}

func compareSwitchMetaAPIESwitch(switchMeta *k8sv1alpha1.SwitchMeta, apiSwitch *inventory.HW, u uniReconciler) bool {
	if apiSwitch.Name != switchMeta.Spec.SwitchName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiSwitch.Name, "k8sValue", switchMeta.Spec.SwitchName)
		return false
	}

	if apiSwitch.Description != switchMeta.Spec.Description {
		u.DebugLogger.Info("Description changed", "netrisValue", apiSwitch.Description, "k8sValue", switchMeta.Spec.Description)
		return false
	}

	if apiSwitch.Tenant.ID != switchMeta.Spec.TenantID {
		u.DebugLogger.Info("Tenant changed", "netrisValue", apiSwitch.Tenant.ID, "k8sValue", switchMeta.Spec.TenantID)
		return false
	}

	if apiSwitch.Site.ID != switchMeta.Spec.SiteID {
		u.DebugLogger.Info("Site changed", "netrisValue", apiSwitch.Site.ID, "k8sValue", switchMeta.Spec.SiteID)
		return false
	}

	if apiSwitch.Nos.Tag != switchMeta.Spec.NOS.Tag {
		u.DebugLogger.Info("NOS changed", "netrisValue", apiSwitch.Nos.Tag, "k8sValue", switchMeta.Spec.NOS.Tag)
		return false
	}

	if apiSwitch.Asn != switchMeta.Spec.ASN {
		u.DebugLogger.Info("ASN changed", "netrisValue", apiSwitch.Asn, "k8sValue", switchMeta.Spec.ASN)
		return false
	}

	if apiSwitch.PortCount != switchMeta.Spec.PortsCount {
		u.DebugLogger.Info("Ports Count changed", "netrisValue", apiSwitch.PortCount, "k8sValue", switchMeta.Spec.PortsCount)
		return false
	}

	if apiSwitch.MacAddress != switchMeta.Spec.MacAddress {
		u.DebugLogger.Info("MAC Address Count changed", "netrisValue", apiSwitch.MacAddress, "k8sValue", switchMeta.Spec.MacAddress)
		return false
	}

	if apiSwitch.Profile.ID != switchMeta.Spec.ProfileID {
		u.DebugLogger.Info("Profile changed", "netrisValue", apiSwitch.Profile.ID, "k8sValue", switchMeta.Spec.ProfileID)
		return false
	}

	if apiSwitch.MainIP.Address != switchMeta.Spec.MainIP {
		u.DebugLogger.Info("MainIP changed", "netrisValue", apiSwitch.MainIP.Address, "k8sValue", switchMeta.Spec.MainIP)
		return false
	}

	if apiSwitch.MgmtIP.Address != switchMeta.Spec.MgmtIP {
		u.DebugLogger.Info("MgmtIP changed", "netrisValue", apiSwitch.MgmtIP.Address, "k8sValue", switchMeta.Spec.MgmtIP)
		return false
	}

	return true
}
