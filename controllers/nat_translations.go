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
	"strconv"
	"strings"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netriswebapi/v2/types/nat"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NatToNatMeta converts the Nat resource to NatMeta type and used for add the Nat for Netris API.
func (r *NatReconciler) NatToNatMeta(nat *k8sv1alpha1.Nat) (*k8sv1alpha1.NatMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := nat.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := nat.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	siteID := 0
	if site, ok := r.NStorage.SitesStorage.FindByName(nat.Spec.Site); ok {
		siteID = site.ID
	} else {
		return nil, fmt.Errorf("Invalid site '%s'", nat.Spec.Site)
	}

	state := nat.Spec.State
	if nat.Spec.State == "" {
		state = "enabled"
	}

	natMeta := &k8sv1alpha1.NatMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(nat.GetUID()),
			Namespace: nat.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.NatMetaSpec{
			Imported:   imported,
			Reclaim:    reclaim,
			NatName:    nat.Name,
			Comment:    nat.Spec.Comment,
			State:      state,
			SiteID:     siteID,
			Action:     strings.ToUpper(nat.Spec.Action),
			Protocol:   nat.Spec.Protocol,
			SrcAddress: nat.Spec.SrcAddress,
			SrcPort:    nat.Spec.SrcPort,
			DstAddress: nat.Spec.DstAddress,
			DstPort:    nat.Spec.DstPort,
			SnatToIP:   nat.Spec.SnatToIP,
			SnatToPool: nat.Spec.SnatToPool,
			DnatToIP:   nat.Spec.DnatToIP,
			DnatToPort: nat.Spec.DnatToPort,
		},
	}

	return natMeta, nil
}

func natCompareFieldsForNewMeta(nat *k8sv1alpha1.Nat, natMeta *k8sv1alpha1.NatMeta) bool {
	imported := false
	reclaim := false
	if i, ok := nat.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := nat.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return nat.GetGeneration() != natMeta.Spec.NatCRGeneration || imported != natMeta.Spec.Imported || reclaim != natMeta.Spec.Reclaim
}

