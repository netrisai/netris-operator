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
	"encoding/json"
	"strings"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netriswebapi/v1/types/inventoryprofile"
	"github.com/r3labs/diff/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InventoryProfileToInventoryProfileMeta converts the InventoryProfile resource to InventoryProfileMeta type and used for add the InventoryProfile for Netris API.
func (r *InventoryProfileReconciler) InventoryProfileToInventoryProfileMeta(inventoryProfile *k8sv1alpha1.InventoryProfile) (*k8sv1alpha1.InventoryProfileMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := inventoryProfile.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := inventoryProfile.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	ntpServers := []string{}
	for _, s := range inventoryProfile.Spec.NTPServers {
		ntpServers = append(ntpServers, string(s))
	}

	dnsServers := []string{}
	for _, s := range inventoryProfile.Spec.DNSServers {
		dnsServers = append(dnsServers, string(s))
	}

	inventoryProfileMeta := &k8sv1alpha1.InventoryProfileMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(inventoryProfile.GetUID()),
			Namespace: inventoryProfile.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.InventoryProfileMetaSpec{
			Imported:             imported,
			Reclaim:              reclaim,
			InventoryProfileName: inventoryProfile.Name,
			Description:          inventoryProfile.Spec.Description,
			Timezone:             inventoryProfile.Spec.Timezone,
			AllowSSHFromIPv4:     inventoryProfile.Spec.AllowSSHFromIPv4,
			AllowSSHFromIPv6:     inventoryProfile.Spec.AllowSSHFromIPv6,
			NTPServers:           ntpServers,
			DNSServers:           dnsServers,
			CustomRules:          inventoryProfile.Spec.CustomRules,
		},
	}

	return inventoryProfileMeta, nil
}

func inventoryProfileCompareFieldsForNewMeta(inventoryProfile *k8sv1alpha1.InventoryProfile, inventoryProfileMeta *k8sv1alpha1.InventoryProfileMeta) bool {
	imported := false
	reclaim := false
	if i, ok := inventoryProfile.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := inventoryProfile.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return inventoryProfile.GetGeneration() != inventoryProfileMeta.Spec.InventoryProfileCRGeneration || imported != inventoryProfileMeta.Spec.Imported || reclaim != inventoryProfileMeta.Spec.Reclaim
}

