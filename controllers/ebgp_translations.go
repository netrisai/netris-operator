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
	"net"
	"strconv"
	"strings"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	api "github.com/netrisai/netrisapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EBGPToEBGPMeta converts the EBGP resource to EBGPMeta type and used for add the EBGP for Netris API.
func (r *EBGPReconciler) EBGPToEBGPMeta(ebgp *k8sv1alpha1.EBGP) (*k8sv1alpha1.EBGPMeta, error) {
	ebgpMeta := &k8sv1alpha1.EBGPMeta{}
	var siteID int
	var nfvID int
	var nfvPortID int
	var state string
	terminateOnSwitch := "false"

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

	if !ebgp.Spec.TerminateOnSwitch {
		if softgate, ok := NStorage.EBGPStorage.FindOffloaderByName(siteID, ebgp.Spec.Softgate); ok {
			nfvID = softgate.SwitchID
			nfvPortID = softgate.OffloadPortID
		} else {
			return ebgpMeta, fmt.Errorf("invalid softgate '%s'", ebgp.Spec.Softgate)
		}
	} else {
		terminateOnSwitch = "true"
	}

	var portID int
	var vlanID int
	var vnetID int

	if ebgp.Spec.Transport.Type == "port" {
		if port, ok := NStorage.EBGPStorage.FindPort(siteID, ebgp.Spec.Transport.Name); ok {
			portID = port.PortID
			vlanID = ebgp.Spec.Transport.VlanID
		} else {
			return ebgpMeta, fmt.Errorf("invalid port '%s'", ebgp.Spec.Transport.Name)
		}
	} else {
		if vnet, ok := NStorage.EBGPStorage.FindVNetByName(ebgp.Spec.Transport.Name); ok {
			vnetID = vnet.ID
		} else {
			return ebgpMeta, fmt.Errorf("invalid vnet '%s'", ebgp.Spec.Transport.Name)
		}
	}

	imported := false
	reclaim := false
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := ebgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	_, net, _ := net.ParseCIDR(ebgp.Spec.LocalIP)
	prefixLength, _ := net.Mask.Size()

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

			NfvID:     nfvID,
			NfvPortID: nfvPortID,

			SwitchPortID: portID,
			Vlan:         vlanID,
			RcircuitID:   vnetID,

			SiteID:            siteID,
			NeighborAs:        ebgp.Spec.NeighborAS,
			LocalIP:           ebgp.Spec.LocalIP,
			RemoteIP:          ebgp.Spec.RemoteIP,
			Description:       ebgp.Spec.Description,
			Status:            state,
			TerminateOnSwitch: terminateOnSwitch,

			NeighborAddress: ebgp.Spec.Multihop.NeighborAddress,
			UpdateSource:    ebgp.Spec.Multihop.UpdateSource,
			Multihop:        ebgp.Spec.Multihop.Hops,

			BgpPassword:        ebgp.Spec.BGPPassword,
			AllowasIn:          ebgp.Spec.AllowAsIn,
			Originate:          originate,
			PrefixLimit:        ebgp.Spec.PrefixInboundMax, // ?
			InboundRouteMap:    ebgpMeta.Spec.InboundRouteMap,
			LocalPreference:    localPreference,
			Weight:             ebgp.Spec.Weight,
			PrependInbound:     ebgp.Spec.PrependInbound,
			PrependOutbound:    ebgp.Spec.PrependOutbound,
			PrefixLength:       prefixLength, // ?
			PrefixListInbound:  strings.Join(ebgp.Spec.PrefixListInbound, "\n"),
			PrefixListOutbound: strings.Join(ebgp.Spec.PrefixListOutbound, "\n"),
			Community:          strings.Join(ebgp.Spec.SendBGPCommunity, "\n"),
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

// EBGPMetaToNetris converts the k8s EBGP resource to Netris type and used for add the EBGP for Netris API.
func EBGPMetaToNetris(ebgpMeta *k8sv1alpha1.EBGPMeta) (*api.APIEBGPAdd, error) {
	ebgpAdd := &api.APIEBGPAdd{
		AllowasIn:          ebgpMeta.Spec.AllowasIn,
		BgpPassword:        ebgpMeta.Spec.BgpPassword,
		Community:          ebgpMeta.Spec.Community,
		Description:        ebgpMeta.Spec.Description,
		InboundRouteMap:    ebgpMeta.Spec.InboundRouteMap,
		IPVersion:          ebgpMeta.Spec.IPVersion,
		LocalIP:            ebgpMeta.Spec.LocalIP,
		LocalPreference:    ebgpMeta.Spec.LocalPreference,
		Multihop:           ebgpMeta.Spec.Multihop,
		Name:               ebgpMeta.Spec.EBGPName,
		NeighborAddress:    ebgpMeta.Spec.NeighborAddress,
		NeighborAs:         strconv.Itoa(ebgpMeta.Spec.NeighborAs),
		NfvID:              ebgpMeta.Spec.NfvID,
		NfvPortID:          ebgpMeta.Spec.NfvPortID,
		Originate:          ebgpMeta.Spec.Originate,
		OutboundRouteMap:   ebgpMeta.Spec.OutboundRouteMap,
		PrefixLength:       ebgpMeta.Spec.PrefixLength,
		PrefixLimit:        strconv.Itoa(ebgpMeta.Spec.PrefixLimit),
		PrefixListInbound:  ebgpMeta.Spec.PrefixListInbound,
		PrefixListOutbound: ebgpMeta.Spec.PrefixListOutbound,
		PrependInbound:     ebgpMeta.Spec.PrependInbound,
		PrependOutbound:    ebgpMeta.Spec.PrependInbound,
		RcircuitID:         ebgpMeta.Spec.RcircuitID,
		RemoteIP:           ebgpMeta.Spec.RemoteIP,
		SiteID:             ebgpMeta.Spec.SiteID,
		Status:             ebgpMeta.Spec.Status,
		SwitchID:           ebgpMeta.Spec.SwitchID,
		SwitchName:         ebgpMeta.Spec.SwitchName,
		SwitchPortID:       ebgpMeta.Spec.SwitchPortID,
		TermSwitchID:       ebgpMeta.Spec.TermSwitchID,
		TermSwitchName:     ebgpMeta.Spec.TermSwitchName,
		TerminateOnSwitch:  ebgpMeta.Spec.TerminateOnSwitch,
		UpdateSource:       ebgpMeta.Spec.UpdateSource,
		Vlan:               ebgpMeta.Spec.Vlan,
		Weight:             ebgpMeta.Spec.Weight,
	}

	return ebgpAdd, nil
}

// EBGPMetaToNetrisUpdate converts the k8s EBGP resource to Netris type and used for update the EBGP for Netris API.
func EBGPMetaToNetrisUpdate(ebgpMeta *k8sv1alpha1.EBGPMeta) (*api.APIEBGPUpdate, error) {
	ebgpAdd := &api.APIEBGPUpdate{
		ID:                 ebgpMeta.Spec.ID,
		AllowasIn:          ebgpMeta.Spec.AllowasIn,
		BgpPassword:        ebgpMeta.Spec.BgpPassword,
		Community:          ebgpMeta.Spec.Community,
		Description:        ebgpMeta.Spec.Description,
		InboundRouteMap:    ebgpMeta.Spec.InboundRouteMap,
		IPVersion:          ebgpMeta.Spec.IPVersion,
		LocalIP:            ebgpMeta.Spec.LocalIP,
		LocalPreference:    ebgpMeta.Spec.LocalPreference,
		Multihop:           ebgpMeta.Spec.Multihop,
		Name:               ebgpMeta.Spec.EBGPName,
		NeighborAddress:    ebgpMeta.Spec.NeighborAddress,
		NeighborAs:         strconv.Itoa(ebgpMeta.Spec.NeighborAs),
		NfvID:              ebgpMeta.Spec.NfvID,
		NfvPortID:          ebgpMeta.Spec.NfvPortID,
		Originate:          ebgpMeta.Spec.Originate,
		OutboundRouteMap:   ebgpMeta.Spec.OutboundRouteMap,
		PrefixLength:       ebgpMeta.Spec.PrefixLength,
		PrefixLimit:        ebgpMeta.Spec.PrefixLimit,
		PrefixListInbound:  ebgpMeta.Spec.PrefixListInbound,
		PrefixListOutbound: ebgpMeta.Spec.PrefixListOutbound,
		PrependInbound:     ebgpMeta.Spec.PrependInbound,
		PrependOutbound:    ebgpMeta.Spec.PrependInbound,
		RcircuitID:         ebgpMeta.Spec.RcircuitID,
		RemoteIP:           ebgpMeta.Spec.RemoteIP,
		SiteID:             ebgpMeta.Spec.SiteID,
		Status:             ebgpMeta.Spec.Status,
		SwitchID:           ebgpMeta.Spec.SwitchID,
		SwitchName:         ebgpMeta.Spec.SwitchName,
		SwitchPortID:       ebgpMeta.Spec.SwitchPortID,
		TermSwitchID:       ebgpMeta.Spec.TermSwitchID,
		TermSwitchName:     ebgpMeta.Spec.TermSwitchName,
		TerminateOnSwitch:  ebgpMeta.Spec.TerminateOnSwitch,
		UpdateSource:       ebgpMeta.Spec.UpdateSource,
		Vlan:               ebgpMeta.Spec.Vlan,
		Weight:             ebgpMeta.Spec.Weight,
	}

	return ebgpAdd, nil
}

func compareEBGPMetaAPIEBGP(vnetMeta *k8sv1alpha1.EBGPMeta, apiEBGP *api.APIEBGP) bool {
	return true
}