func natMustUpdateAnnotations(nat *k8sv1alpha1.Nat) bool {
	update := false
	if i, ok := nat.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := nat.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func natUpdateDefaultAnnotations(nat *k8sv1alpha1.Nat) {
	imported := "false"
	reclaim := "delete"
	if i, ok := nat.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := nat.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := nat.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	nat.SetAnnotations(annotations)
}

// NatMetaToNetris converts the k8s Nat resource to Netris type and used for add the Nat for Netris API.
func NatMetaToNetris(natMeta *k8sv1alpha1.NatMeta) (*nat.NATw, error) {
	natAdd := &nat.NATw{
		Name:               natMeta.Spec.NatName,
		Comment:            natMeta.Spec.Comment,
		State:              natMeta.Spec.State,
		Site:               nat.IDName{ID: natMeta.Spec.SiteID},
		Action:             natMeta.Spec.Action,
		Protocol:           natMeta.Spec.Protocol,
		SourceAddress:      natMeta.Spec.SrcAddress,
		SourcePort:         natMeta.Spec.SrcPort,
		DestinationAddress: natMeta.Spec.DstAddress,
		DestinationPort:    natMeta.Spec.SrcPort,
		SnatToIP:           natMeta.Spec.SnatToIP,
		SnatToPool:         natMeta.Spec.SnatToPool,
		DnatToIP:           natMeta.Spec.DnatToIP,
		DnatToPort:         strconv.Itoa(natMeta.Spec.DnatToPort),
	}

	return natAdd, nil
}

// NatMetaToNetrisUpdate converts the k8s Nat resource to Netris type and used for update the Nat for Netris API.
func NatMetaToNetrisUpdate(natMeta *k8sv1alpha1.NatMeta) (*nat.NATw, error) {
	natAdd := &nat.NATw{
		Name:               natMeta.Spec.NatName,
		Comment:            natMeta.Spec.Comment,
		State:              natMeta.Spec.State,
		Site:               nat.IDName{ID: natMeta.Spec.SiteID},
		Action:             natMeta.Spec.Action,
		Protocol:           natMeta.Spec.Protocol,
		SourceAddress:      natMeta.Spec.SrcAddress,
		SourcePort:         natMeta.Spec.SrcPort,
		DestinationAddress: natMeta.Spec.DstAddress,
		DestinationPort:    natMeta.Spec.DstPort,
		SnatToIP:           natMeta.Spec.SnatToIP,
		SnatToPool:         natMeta.Spec.SnatToPool,
		DnatToIP:           natMeta.Spec.DnatToIP,
		DnatToPort:         strconv.Itoa(natMeta.Spec.DnatToPort),
	}

	return natAdd, nil
}

func compareNatMetaAPIENat(natMeta *k8sv1alpha1.NatMeta, apiNat *nat.NAT, u uniReconciler) bool {
	if apiNat.Name != natMeta.Spec.NatName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiNat.Name, "k8sValue", natMeta.Spec.NatName)
		return false
	}
	if apiNat.Comment != natMeta.Spec.Comment {
		u.DebugLogger.Info("Comment changed", "netrisValue", apiNat.Comment, "k8sValue", natMeta.Spec.Comment)
		return false
	}
	if apiNat.State.Value != natMeta.Spec.State {
		u.DebugLogger.Info("State changed", "netrisValue", apiNat.State.Value, "k8sValue", natMeta.Spec.State)
		return false
	}
	if apiNat.Site.ID != natMeta.Spec.SiteID {
		u.DebugLogger.Info("Sote changed", "netrisValue", apiNat.Site.ID, "k8sValue", natMeta.Spec.SiteID)
		return false
	}
	apiAction := apiNat.Action.Label
	if apiAction == "ACCEPT" {
		apiAction = "ACCEPT_SNAT"
	}
	if apiAction != natMeta.Spec.Action {
		u.DebugLogger.Info("Action changed", "netrisValue", apiNat.Action.Label, "k8sValue", natMeta.Spec.Action)
		return false
	}
	if apiNat.Protocol.Value != natMeta.Spec.Protocol {
		u.DebugLogger.Info("Protocol changed", "netrisValue", apiNat.Protocol.Value, "k8sValue", natMeta.Spec.Protocol)
		return false
	}
	if apiNat.SourceAddress != natMeta.Spec.SrcAddress {
		u.DebugLogger.Info("SourceAddress changed", "netrisValue", apiNat.SourceAddress, "k8sValue", natMeta.Spec.SrcAddress)
		return false
	}
	if (apiNat.Protocol.Value == "tcp" || apiNat.Protocol.Value == "udp") && apiNat.SourcePort != natMeta.Spec.SrcPort {
		u.DebugLogger.Info("SourcePort changed", "netrisValue", apiNat.SourcePort, "k8sValue", natMeta.Spec.SrcPort)
		return false
	}
	if apiNat.DestinationAddress != natMeta.Spec.DstAddress {
		natMetaDst := strings.Split(natMeta.Spec.DstAddress, "/")[0]
		if apiNat.DestinationAddress != natMetaDst {
			u.DebugLogger.Info("DestinationAddress changed", "netrisValue", apiNat.DestinationAddress, "k8sValue", natMeta.Spec.DstAddress)
			return false
		}
	}
	if (apiNat.Protocol.Value == "tcp" || apiNat.Protocol.Value == "udp") && apiNat.DestinationPort != natMeta.Spec.DstPort {
		u.DebugLogger.Info("DestinationPort changed", "netrisValue", apiNat.DestinationPort, "k8sValue", natMeta.Spec.DstPort)
		return false
	}
	if apiNat.SnatToIP != natMeta.Spec.SnatToIP {
		u.DebugLogger.Info("SnatToIP changed", "netrisValue", apiNat.SnatToIP, "k8sValue", natMeta.Spec.SnatToIP)
		return false
	}
	if apiNat.SnatToPool != natMeta.Spec.SnatToPool {
		u.DebugLogger.Info("SnatToPool changed", "netrisValue", apiNat.SnatToPool, "k8sValue", natMeta.Spec.SnatToPool)
		return false
	}
	if apiNat.DnatToIP != natMeta.Spec.DnatToIP {
		u.DebugLogger.Info("DnatToIP changed", "netrisValue", apiNat.DnatToIP, "k8sValue", natMeta.Spec.DnatToIP)
		return false
	}
	if apiNat.DnatToPort != natMeta.Spec.DnatToPort {
		u.DebugLogger.Info("DnatToPort changed", "netrisValue", apiNat.DnatToPort, "k8sValue", natMeta.Spec.DnatToPort)
		return false
	}

	return true
}