func inventoryProfileMustUpdateAnnotations(inventoryProfile *k8sv1alpha1.InventoryProfile) bool {
	update := false
	if i, ok := inventoryProfile.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := inventoryProfile.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func inventoryProfileUpdateDefaultAnnotations(inventoryProfile *k8sv1alpha1.InventoryProfile) {
	imported := "false"
	reclaim := "delete"
	if i, ok := inventoryProfile.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := inventoryProfile.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := inventoryProfile.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	inventoryProfile.SetAnnotations(annotations)
}

// InventoryProfileMetaToNetris converts the k8s InventoryProfile resource to Netris type and used for add the InventoryProfile for Netris API.
func InventoryProfileMetaToNetris(inventoryProfileMeta *k8sv1alpha1.InventoryProfileMeta) (*inventoryprofile.ProfileW, error) {
	customRules := []inventoryprofile.CustomRule{}

	for _, customRule := range inventoryProfileMeta.Spec.CustomRules {
		customRules = append(customRules, inventoryprofile.CustomRule{
			SrcSubnet: customRule.SrcSubnet,
			SrcPort:   customRule.SrcPort,
			DstPort:   customRule.DstPort,
			Protocol:  customRule.Protocol,
		})
	}

	inventoryProfileAdd := &inventoryprofile.ProfileW{
		Name:        inventoryProfileMeta.Spec.InventoryProfileName,
		Description: inventoryProfileMeta.Spec.Description,
		Timezone:    inventoryprofile.Timezone{Label: inventoryProfileMeta.Spec.Timezone, TzCode: inventoryProfileMeta.Spec.Timezone},
		Ipv4List:    strings.Join(inventoryProfileMeta.Spec.AllowSSHFromIPv4, ","),
		Ipv6List:    strings.Join(inventoryProfileMeta.Spec.AllowSSHFromIPv6, ","),
		NTPServers:  strings.Join(inventoryProfileMeta.Spec.NTPServers, ","),
		DNSServers:  strings.Join(inventoryProfileMeta.Spec.DNSServers, ","),
		CustomRules: customRules,
	}

	return inventoryProfileAdd, nil
}

// InventoryProfileMetaToNetrisUpdate converts the k8s InventoryProfile resource to Netris type and used for update the InventoryProfile for Netris API.
func InventoryProfileMetaToNetrisUpdate(inventoryProfileMeta *k8sv1alpha1.InventoryProfileMeta) (*inventoryprofile.ProfileW, error) {
	customRules := []inventoryprofile.CustomRule{}

	for _, customRule := range inventoryProfileMeta.Spec.CustomRules {
		customRules = append(customRules, inventoryprofile.CustomRule{
			SrcSubnet: customRule.SrcSubnet,
			SrcPort:   customRule.SrcPort,
			DstPort:   customRule.DstPort,
			Protocol:  customRule.Protocol,
		})
	}

	inventoryProfileAdd := &inventoryprofile.ProfileW{
		ID:          inventoryProfileMeta.Spec.ID,
		Name:        inventoryProfileMeta.Spec.InventoryProfileName,
		Description: inventoryProfileMeta.Spec.Description,
		Timezone:    inventoryprofile.Timezone{Label: inventoryProfileMeta.Spec.Timezone, TzCode: inventoryProfileMeta.Spec.Timezone},
		Ipv4List:    strings.Join(inventoryProfileMeta.Spec.AllowSSHFromIPv4, ","),
		Ipv6List:    strings.Join(inventoryProfileMeta.Spec.AllowSSHFromIPv6, ","),
		NTPServers:  strings.Join(inventoryProfileMeta.Spec.NTPServers, ","),
		DNSServers:  strings.Join(inventoryProfileMeta.Spec.DNSServers, ","),
		CustomRules: customRules,
	}

	return inventoryProfileAdd, nil
}

func compareInventoryProfileMetaAPIEInventoryProfile(inventoryProfileMeta *k8sv1alpha1.InventoryProfileMeta, apiInventoryProfile *inventoryprofile.Profile, u uniReconciler) bool {
	if apiInventoryProfile.Name != inventoryProfileMeta.Spec.InventoryProfileName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiInventoryProfile.Name, "k8sValue", inventoryProfileMeta.Spec.InventoryProfileName)
		return false
	}
	if apiInventoryProfile.Description != inventoryProfileMeta.Spec.Description {
		u.DebugLogger.Info("Description changed", "netrisValue", apiInventoryProfile.Description, "k8sValue", inventoryProfileMeta.Spec.Description)
		return false
	}
	timeZone := unmarshalTimezone(apiInventoryProfile.Timezone)
	if timeZone.TzCode != inventoryProfileMeta.Spec.Timezone {
		u.DebugLogger.Info("Timezone changed", "netrisValue", timeZone.TzCode, "k8sValue", inventoryProfileMeta.Spec.Timezone)
		return false
	}

	if changelog, _ := diff.Diff(strings.Join(inventoryProfileMeta.Spec.AllowSSHFromIPv4, ","), apiInventoryProfile.Ipv4SSH); len(changelog) > 0 {
		u.DebugLogger.Info("AllowSSHFromIPv4 changed", "netrisValue", apiInventoryProfile.Ipv4SSH, "k8sValue", strings.Join(inventoryProfileMeta.Spec.AllowSSHFromIPv4, ","))
		return false
	}
	if changelog, _ := diff.Diff(strings.Join(inventoryProfileMeta.Spec.AllowSSHFromIPv6, ","), apiInventoryProfile.Ipv6SSH); len(changelog) > 0 {
		u.DebugLogger.Info("AllowSSHFromIPv6 changed", "netrisValue", apiInventoryProfile.Ipv6SSH, "k8sValue", strings.Join(inventoryProfileMeta.Spec.AllowSSHFromIPv6, ","))
		return false
	}
	if changelog, _ := diff.Diff(strings.Join(inventoryProfileMeta.Spec.NTPServers, ","), apiInventoryProfile.NTPServers); len(changelog) > 0 {
		u.DebugLogger.Info("NTPServers changed", "netrisValue", apiInventoryProfile.NTPServers, "k8sValue", strings.Join(inventoryProfileMeta.Spec.NTPServers, ","))
		return false
	}
	if changelog, _ := diff.Diff(strings.Join(inventoryProfileMeta.Spec.DNSServers, ","), apiInventoryProfile.DNSServers); len(changelog) > 0 {
		u.DebugLogger.Info("DNSServers changed", "netrisValue", apiInventoryProfile.DNSServers, "k8sValue", strings.Join(inventoryProfileMeta.Spec.DNSServers, ","))
		return false
	}

	if ok := compareInventoryProfileAPIInventoryProfileCustomRules(inventoryProfileMeta.Spec.CustomRules, apiInventoryProfile.CustomRules); !ok {
		u.DebugLogger.Info("CustomRules changed", "netrisValue", apiInventoryProfile.CustomRules, "k8sValue", inventoryProfileMeta.Spec.CustomRules, ",")
		return false
	}

	return true
}

func compareInventoryProfileAPIInventoryProfileCustomRules(InventoryProfileRules []k8sv1alpha1.InventoryProfileCustomRule, apiProfileRules []inventoryprofile.CustomRule) bool {
	type rule struct {
		SrcSubnet string `diff:"srcSubnet"`
		SrcPort   string `diff:"srcPort,omitempty"`
		DstPort   string `diff:"dstPort,omitempty"`
		Protocol  string `diff:"protocol"`
	}

	ProfileRules := []rule{}
	apiRules := []rule{}

	for _, m := range InventoryProfileRules {
		ProfileRules = append(ProfileRules, rule{
			SrcSubnet: m.SrcSubnet,
			SrcPort:   m.SrcPort,
			DstPort:   m.DstPort,
			Protocol:  m.Protocol,
		})
	}

	for _, m := range apiProfileRules {
		apiRules = append(apiRules, rule{
			SrcSubnet: m.SrcSubnet,
			SrcPort:   m.SrcPort,
			DstPort:   m.DstPort,
			Protocol:  m.Protocol,
		})
	}

	changelog, _ := diff.Diff(ProfileRules, apiRules)
	return len(changelog) <= 0
}

func unmarshalTimezone(s string) *inventoryprofile.Timezone {
	timezone := &inventoryprofile.Timezone{}
	_ = json.Unmarshal([]byte(s), timezone)
	return timezone
}
