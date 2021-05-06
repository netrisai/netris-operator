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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VnetToVnetMeta converts the VNet resource to VNetMeta type and used for add the VNet for Netris API.
func (r *EBGPReconciler) EBGPToEBGPMeta(ebgp *k8sv1alpha1.EBGP) (*k8sv1alpha1.EBGPMeta, error) {
	ebgpMeta := &k8sv1alpha1.EBGPMeta{}
	var siteID int
	// var softgate string
	var state string
	originate := "false"
	localPreference := 100
	if site, ok := NStorage.SitesStorage.FindByName(ebgp.Spec.Site); ok {
		siteID = site.ID
	} else {
		return ebgpMeta, fmt.Errorf("invalid site '%s'", ebgp.Spec.Site)
	}

	if ebgp.Spec.DefaultOriginate {
		originate = "true"
	}

	if ebgp.Spec.LocalPreference > 0 {
		localPreference = ebgp.Spec.LocalPreference
	}

	if ebgp.Spec.State == "" {
		state = "enabled"
	}

	// if !ebgp.Spec.TerminateOnSwitch {
	// softgate = ebgp.Spec.Softgate
	// }

	imported := false
	reclaim := false
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	ebgpMeta = &k8sv1alpha1.EBGPMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(ebgp.GetUID()),
			Namespace: ebgp.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.EBGPMetaSpec{
			Imported: imported,
			Reclaim:  reclaim,
			Name:     string(ebgp.GetUID()),
			EBGPName: ebgp.Name,
			// Softgate: softgate, ?
			// Transport ?
			SiteID:            siteID,
			NeighborAs:        ebgp.Spec.NeighborAS,
			LocalIP:           ebgp.Spec.LocalIP,
			RemoteIP:          ebgp.Spec.RemoteIP,
			Description:       ebgp.Spec.Description,
			Status:            state,
			TerminateOnSwitch: ebgpMeta.Spec.TerminateOnSwitch,
			// Multihop ?
			BgpPassword:     ebgp.Spec.BGPPassword,
			AllowasIn:       ebgp.Spec.AllowAsIn,
			Originate:       originate,
			PrefixLimit:     ebgp.Spec.PrefixInboundMax, // ?
			InboundRouteMap: ebgpMeta.Spec.InboundRouteMap,
			LocalPreference: localPreference,
			Weight:          ebgp.Spec.Weight,
			PrependInbound:  ebgp.Spec.PrependInbound,
			PrependOutbound: ebgp.Spec.PrependOutbound,
			// PrefixLength: , ?
			PrefixListInbound:  ebgpMeta.Spec.PrefixListInbound,
			PrefixListOutbound: ebgpMeta.Spec.PrefixListOutbound,
			// Community:          ebgp.Spec.SendBGPCommunity, ?
		},
	}

	return ebgpMeta, nil
}

func ebgpCompareFieldsForNewMeta(ebgp *k8sv1alpha1.EBGP, ebgpMeta *k8sv1alpha1.EBGPMeta) bool {
	imported := false
	reclaim := false
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return ebgp.GetGeneration() != ebgpMeta.Spec.EBGPCRGeneration || imported != ebgpMeta.Spec.Imported || reclaim != ebgpMeta.Spec.Reclaim
}

func ebgpMustUpdateAnnotations(ebgp *k8sv1alpha1.EBGP) bool {
	update := false
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func ebgpUpdateDefaultAnnotations(ebgp *k8sv1alpha1.EBGP) {
	imported := "false"
	reclaim := "delete"
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := ebgp.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	ebgp.SetAnnotations(annotations)
}
