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
	"net"
	"strconv"
	"strings"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netriswebapi/v2/types/bgp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BGPToBGPMeta converts the BGP resource to BGPMeta type and used for add the BGP for Netris API.
func (r *BGPReconciler) BGPToBGPMeta(bgp *k8sv1alpha1.BGP) (*k8sv1alpha1.BGPMeta, error) {
	bgpMeta := &k8sv1alpha1.BGPMeta{}
	var (
		vlanID    = 1
		state     = "enabled"
		imported  = false
		reclaim   = false
		ipVersion = "ipv6"
		hwID      = 0
		port      = ""
		vnetID    = 0
	)

	originate := "disabled"
	localPreference := 100

	if bgp.Spec.DefaultOriginate {
		originate = "enabled"
	}

	if bgp.Spec.Transport.VlanID > 1 {
		vlanID = bgp.Spec.Transport.VlanID
	}

	if bgp.Spec.LocalPreference > 0 {
		localPreference = bgp.Spec.LocalPreference
	}

	if len(bgp.Spec.State) > 0 {
		state = bgp.Spec.State
	}

	if bgp.Spec.Transport.Type == "" {
		bgp.Spec.Transport.Type = "port"
	}

	if bgp.Spec.Transport.Type == "port" {
		port = bgp.Spec.Transport.Name
	} else {
		vnets, err := r.Cred.VNet().Get()
		if err != nil {
			return nil, err
		}
		for _, vnet := range vnets {
			if vnet.Name == bgp.Spec.Transport.Name {
				vnetID = vnet.ID
			}
		}
	}

	inventory, err := r.Cred.Inventory().Get()
	if err != nil {
		return nil, err
	}

	for _, hw := range inventory {
		if hw.Name == bgp.Spec.Hardware {
			hwID = hw.ID
		}
	}

	if i, ok := bgp.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := bgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	localIP, cidr, _ := net.ParseCIDR(bgp.Spec.LocalIP)
	remoteIP, _, _ := net.ParseCIDR(bgp.Spec.RemoteIP)
	prefixLength, _ := cidr.Mask.Size()
	if localIP.To4() != nil {
		ipVersion = "ipv4"
	}

	var neighborAddress *string
	if bgp.Spec.Multihop.NeighborAddress != "" && bgp.Spec.Multihop.Hops > 0 {
		neighborAddress = &bgp.Spec.Multihop.NeighborAddress
	}

	bgpMeta = &k8sv1alpha1.BGPMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(bgp.GetUID()),
			Namespace: bgp.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.BGPMetaSpec{
			Imported:    imported,
			Reclaim:     reclaim,
			Name:        string(bgp.GetUID()),
			HWID:        hwID,
			VnetID:      vnetID,
			Port:        port,
			Site:        bgp.Spec.Site,
			BGPName:     bgp.Name,
			Vlan:        vlanID,
			NeighborAs:  bgp.Spec.NeighborAS,
			LocalIP:     localIP.String(),
			RemoteIP:    remoteIP.String(),
			Description: bgp.Spec.Description,
			Status:      state,

			NeighborAddress: neighborAddress,
			UpdateSource:    bgp.Spec.Multihop.UpdateSource,
			Multihop:        bgp.Spec.Multihop.Hops,

			BgpPassword:        bgp.Spec.BGPPassword,
			AllowasIn:          bgp.Spec.AllowAsIn,
			Originate:          originate,
			PrefixLimit:        bgp.Spec.PrefixInboundMax, // ?
			IPVersion:          ipVersion,
			InboundRouteMap:    bgpMeta.Spec.InboundRouteMap,
			LocalPreference:    localPreference,
			Weight:             bgp.Spec.Weight,
			PrependInbound:     bgp.Spec.PrependInbound,
			PrependOutbound:    bgp.Spec.PrependOutbound,
			PrefixLength:       prefixLength, // ?
			PrefixListInbound:  strings.Join(bgp.Spec.PrefixListInbound, "\n"),
			PrefixListOutbound: strings.Join(bgp.Spec.PrefixListOutbound, "\n"),
			Community:          strings.Join(bgp.Spec.SendBGPCommunity, "\n"),
		},
	}

	return bgpMeta, nil
}

func bgpCompareFieldsForNewMeta(bgp *k8sv1alpha1.BGP, bgpMeta *k8sv1alpha1.BGPMeta) bool {
	imported := false
	reclaim := false
	if i, ok := bgp.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := bgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return bgp.GetGeneration() != bgpMeta.Spec.BGPCRGeneration || imported != bgpMeta.Spec.Imported || reclaim != bgpMeta.Spec.Reclaim
}

