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
	"github.com/netrisai/netriswebapi/v2/types/bgp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BGPToBGPMeta converts the BGP resource to BGPMeta type and used for add the BGP for Netris API.
func (r *BGPReconciler) BGPToBGPMeta(bgp *k8sv1alpha1.BGP) (*k8sv1alpha1.BGPMeta, error) {
	bgpMeta := &k8sv1alpha1.BGPMeta{}
	var (
		vlanID    = 0
		state     = "enabled"
		imported  = false
		reclaim   = false
		ipVersion = "ipv6"
		hwID      = 0
		portID    = 0
		vnetID    = 0
	)

	originate := "disabled"
	localPreference := 100

	if bgp.Spec.DefaultOriginate {
		originate = "enabled"
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
		if port, ok := r.NStorage.PortsStorage.FindByName(bgp.Spec.Transport.Name); ok {
			portID = port.ID
		} else if bgp.Spec.Transport.Name != "" {
			return nil, fmt.Errorf("Coundn't find port %s", bgp.Spec.Transport.Name)
		}
		vlanID = 1
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

	if bgp.Spec.Transport.VlanID > 1 {
		vlanID = bgp.Spec.Transport.VlanID
	}

	inventory, err := r.Cred.Inventory().Get()
	if err != nil {
		return nil, err
	}

	for _, hw := range inventory {
		if hw.Name == bgp.Spec.Hardware && bgp.Spec.Hardware != "auto" && bgp.Spec.Hardware != "" {
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

	var neighborAddress string

	if bgp.Spec.Multihop.NeighborAddress != "" && bgp.Spec.Multihop.Hops > 0 {
		neighborAddress = bgp.Spec.Multihop.NeighborAddress
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
			PortID:      portID,
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
			PrefixLimit:        strconv.Itoa(bgp.Spec.PrefixInboundMax), // ?
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

	var hwID interface{}
	if bgpMeta.Spec.HWID > 0 {
		hwID = bgpMeta.Spec.HWID
	} else {
		hwID = "auto"
	}

	bgpAdd := &bgp.EBGPAdd{
		AllowAsIn:          bgpMeta.Spec.AllowasIn,
		BgpPassword:        bgpMeta.Spec.BgpPassword,
		BgpCommunity:       bgpMeta.Spec.Community,
		Hardware:           bgp.IDNone{ID: hwID},
		Vnet:               bgp.IDNone{ID: vnetID},
		Port:               bgp.IDName{ID: bgpMeta.Spec.PortID},
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

	var hwID interface{}
	if bgpMeta.Spec.HWID > 0 {
		hwID = bgpMeta.Spec.HWID
	} else {
		hwID = "auto"
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
		Hardware:           bgp.IDNone{ID: hwID},
		Port:               bgp.IDName{ID: bgpMeta.Spec.PortID},
		Vnet:               bgp.IDNone{ID: vnetID},
		UpdateSource:       bgpMeta.Spec.UpdateSource,
		Vlan:               bgpMeta.Spec.Vlan,
		Weight:             bgpMeta.Spec.Weight,
	}

	return bgpAdd, nil
}

func compareBGPMetaAPIEBGP(bgpMeta *k8sv1alpha1.BGPMeta, apiBGP *bgp.EBGP, u uniReconciler) bool {
	if apiBGP.AllowasIn != bgpMeta.Spec.AllowasIn {
		u.DebugLogger.Info("AllowasIn changed", "netrisValue", apiBGP.AllowasIn, "k8sValue", bgpMeta.Spec.AllowasIn)
		return false
	}
	if apiBGP.BgpPassword != bgpMeta.Spec.BgpPassword {
		u.DebugLogger.Info("BgpPassword changed", "netrisValue", apiBGP.BgpPassword, "k8sValue", bgpMeta.Spec.BgpPassword)
		return false
	}
	if apiBGP.Community != bgpMeta.Spec.Community {
		u.DebugLogger.Info("Community changed", "netrisValue", apiBGP.Community, "k8sValue", bgpMeta.Spec.Community)
		return false
	}
	if apiBGP.Description != bgpMeta.Spec.Description {
		u.DebugLogger.Info("Description changed", "netrisValue", apiBGP.Description, "k8sValue", bgpMeta.Spec.Description)
		return false
	}
	if apiBGP.InboundRouteMap != bgpMeta.Spec.InboundRouteMap {
		u.DebugLogger.Info("InboundRouteMap changed", "netrisValue", apiBGP.InboundRouteMap, "k8sValue", bgpMeta.Spec.InboundRouteMap)
		return false
	}
	if apiBGP.IPVersion != bgpMeta.Spec.IPVersion {
		u.DebugLogger.Info("IPVersion changed", "netrisValue", apiBGP.IPVersion, "k8sValue", bgpMeta.Spec.IPVersion)
		return false
	}
	if apiBGP.LocalIP != bgpMeta.Spec.LocalIP {
		u.DebugLogger.Info("LocalIP changed", "netrisValue", apiBGP.LocalIP, "k8sValue", bgpMeta.Spec.LocalIP)
		return false
	}
	if apiBGP.LocalPreference != bgpMeta.Spec.LocalPreference {
		u.DebugLogger.Info("LocalPreference changed", "netrisValue", apiBGP.LocalPreference, "k8sValue", bgpMeta.Spec.LocalPreference)
		return false
	}
	if apiBGP.Multihop != bgpMeta.Spec.Multihop {
		u.DebugLogger.Info("Multihop changed", "netrisValue", apiBGP.Multihop, "k8sValue", bgpMeta.Spec.Multihop)
		return false
	}
	if apiBGP.Name != bgpMeta.Spec.BGPName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiBGP.Name, "k8sValue", bgpMeta.Spec.BGPName)
		return false
	}
	neighborAddress := ""
	if bgpMeta.Spec.NeighborAddress != "" {
		neighborAddress = bgpMeta.Spec.NeighborAddress
	}
	if apiBGP.NeighborAddress != neighborAddress {
		u.DebugLogger.Info("NeighborAddress changed", "netrisValue", apiBGP.NeighborAddress, "k8sValue", neighborAddress)
		return false
	}
	if apiBGP.NeighborAs != bgpMeta.Spec.NeighborAs {
		u.DebugLogger.Info("NeighborAs changed", "netrisValue", apiBGP.NeighborAs, "k8sValue", bgpMeta.Spec.NeighborAs)
		return false
	}
	if port, ok := u.NStorage.PortsStorage.FindByID(apiBGP.Port.ID); ok {
		if port.ID != bgpMeta.Spec.PortID {
			u.DebugLogger.Info("Port changed", "netrisValue", port.ID, "k8sValue", bgpMeta.Spec.PortID)
			return false
		}
	}
	if apiBGP.Originate != bgpMeta.Spec.Originate {
		u.DebugLogger.Info("Originate changed", "netrisValue", apiBGP.Originate, "k8sValue", bgpMeta.Spec.Originate)
		return false
	}
	if apiBGP.OutboundRouteMap != bgpMeta.Spec.OutboundRouteMap {
		u.DebugLogger.Info("OutboundRouteMap changed", "netrisValue", apiBGP.OutboundRouteMap, "k8sValue", bgpMeta.Spec.OutboundRouteMap)
		return false
	}
	if apiBGP.PrefixLength != bgpMeta.Spec.PrefixLength {
		u.DebugLogger.Info("PrefixLength changed", "netrisValue", apiBGP.PrefixLength, "k8sValue", bgpMeta.Spec.PrefixLength)
		return false
	}
	prefixLimit, _ := strconv.Atoi(bgpMeta.Spec.PrefixLimit)
	if apiBGP.PrefixLimit != prefixLimit {
		u.DebugLogger.Info("PrefixLimit changed", "netrisValue", apiBGP.PrefixLimit, "k8sValue", prefixLimit)
		return false
	}
	if apiBGP.PrefixListInbound != bgpMeta.Spec.PrefixListInbound {
		u.DebugLogger.Info("PrefixListInbound changed", "netrisValue", apiBGP.PrefixListInbound, "k8sValue", bgpMeta.Spec.PrefixListInbound)
		return false
	}
	if apiBGP.PrefixListOutbound != bgpMeta.Spec.PrefixListOutbound {
		u.DebugLogger.Info("PrefixListOutbound changed", "netrisValue", apiBGP.PrefixListOutbound, "k8sValue", bgpMeta.Spec.PrefixListOutbound)
		return false
	}
	if apiBGP.PrependInbound != bgpMeta.Spec.PrependInbound {
		u.DebugLogger.Info("PrependInbound changed", "netrisValue", apiBGP.PrependInbound, "k8sValue", bgpMeta.Spec.PrependInbound)
		return false
	}
	if apiBGP.PrependOutbound != bgpMeta.Spec.PrependInbound {
		u.DebugLogger.Info("PrependOutbound changed", "netrisValue", apiBGP.PrependOutbound, "k8sValue", bgpMeta.Spec.PrependOutbound)
		return false
	}
	if apiBGP.RemoteIP != bgpMeta.Spec.RemoteIP {
		u.DebugLogger.Info("RemoteIP changed", "netrisValue", apiBGP.RemoteIP, "k8sValue", bgpMeta.Spec.RemoteIP)
		return false
	}
	if apiBGP.SiteName != bgpMeta.Spec.Site {
		u.DebugLogger.Info("SiteName changed", "netrisValue", apiBGP.SiteName, "k8sValue", bgpMeta.Spec.Site)
		return false
	}
	if apiBGP.Status != bgpMeta.Spec.Status {
		u.DebugLogger.Info("Status changed", "netrisValue", apiBGP.Status, "k8sValue", bgpMeta.Spec.Status)
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
		u.DebugLogger.Info("UpdateSource changed", "netrisValue", apiBGP.UpdateSource, "k8sValue", bgpMeta.Spec.UpdateSource)
		return false
	}
	if apiBGP.Vlan != bgpMeta.Spec.Vlan {
		u.DebugLogger.Info("Vlan changed", "netrisValue", apiBGP.Vlan, "k8sValue", bgpMeta.Spec.Vlan)
		return false
	}
	if apiBGP.Weight != bgpMeta.Spec.Weight {
		u.DebugLogger.Info("Weight changed", "netrisValue", apiBGP.Weight, "k8sValue", bgpMeta.Spec.Weight)
		return false
	}

	return true
}
