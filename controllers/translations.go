/*
Copyright 2020.

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
	api "github.com/netrisai/netrisapi"
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
		for _, port := range site.SwitchPorts {
			ports = append(ports, port)
		}
		for _, gateway := range site.Gateways {
			apiGateways = append(apiGateways, makeGateway(gateway))
		}
	}
	prts := getPortsMeta(ports)

	sites := getSites(siteNames)
	sitesList := []k8sv1alpha1.VNetMetaSite{}

	for name, id := range sites {
		sitesList = append(sitesList, k8sv1alpha1.VNetMetaSite{
			Name: name,
			ID:   id,
		})
	}

	tenantID := 0

	tenant, ok := NStorage.TenantsStorage.FindByName(vnet.Spec.Owner)
	if !ok {
		return nil, fmt.Errorf("Tenant '%s' not found", vnet.Spec.Owner)
	}
	tenantID = tenant.ID

	guestTenants := []int{}
	for _, guest := range vnet.Spec.GuestTenants {
		tenant, ok := NStorage.TenantsStorage.FindByName(guest)
		if !ok {

			return nil, fmt.Errorf("Guest tenant '%s' not found", guest)
		}
		guestTenants = append(guestTenants, tenant.ID)
	}

	state := "active"
	if len(vnet.Spec.State) > 0 {
		if !(vnet.Spec.State == "active" || vnet.Spec.State == "disabled") {
			return nil, fmt.Errorf("Invalid spec.state field")
		}
		state = vnet.Spec.State
	}

	imported := false
	if i, ok := vnet.GetAnnotations()["resource.k8s.netris.ai/import"]; ok {
		if i == "true" {
			imported = true
		}
	}

	vnetMeta := &k8sv1alpha1.VNetMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(vnet.GetUID()),
			Namespace: "default",
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.VNetMetaSpec{
			Imported:     imported,
			Name:         string(vnet.GetUID()),
			VnetName:     vnet.Name,
			Sites:        sitesList,
			State:        state,
			OwnerID:      tenantID,
			Tenants:      guestTenants,
			Gateways:     apiGateways,
			Members:      prts,
			Provisioning: 1,
			VaMode:       false,
			VaNativeVLAN: 1,
			VaVLANs:      "",
		},
	}

	return vnetMeta, nil
}

// VnetMetaToNetris converts the k8s VNet resource to Netris type and used for add the VNet for Netris API.
func VnetMetaToNetris(vnetMeta *k8sv1alpha1.VNetMeta) (*api.APIVNetAdd, error) {
	siteNames := []string{}
	apiGateways := []api.APIVNetGateway{}

	for _, site := range vnetMeta.Spec.Sites {
		siteNames = append(siteNames, site.Name)
	}
	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateways = append(apiGateways, api.APIVNetGateway{
			Gateway:  gateway.Gateway,
			GwLength: gateway.GwLength,
			ID:       gateway.ID,
			Version:  gateway.Version,
		})
	}

	sites := getSites(siteNames)
	siteIDs := []int{}
	for _, id := range sites {
		siteIDs = append(siteIDs, id)
	}

	vnetAdd := &api.APIVNetAdd{
		Name:         vnetMeta.Spec.VnetName,
		Sites:        siteIDs,
		Owner:        vnetMeta.Spec.OwnerID,
		State:        vnetMeta.Spec.State,
		Tenants:      vnetMeta.Spec.Tenants,
		Gateways:     apiGateways,
		Members:      k8sMemberToAPIMember(vnetMeta.Spec.Members).String(),
		VaMode:       false,
		VaNativeVLAN: 1,
		VaVLANs:      "",
		Provisioning: 1,
	}

	return vnetAdd, nil
}

// VnetMetaToNetrisUpdate converts the k8s VNet resource to Netris type and used for update the VNet for Netris API.
func VnetMetaToNetrisUpdate(vnetMeta *k8sv1alpha1.VNetMeta) (*api.APIVNetUpdate, error) {
	apiGateways := []api.APIVNetGateway{}

	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateways = append(apiGateways, api.APIVNetGateway{
			Gateway:  gateway.Gateway,
			GwLength: gateway.GwLength,
			ID:       gateway.ID,
			Version:  gateway.Version,
		})
	}

	siteIDs := []int{}
	for _, site := range vnetMeta.Spec.Sites {
		siteIDs = append(siteIDs, site.ID)
	}

	vnetUpdate := &api.APIVNetUpdate{
		ID:           vnetMeta.Spec.ID,
		Name:         vnetMeta.Spec.VnetName,
		Sites:        siteIDs,
		State:        vnetMeta.Spec.State,
		Owner:        vnetMeta.Spec.OwnerID,
		Tenants:      vnetMeta.Spec.Tenants,
		Gateways:     apiGateways,
		Members:      k8sMemberToAPIMember(vnetMeta.Spec.Members).String(),
		VaMode:       false,
		VaNativeVLAN: "1",
		VaVLANs:      "",
		Provisioning: 1,
	}

	return vnetUpdate, nil
}

func compareVNetMetaAPIVnetGateways(vnetMetaGateways []k8sv1alpha1.VNetMetaGateway, apiVnetGateways []api.APIVNetGateway) bool {

	type gateway struct {
		Gateway string `diff:"gateway"`
		Length  int    `diff:"gwLength"`
	}

	vnetGateways := []gateway{}
	apiGateways := []gateway{}

	for _, g := range vnetMetaGateways {
		vnetGateways = append(vnetGateways, gateway{
			Gateway: g.Gateway,
			Length:  g.GwLength,
		})
	}

	for _, g := range apiVnetGateways {
		apiGateways = append(apiGateways, gateway{
			Gateway: g.Gateway,
			Length:  g.GwLength,
		})
	}

	changelog, _ := diff.Diff(vnetGateways, apiGateways)

	if len(changelog) > 0 {
		return false
	}

	return true
}

func compareVNetMetaAPIVnetMembers(vnetMetaMembers []k8sv1alpha1.VNetMetaMember, apiVnetMembers []api.APIVNetInfoMember) bool {

	type member struct {
		PortID   int    `diff:"port_id"`
		TenantID int    `diff:"tenant_id"`
		VLANID   int    `diff:"vlan_id"`
		State    string `diff:"state"`
	}

	vnetMembers := []member{}
	apiMembers := []member{}

	for _, m := range vnetMetaMembers {
		vnetMembers = append(vnetMembers, member{
			PortID:   m.PortID,
			TenantID: m.TenantID,
			VLANID:   m.VLANID,
			State:    m.MemberState,
		})
	}

	for _, m := range apiVnetMembers {
		apiMembers = append(apiMembers, member{
			PortID:   m.PortID,
			TenantID: m.TenantID,
			VLANID:   m.VlanID,
			State:    m.MemberState,
		})
	}

	changelog, _ := diff.Diff(vnetMembers, apiMembers)

	if len(changelog) > 0 {
		return false
	}

	return true
}

func compareVNetMetaAPIVnetTenants(vnetMetaTenants []int, apiVnetTenants []int) bool {
	changelog, _ := diff.Diff(vnetMetaTenants, apiVnetTenants)

	if len(changelog) > 0 {
		return false
	}

	return true
}

func compareVNetMetaAPIVnetSites(vnetMetaSites []k8sv1alpha1.VNetMetaSite, apiVnetSites []int) bool {

	k8sSites := make(map[int]string)
	for _, site := range vnetMetaSites {
		k8sSites[site.ID] = ""
	}

	for _, siteID := range apiVnetSites {
		if _, ok := k8sSites[siteID]; !ok {
			return false
		}
	}

	return true
}

func compareVNetMetaAPIVnet(vnetMeta *k8sv1alpha1.VNetMeta, apiVnet *api.APIVNetInfo) bool {

	if ok := compareVNetMetaAPIVnetSites(vnetMeta.Spec.Sites, apiVnet.SitesID); !ok {
		return false
	}
	if ok := compareVNetMetaAPIVnetGateways(vnetMeta.Spec.Gateways, apiVnet.Gateways); !ok {
		return false
	}
	if ok := compareVNetMetaAPIVnetMembers(vnetMeta.Spec.Members, apiVnet.Members); !ok {
		return false
	}

	if vnetMeta.Spec.VnetName != apiVnet.Name {
		return false
	}

	if vnetMeta.Spec.OwnerID != apiVnet.Owner {
		return false
	}

	if ok := compareVNetMetaAPIVnetTenants(vnetMeta.Spec.Tenants, apiVnet.TenantsID); !ok {
		return false
	}

	apiVaMode := false
	if apiVnet.VaMode > 0 {
		apiVaMode = true
	}

	if vnetMeta.Spec.VaMode != apiVaMode {
		return false
	}

	if vnetMeta.Spec.VaVLANs != apiVnet.VaVlans {
		return false
	}

	if vnetMeta.Spec.State != apiVnet.State {
		return false
	}

	return true
}

func k8sMemberToAPIMember(portNames []k8sv1alpha1.VNetMetaMember) *api.APIVNetMembers {
	members := &api.APIVNetMembers{}
	for _, port := range portNames {
		members.Add(api.APIVNetMember{
			ChildPort:      port.ChildPort,
			LACP:           port.LACP,
			MemberState:    port.MemberState,
			ParentPort:     port.ParentPort,
			PortIsUntagged: port.PortIsUntagged,
			PortID:         port.PortID,
			PortName:       port.PortName,
			TenantID:       port.TenantID,
			VLANID:         port.VLANID,
		})
	}
	return members
}
