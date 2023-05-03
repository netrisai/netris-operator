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
	"github.com/netrisai/netriswebapi/v2/types/vnet"
	"github.com/r3labs/diff/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VnetToVnetMeta converts the VNet resource to VNetMeta type and used for add the VNet for Netris API.
func (r *VNetReconciler) VnetToVnetMeta(vnet *k8sv1alpha1.VNet) (*k8sv1alpha1.VNetMeta, error) {
	ports := []k8sv1alpha1.VNetSwitchPort{}
	siteNames := []string{}
	apiGateways := []k8sv1alpha1.VNetMetaGateway{}

	for _, site := range vnet.Spec.Sites {
		siteNames = append(siteNames, site.Name)
		ports = append(ports, site.SwitchPorts...)
		for _, gateway := range site.Gateways {
			apiGateways = append(apiGateways, makeGateway(gateway))
		}
	}
	prts, err := r.getPortsMeta(ports)
	if err != nil {
		return nil, err
	}

	portsList := []k8sv1alpha1.VNetMetaMember{}
	for _, port := range prts {
		p := port
		if (port.Vlan == "" || port.Vlan == "1") && vnet.Spec.VlanID != "" {
			p.Vlan = vnet.Spec.VlanID
		}
		portsList = append(portsList, p)
	}

	sites := getSites(siteNames, r.NStorage)
	sitesList := []k8sv1alpha1.VNetMetaSite{}

	for name, id := range sites {
		sitesList = append(sitesList, k8sv1alpha1.VNetMetaSite{
			Name: name,
			ID:   id,
		})
	}

	state := "active"
	if len(vnet.Spec.State) > 0 {
		if !(vnet.Spec.State == "active" || vnet.Spec.State == "disabled") {
			return nil, fmt.Errorf("invalid spec.state field")
		}
		state = vnet.Spec.State
	}

	imported := false
	reclaim := false
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	vnetMeta := &k8sv1alpha1.VNetMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(vnet.GetUID()),
			Namespace: vnet.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.VNetMetaSpec{
			Imported:     imported,
			Reclaim:      reclaim,
			Name:         string(vnet.GetUID()),
			VnetName:     vnet.Name,
			Sites:        sitesList,
			State:        state,
			Owner:        vnet.Spec.Owner,
			Tenants:      vnet.Spec.GuestTenants,
			Gateways:     apiGateways,
			Members:      portsList,
			Provisioning: 1,
			VaMode:       false,
			VaNativeVLAN: 1,
			VaVLANs:      "",
			VlanID:       vnet.Spec.VlanID,
		},
	}

	return vnetMeta, nil
}

// VnetMetaToNetris converts the k8s VNet resource to Netris type and used for add the VNet for Netris API.
func (r *VNetMetaReconciler) VnetMetaToNetris(vnetMeta *k8sv1alpha1.VNetMeta) (*vnet.VNetAdd, error) {
	apiGateways := []vnet.VNetAddGateway{}

	sites := []vnet.VNetAddSite{}
	vlanid := vnetMeta.Spec.VlanID
	members := []vnet.VNetAddPort{}

	for _, site := range vnetMeta.Spec.Sites {
		sites = append(sites, vnet.VNetAddSite{Name: site.Name})
	}

	for _, port := range vnetMeta.Spec.Members {
		vID := vlanid
		if (port.Vlan != "1" || vlanid == "") && vlanid != "auto" {
			vID = port.Vlan
		}
		members = append(members, vnet.VNetAddPort{
			Name:  port.Name,
			Vlan:  vID,
			Lacp:  port.Lacp,
			State: "active",
			ID:    port.ID,
		})
	}

	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateway := vnet.VNetAddGateway{
			Prefix: fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength),
		}
		if gateway.DHCP {
			apiGateway.DHCPEnabled = true
			apiGateway.DHCPLeaseCount = 2
			if gateway.DHCPStartIP != "" {
				apiGateway.DHCP = &vnet.VNetGatewayDHCP{
					OptionSet: vnet.IDName{Name: gateway.DHCPOptionSet},
					Start:     gateway.DHCPStartIP,
					End:       gateway.DHCPEndIP,
				}
			}

		}
		apiGateways = append(apiGateways, apiGateway)
	}

	guestTenants := []vnet.VNetAddTenant{}
	for _, tenant := range vnetMeta.Spec.Tenants {
		guestTenants = append(guestTenants, vnet.VNetAddTenant{Name: tenant})
	}

	vnetAdd := &vnet.VNetAdd{
		Name:         vnetMeta.Spec.VnetName,
		Sites:        sites,
		Tenant:       vnet.VNetAddTenant{Name: vnetMeta.Spec.Owner},
		State:        vnetMeta.Spec.State,
		GuestTenants: guestTenants,
		Gateways:     apiGateways,
		Ports:        members,
		NativeVlan:   1,
		Vlan:         vnetMeta.Spec.VlanID,
		Tags:         []string{},
	}

	return vnetAdd, nil
}

