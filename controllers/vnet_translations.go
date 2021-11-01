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
			return nil, fmt.Errorf("Invalid spec.state field")
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
func (r *VNetMetaReconciler) VnetMetaToNetris(vnetMeta *k8sv1alpha1.VNetMeta) (*vnet.VNetAdd, error) {
	apiGateways := []vnet.VNetAddGateway{}

	sites := []vnet.VNetAddSite{}

	for _, site := range vnetMeta.Spec.Sites {
		sites = append(sites, vnet.VNetAddSite{Name: site.Name})
	}
	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateways = append(apiGateways, vnet.VNetAddGateway{
			Prefix: fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength),
		})
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
		Ports:        k8sMemberToAPIMember(vnetMeta.Spec.Members),
		NativeVlan:   1,
	}

	return vnetAdd, nil
}

// VnetMetaToNetrisUpdate converts the k8s VNet resource to Netris type and used for update the VNet for Netris API.
func VnetMetaToNetrisUpdate(vnetMeta *k8sv1alpha1.VNetMeta) (*vnet.VNetUpdate, error) {
	apiGateways := []vnet.VNetUpdateGateway{}

	sites := []vnet.VNetUpdateSite{}

	for _, site := range vnetMeta.Spec.Sites {
		sites = append(sites, vnet.VNetUpdateSite{Name: site.Name})
	}
	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateways = append(apiGateways, vnet.VNetUpdateGateway{
			Prefix: fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength),
		})
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
		Ports:        k8sMemberToAPIMemberUpdate(vnetMeta.Spec.Members),
		NativeVlan:   1,
	}

	return vnetUpdate, nil
}

func compareVNetMetaAPIVnetGateways(vnetMetaGateways []k8sv1alpha1.VNetMetaGateway, apiVnetGateways []vnet.VNetDetailedGateway) bool {
	type gateway struct {
		Prefix string `diff:"gateway"`
	}

	vnetGateways := []gateway{}
	apiGateways := []gateway{}

	for _, g := range vnetMetaGateways {
		vnetGateways = append(vnetGateways, gateway{
			Prefix: fmt.Sprintf("%s/%d", g.Gateway, g.GwLength),
		})
	}

	for _, g := range apiVnetGateways {
		apiGateways = append(apiGateways, gateway{
			Prefix: g.Prefix,
		})
	}

	changelog, _ := diff.Diff(vnetGateways, apiGateways)

	return len(changelog) <= 0
}

func compareVNetMetaAPIVnetMembers(vnetMetaMembers []k8sv1alpha1.VNetMetaMember, apiVnetMembers []vnet.VNetDetailedPort) bool {
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
		})
	}

	for _, m := range apiVnetMembers {
		vlan, _ := strconv.Atoi(m.Vlan)
		apiMembers = append(apiMembers, member{
			PortID:   m.ID,
			TenantID: m.Tenant.ID,
			VLANID:   vlan,
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
		fmt.Println("apiVnet.Sites")
		return false
	}
	if ok := compareVNetMetaAPIVnetGateways(vnetMeta.Spec.Gateways, apiVnet.Gateways); !ok {
		fmt.Println("apiVnet.Gateways")
		return false
	}
	if ok := compareVNetMetaAPIVnetMembers(vnetMeta.Spec.Members, apiVnet.Ports); !ok {
		fmt.Println("apiVnet.Ports")
		return false
	}

	if vnetMeta.Spec.VnetName != apiVnet.Name {
		fmt.Println("apiVnet.Name ")
		return false
	}

	if vnetMeta.Spec.Owner != apiVnet.Tenant.Name {
		fmt.Println(vnetMeta.Spec.Owner, apiVnet.Tenant.Name)
		fmt.Println("Tenant.Name ")
		return false
	}

	if ok := compareVNetMetaAPIVnetTenants(vnetMeta.Spec.Tenants, apiVnet.GuestTenants); !ok {
		fmt.Println("apiVnet.GuestTenants")
		return false
	}

	if vnetMeta.Spec.State != apiVnet.State {
		return false
	}

	return true
}

func k8sMemberToAPIMember(portNames []k8sv1alpha1.VNetMetaMember) []vnet.VNetAddPort {
	members := []vnet.VNetAddPort{}
	for _, port := range portNames {
		members = append(members, vnet.VNetAddPort{
			// Port:   port.ChildPort,
			Lacp:  port.LACP,
			State: port.MemberState,
			ID:    port.PortID,
			Name:  port.PortName,
			Vlan:  strconv.Itoa(port.VLANID),
		})
	}
	return members
}

func k8sMemberToAPIMemberUpdate(portNames []k8sv1alpha1.VNetMetaMember) []vnet.VNetUpdatePort {
	members := []vnet.VNetUpdatePort{}
	for _, port := range portNames {
		members = append(members, vnet.VNetUpdatePort{
			// Port:   port.ChildPort,
			Lacp:  port.LACP,
			State: port.MemberState,
			ID:    port.PortID,
			Name:  port.PortName,
			Vlan:  strconv.Itoa(port.VLANID),
		})
	}
	return members
}

func findGatewayDuplicates(items []k8sv1alpha1.VNetGateway) (string, bool) {
	tmpMap := make(map[string]int)
	for _, s := range items {
		str := s.String()
		tmpMap[str] += 1
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