func bgpMustUpdateAnnotations(bgp *k8sv1alpha1.BGP) bool {
	update := false
	if i, ok := bgp.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := bgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func bgpUpdateDefaultAnnotations(bgp *k8sv1alpha1.BGP) {
	imported := "false"
	reclaim := "delete"
	if i, ok := bgp.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := bgp.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := bgp.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	bgp.SetAnnotations(annotations)
}

// BGPMetaToNetris converts the k8s BGP resource to Netris type and used for add the BGP for Netris API.
func BGPMetaToNetris(bgpMeta *k8sv1alpha1.BGPMeta) (*bgp.EBGPAdd, error) {
	var vnetID interface{}
	if bgpMeta.Spec.VnetID > 0 {
		vnetID = bgpMeta.Spec.VnetID
	} else {
		vnetID = "none"
	}
	bgpAdd := &bgp.EBGPAdd{
		AllowAsIn:          bgpMeta.Spec.AllowasIn,
		BgpPassword:        bgpMeta.Spec.BgpPassword,
		BgpCommunity:       bgpMeta.Spec.Community,
		Hardware:           bgp.IDName{ID: bgpMeta.Spec.HWID},
		Vnet:               bgp.IDNone{ID: vnetID},
		Port:               bgp.IDName{Name: bgpMeta.Spec.Port},
		Description:        bgpMeta.Spec.Description,
		InboundRouteMap:    bgpMeta.Spec.InboundRouteMap,
		IPFamily:           bgpMeta.Spec.IPVersion,
		LocalIP:            bgpMeta.Spec.LocalIP,
		LocalPreference:    bgpMeta.Spec.LocalPreference,
		Multihop:           bgpMeta.Spec.Multihop,
		Name:               bgpMeta.Spec.BGPName,
		NeighborAddress:    bgpMeta.Spec.NeighborAddress,
		NeighborAS:         bgpMeta.Spec.NeighborAs,
		DefaultOriginate:   bgpMeta.Spec.Originate,
		OutboundRouteMap:   bgpMeta.Spec.OutboundRouteMap,
		PrefixLength:       bgpMeta.Spec.PrefixLength,
		PrefixInboundMax:   bgpMeta.Spec.PrefixLimit,
		PrefixListInbound:  bgpMeta.Spec.PrefixListInbound,
		PrefixListOutbound: bgpMeta.Spec.PrefixListOutbound,
		PrependInbound:     bgpMeta.Spec.PrependInbound,
		PrependOutbound:    bgpMeta.Spec.PrependInbound,
		RemoteIP:           bgpMeta.Spec.RemoteIP,
		Site:               bgp.IDName{Name: bgpMeta.Spec.Site},
		State:              bgpMeta.Spec.Status,
		UpdateSource:       bgpMeta.Spec.UpdateSource,
		Vlan:               bgpMeta.Spec.Vlan,
		Weight:             bgpMeta.Spec.Weight,
	}

	return bgpAdd, nil
}

// BGPMetaToNetrisUpdate converts the k8s BGP resource to Netris type and used for update the BGP for Netris API.
func BGPMetaToNetrisUpdate(bgpMeta *k8sv1alpha1.BGPMeta) (*bgp.EBGPUpdate, error) {
	var vnetID interface{}
	if bgpMeta.Spec.VnetID > 0 {
		vnetID = bgpMeta.Spec.VnetID
	} else {
		vnetID = "none"
	}
	bgpAdd := &bgp.EBGPUpdate{
		AllowAsIn:          bgpMeta.Spec.AllowasIn,
		BgpPassword:        bgpMeta.Spec.BgpPassword,
		BgpCommunity:       bgpMeta.Spec.Community,
		Description:        bgpMeta.Spec.Description,
		InboundRouteMap:    bgpMeta.Spec.InboundRouteMap,
		IPFamily:           bgpMeta.Spec.IPVersion,
		LocalIP:            bgpMeta.Spec.LocalIP,
		LocalPreference:    bgpMeta.Spec.LocalPreference,
		Multihop:           bgpMeta.Spec.Multihop,
		Name:               bgpMeta.Spec.BGPName,
		NeighborAddress:    bgpMeta.Spec.NeighborAddress,
		NeighborAS:         bgpMeta.Spec.NeighborAs,
		DefaultOriginate:   bgpMeta.Spec.Originate,
		OutboundRouteMap:   bgpMeta.Spec.OutboundRouteMap,
		PrefixLength:       bgpMeta.Spec.PrefixLength,
		PrefixInboundMax:   bgpMeta.Spec.PrefixLimit,
		PrefixListInbound:  bgpMeta.Spec.PrefixListInbound,
		PrefixListOutbound: bgpMeta.Spec.PrefixListOutbound,
		PrependInbound:     bgpMeta.Spec.PrependInbound,
		PrependOutbound:    bgpMeta.Spec.PrependInbound,
		RemoteIP:           bgpMeta.Spec.RemoteIP,
		Site:               bgp.IDName{Name: bgpMeta.Spec.Site},
		State:              bgpMeta.Spec.Status,
		Hardware:           bgp.IDName{ID: bgpMeta.Spec.HWID},
		Port:               bgp.IDName{Name: bgpMeta.Spec.Port},
		Vnet:               bgp.IDNone{ID: vnetID},
		UpdateSource:       bgpMeta.Spec.UpdateSource,
		Vlan:               bgpMeta.Spec.Vlan,
		Weight:             bgpMeta.Spec.Weight,
	}

	return bgpAdd, nil
}

func compareBGPMetaAPIEBGP(bgpMeta *k8sv1alpha1.BGPMeta, apiBGP *bgp.EBGP) bool {
	if apiBGP.AllowasIn != bgpMeta.Spec.AllowasIn {
		return false
	}
	if apiBGP.BgpPassword != bgpMeta.Spec.BgpPassword {
		return false
	}
	if apiBGP.Community != bgpMeta.Spec.Community {
		return false
	}
	if apiBGP.Description != bgpMeta.Spec.Description {
		return false
	}
	if apiBGP.InboundRouteMap != strconv.Itoa(bgpMeta.Spec.InboundRouteMap) {
		return false
	}
	if apiBGP.IPVersion != bgpMeta.Spec.IPVersion {
		return false
	}
	if apiBGP.LocalIP != bgpMeta.Spec.LocalIP {
		return false
	}
	if apiBGP.LocalPreference != bgpMeta.Spec.LocalPreference {
		return false
	}
	if apiBGP.Multihop != bgpMeta.Spec.Multihop {
		return false
	}
	if apiBGP.Name != bgpMeta.Spec.BGPName {
		return false
	}
	neighborAddress := ""
	if bgpMeta.Spec.NeighborAddress != nil {
		neighborAddress = *bgpMeta.Spec.NeighborAddress
	}
	if apiBGP.NeighborAddress != neighborAddress {
		return false
	}
	if apiBGP.NeighborAs != bgpMeta.Spec.NeighborAs {
		return false
	}
	// if apiBGP.NfvID != bgpMeta.Spec.NfvID {
	// 	return false
	// }
	// if apiBGP.NfvPortID != bgpMeta.Spec.NfvPortID {
	// 	return false
	// }
	if apiBGP.Originate != bgpMeta.Spec.Originate {
		return false
	}
	if apiBGP.OutboundRouteMap != strconv.Itoa(bgpMeta.Spec.OutboundRouteMap) {
		return false
	}
	if apiBGP.PrefixLength != bgpMeta.Spec.PrefixLength {
		return false
	}
	// if apiBGP.PrefixLimit != bgpMeta.Spec.PrefixLimit {
	// 	return false
	// }
	if apiBGP.PrefixListInbound != bgpMeta.Spec.PrefixListInbound {
		return false
	}
	if apiBGP.PrefixListOutbound != bgpMeta.Spec.PrefixListOutbound {
		return false
	}
	if apiBGP.PrependInbound != bgpMeta.Spec.PrependInbound {
		return false
	}
	if apiBGP.PrependOutbound != bgpMeta.Spec.PrependInbound {
		return false
	}
	if apiBGP.RemoteIP != bgpMeta.Spec.RemoteIP {
		return false
	}
	if apiBGP.SiteName != bgpMeta.Spec.Site {
		return false
	}
	if apiBGP.Status != bgpMeta.Spec.Status {
		return false
	}
	// if apiBGP.PortName != bgpMeta.Spec.Port {
	// 	fmt.Println("apiBGP.PortName", apiBGP.PortName)
	// 	return false
	// }
	// if apiBGP.TermSwitchID != bgpMeta.Spec.TermSwitchID {
	// 	return false
	// }

	if apiBGP.UpdateSource != bgpMeta.Spec.UpdateSource {
		return false
	}
	if apiBGP.Vlan != bgpMeta.Spec.Vlan {
		return false
	}
	if apiBGP.Weight != bgpMeta.Spec.Weight {
		return false
	}

	return true
}