// VnetMetaToNetrisUpdate converts the k8s VNet resource to Netris type and used for update the VNet for Netris API.
func VnetMetaToNetrisUpdate(vnetMeta *k8sv1alpha1.VNetMeta) (*vnet.VNetUpdate, error) {
	apiGateways := []vnet.VNetUpdateGateway{}

	sites := []vnet.VNetUpdateSite{}
	vlanid := vnetMeta.Spec.VlanID
	members := []vnet.VNetUpdatePort{}

	for _, site := range vnetMeta.Spec.Sites {
		sites = append(sites, vnet.VNetUpdateSite{Name: site.Name})
	}

	for _, port := range vnetMeta.Spec.Members {
		vID := vlanid
		if (port.Vlan != "1" || vlanid == "") && vlanid != "auto" {
			vID = port.Vlan
		}
		members = append(members, vnet.VNetUpdatePort{
			Name:  port.Name,
			Vlan:  vID,
			Lacp:  port.Lacp,
			State: "active",
			ID:    port.ID,
		})
	}

	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateway := vnet.VNetUpdateGateway{
			Prefix: fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength),
		}
		if gateway.DHCP {
			apiGateway.DHCPEnabled = true
			apiGateway.DHCPLeaseCount = 2
			if gateway.DHCPStartIP != "" {
				apiGateway.DHCP = &vnet.VNetGatewayDHCP{
					OptionSet: vnet.IDName{Name: gateway.DHCPOptionSet},
					Start:     gateway.DHCPStartIP,
					End:       gateway.DHCPEndIP,
				}
			}
		}
		apiGateways = append(apiGateways, apiGateway)
	}

	guestTenants := []vnet.VNetUpdateGuestTenant{}
	for _, tenant := range vnetMeta.Spec.Tenants {
		guestTenants = append(guestTenants, vnet.VNetUpdateGuestTenant{Name: tenant})
	}

	vnetUpdate := &vnet.VNetUpdate{
		Name:         vnetMeta.Spec.VnetName,
		Sites:        sites,
		State:        vnetMeta.Spec.State,
		GuestTenants: guestTenants,
		Gateways:     apiGateways,
		Ports:        members,
		NativeVlan:   1,
		Vlan:         vnetMeta.Spec.VlanID,
		Tags:         []string{},
	}

	return vnetUpdate, nil
}

func compareVNetMetaAPIVnetGateways(vnetMetaGateways []k8sv1alpha1.VNetMetaGateway, apiVnetGateways []vnet.VNetDetailedGateway) bool {
	type compareGateway struct {
		Prefix        string `diff:"gateway"`
		DHCP          bool   `json:"dhcp"`
		DHCPOptionSet string `json:"dhcpOptionSet"`
		DHCPStartIP   string `json:"dhcpStartIP"`
		DHCPEndIP     string `json:"dhcpEndIP"`
	}

	vnetGateways := []compareGateway{}
	apiGateways := []compareGateway{}

	for _, gateway := range vnetMetaGateways {
		apiGateway := compareGateway{
			Prefix: fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength),
		}
		if gateway.DHCP {
			apiGateway.DHCP = true
			apiGateway.DHCPOptionSet = gateway.DHCPOptionSet
			apiGateway.DHCPStartIP = gateway.DHCPStartIP
			apiGateway.DHCPEndIP = gateway.DHCPEndIP
		}
		apiGateways = append(apiGateways, apiGateway)
	}

	for _, gateway := range apiVnetGateways {
		vnetGateway := compareGateway{
			Prefix: gateway.Prefix,
		}
		if gateway.DHCP != nil && gateway.DHCPEnabled {
			vnetGateway.DHCP = true
			vnetGateway.DHCPOptionSet = gateway.DHCP.OptionSet.Name
			vnetGateway.DHCPStartIP = gateway.DHCP.Start
			vnetGateway.DHCPEndIP = gateway.DHCP.End
		}
		vnetGateways = append(vnetGateways, vnetGateway)
	}

	changelog, _ := diff.Diff(vnetGateways, apiGateways)

	return len(changelog) <= 0
}

