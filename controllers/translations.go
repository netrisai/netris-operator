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

	vnetMeta := &k8sv1alpha1.VNetMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(vnet.GetUID()),
			Namespace: "default",
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.VNetMetaSpec{
			Name:     string(vnet.GetUID()),
			VnetName: vnet.Name,
			Sites:    sitesList,
			OwnerID:  tenantID,
			Tenants:  []int{}, // AAAAAAA
			Gateways: apiGateways,
			Members:  prts,

			VaMode:       false,
			VaNativeVLAN: 1,
			VaVLANs:      "",
		},
	}

	vnetMeta.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

	return vnetMeta, nil
}

// VnetMetaToNetris converts the k8s VNet resource to Netris type and used for add the VNet for Netris API.
func VnetMetaToNetris(vnetMeta *k8sv1alpha1.VNetMeta) (*api.APIVNetAdd, error) {
	ports := []k8sv1alpha1.VNetMetaMember{}
	siteNames := []string{}
	apiGateways := []api.APIVNetGateway{}

	for _, site := range vnetMeta.Spec.Sites {
		siteNames = append(siteNames, site.Name)
	}
	for _, port := range vnetMeta.Spec.Members {
		ports = append(ports, port)
	}
	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateways = append(apiGateways, api.APIVNetGateway{
			Gateway:  gateway.Gateway,
			GwLength: gateway.GwLength,
			ID:       gateway.ID,
			Version:  gateway.Version,
		})
	}

	prts := getPorts(ports)

	sites := getSites(siteNames)
	siteIDs := []int{}
	for _, id := range sites {
		siteIDs = append(siteIDs, id)
	}

	vnetAdd := &api.APIVNetAdd{
		Name:         vnetMeta.Spec.VnetName,
		Sites:        siteIDs,
		Owner:        vnetMeta.Spec.OwnerID,
		Tenants:      []int{}, // AAAAAAA
		Gateways:     apiGateways,
		Members:      prts.String(),
		VaMode:       false,
		VaNativeVLAN: 1,
		VaVLANs:      "",
	}

	return vnetAdd, nil
}

// VnetMetaToNetrisUpdate converts the k8s VNet resource to Netris type and used for update the VNet for Netris API.
func VnetMetaToNetrisUpdate(vnetMeta *k8sv1alpha1.VNetMeta) (*api.APIVNetUpdate, error) {
	ports := []k8sv1alpha1.VNetMetaMember{}

	apiGateways := []api.APIVNetGateway{}

	for _, port := range vnetMeta.Spec.Members {
		ports = append(ports, port)
	}
	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateways = append(apiGateways, api.APIVNetGateway{
			Gateway:  gateway.Gateway,
			GwLength: gateway.GwLength,
			ID:       gateway.ID,
			Version:  gateway.Version,
		})
	}

	prts := getPorts(ports)

	siteIDs := []int{}
	for _, site := range vnetMeta.Spec.Sites {
		siteIDs = append(siteIDs, site.ID)
	}

	vnetUpdate := &api.APIVNetUpdate{
		ID:           vnetMeta.Spec.ID,
		Name:         vnetMeta.Spec.VnetName,
		Sites:        siteIDs,
		Owner:        vnetMeta.Spec.OwnerID,
		Tenants:      []int{}, // AAAAAAA
		Gateways:     apiGateways,
		Members:      prts.String(),
		VaMode:       false,
		VaNativeVLAN: "1",
		VaVLANs:      "",
	}

	return vnetUpdate, nil
}

func compareVNetMetaAPIVnetGateways(vnetMetaGateways []k8sv1alpha1.VNetMetaGateway, apiVnetGateways []api.APIVNetGateway) bool {
	k8sGateways := make(map[string]k8sv1alpha1.VNetMetaGateway)
	for _, gateway := range vnetMetaGateways {
		k8sGateways[fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength)] = gateway
	}

	netrisGateways := make(map[string]api.APIVNetGateway)
	for _, gateway := range apiVnetGateways {
		netrisGateways[fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength)] = gateway
	}

	for address := range k8sGateways {
		if _, ok := netrisGateways[address]; !ok {
			return false
		}
	}

	return true
}

func compareVNetMetaAPIVnetMembers(vnetMetaMembers []k8sv1alpha1.VNetMetaMember, apiVnetMembers []api.APIVNetInfoMember) bool {
	k8sMembers := make(map[int]k8sv1alpha1.VNetMetaMember)
	for _, member := range vnetMetaMembers {
		k8sMembers[member.PortID] = member
	}

	netrisMembers := make(map[int]api.APIVNetInfoMember)
	for _, member := range apiVnetMembers {
		netrisMembers[member.PortID] = member
	}

	for portID := range k8sMembers {
		if _, ok := netrisMembers[portID]; !ok {
			return false
		}
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

	return true
}