func compareVNetMetaAPIVnetMembers(vnetMetaMembers []k8sv1alpha1.VNetMetaMember, apiVnetMembers []vnet.VNetDetailedPort) bool {
	type member struct {
		PortID int    `diff:"port_id"`
		VLANID string `diff:"vlan_id"`
	}

	vnetMembers := []member{}
	apiMembers := []member{}

	for _, m := range vnetMetaMembers {
		vnetMembers = append(vnetMembers, member{
			PortID: m.ID,
			VLANID: m.Vlan,
		})
	}

	for _, m := range apiVnetMembers {
		apiMembers = append(apiMembers, member{
			PortID: m.ID,
			VLANID: m.Vlan,
		})
	}

	changelog, _ := diff.Diff(vnetMembers, apiMembers)
	return len(changelog) <= 0
}

func compareVNetMetaAPIVnetTenants(vnetMetaTenants []string, apiVnetTenants []vnet.VNetDetailedGuestTenant) bool {
	tenantList := []string{}
	for _, tenant := range apiVnetTenants {
		tenantList = append(tenantList, tenant.Name)
	}
	changelog, _ := diff.Diff(vnetMetaTenants, tenantList)
	return len(changelog) <= 0
}

func compareVNetMetaAPIVnetSites(vnetMetaSites []k8sv1alpha1.VNetMetaSite, apiVnetSites []vnet.VNetDetailedSite) bool {
	k8sSites := make(map[string]string)
	for _, site := range vnetMetaSites {
		k8sSites[site.Name] = ""
	}

	for _, site := range apiVnetSites {
		if _, ok := k8sSites[site.Name]; !ok {
			return false
		}
	}

	return true
}

func compareVNetMetaAPIVnet(vnetMeta *k8sv1alpha1.VNetMeta, apiVnet *vnet.VNetDetailed) bool {
	if ok := compareVNetMetaAPIVnetSites(vnetMeta.Spec.Sites, apiVnet.Sites); !ok {
		return false
	}
	if ok := compareVNetMetaAPIVnetGateways(vnetMeta.Spec.Gateways, apiVnet.Gateways); !ok {
		return false
	}
	vlanAuto := false
	for _, port := range vnetMeta.Spec.Members {
		if port.Vlan == "auto" {
			vlanAuto = true
			break
		}
	}
	if ok := compareVNetMetaAPIVnetMembers(vnetMeta.Spec.Members, apiVnet.Ports); !ok && !vlanAuto {
		return false
	}

	if vnetMeta.Spec.VnetName != apiVnet.Name {
		return false
	}

	if vnetMeta.Spec.Owner != apiVnet.Tenant.Name {
		fmt.Println(vnetMeta.Spec.Owner, apiVnet.Tenant.Name)
		return false
	}

	if ok := compareVNetMetaAPIVnetTenants(vnetMeta.Spec.Tenants, apiVnet.GuestTenants); !ok {
		return false
	}

	if vnetMeta.Spec.State != apiVnet.State {
		return false
	}

	return true
}

func findGatewayDuplicates(items []k8sv1alpha1.VNetGateway) (string, bool) {
	tmpMap := make(map[string]int)
	for _, s := range items {
		str := s.Prefix
		tmpMap[str]++
		if tmpMap[str] > 1 {
			return str, true
		}
	}
	return "", false
}

func vnetCompareFieldsForNewMeta(vnet *k8sv1alpha1.VNet, vnetMeta *k8sv1alpha1.VNetMeta) bool {
	imported := false
	reclaim := false
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return vnet.GetGeneration() != vnetMeta.Spec.VnetCRGeneration || imported != vnetMeta.Spec.Imported || reclaim != vnetMeta.Spec.Reclaim
}

func vnetMustUpdateAnnotations(vnet *k8sv1alpha1.VNet) bool {
	update := false
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func vnetUpdateDefaultAnnotations(vnet *k8sv1alpha1.VNet) {
	imported := "false"
	reclaim := "delete"
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := vnet.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	vnet.SetAnnotations(annotations)
}
