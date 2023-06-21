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
	"github.com/netrisai/netriswebapi/v2/types/ipam"
	"github.com/r3labs/diff/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SubnetToSubnetMeta converts the Subnet resource to SubnetMeta type and used for add the Subnet for Netris API.
func (r *SubnetReconciler) SubnetToSubnetMeta(subnet *k8sv1alpha1.Subnet) (*k8sv1alpha1.SubnetMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := subnet.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := subnet.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	sites := []int{}
	for _, s := range subnet.Spec.Sites {
		if site, ok := r.NStorage.SitesStorage.FindByName(s); ok {
			sites = append(sites, site.ID)
		} else {
			return nil, fmt.Errorf("invalid site '%s'", s)
		}
	}

	tenantID := 0
	if tenant, ok := r.NStorage.TenantsStorage.FindByName(subnet.Spec.Tenant); ok {
		tenantID = tenant.ID
	} else {
		return nil, fmt.Errorf("invalid tenant '%s'", subnet.Spec.Tenant)
	}

	subnetMeta := &k8sv1alpha1.SubnetMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(subnet.GetUID()),
			Namespace: subnet.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.SubnetMetaSpec{
			Imported:       imported,
			Reclaim:        reclaim,
			SubnetName:     subnet.Name,
			Prefix:         subnet.Spec.Prefix,
			TenantID:       tenantID,
			Purpose:        subnet.Spec.Purpose,
			DefaultGateway: subnet.Spec.DefaultGateway,
			Sites:          sites,
		},
	}

	return subnetMeta, nil
}

func subnetCompareFieldsForNewMeta(subnet *k8sv1alpha1.Subnet, subnetMeta *k8sv1alpha1.SubnetMeta) bool {
	imported := false
	reclaim := false
	if i, ok := subnet.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := subnet.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return subnet.GetGeneration() != subnetMeta.Spec.SubnetCRGeneration || imported != subnetMeta.Spec.Imported || reclaim != subnetMeta.Spec.Reclaim
}

func subnetMustUpdateAnnotations(subnet *k8sv1alpha1.Subnet) bool {
	update := false
	if i, ok := subnet.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := subnet.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func subnetUpdateDefaultAnnotations(subnet *k8sv1alpha1.Subnet) {
	imported := "false"
	reclaim := "delete"
	if i, ok := subnet.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := subnet.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := subnet.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	subnet.SetAnnotations(annotations)
}

// SubnetMetaToNetris converts the k8s Subnet resource to Netris type and used for add the Subnet for Netris API.
func SubnetMetaToNetris(subnetMeta *k8sv1alpha1.SubnetMeta) (*ipam.Subnet, error) {
	sites := []ipam.IDName{}
	for _, site := range subnetMeta.Spec.Sites {
		sites = append(sites, ipam.IDName{ID: site})
	}
	subnetAdd := &ipam.Subnet{
		Name:           subnetMeta.Spec.SubnetName,
		Prefix:         subnetMeta.Spec.Prefix,
		Tenant:         ipam.IDName{ID: subnetMeta.Spec.TenantID},
		Purpose:        subnetMeta.Spec.Purpose,
		DefaultGateway: subnetMeta.Spec.DefaultGateway,
		Sites:          sites,
	}

	return subnetAdd, nil
}

// SubnetMetaToNetrisUpdate converts the k8s Subnet resource to Netris type and used for update the Subnet for Netris API.
func SubnetMetaToNetrisUpdate(subnetMeta *k8sv1alpha1.SubnetMeta) (*ipam.Subnet, error) {
	sites := []ipam.IDName{}
	for _, site := range subnetMeta.Spec.Sites {
		sites = append(sites, ipam.IDName{ID: site})
	}
	subnetAdd := &ipam.Subnet{
		Name:           subnetMeta.Spec.SubnetName,
		Prefix:         subnetMeta.Spec.Prefix,
		Tenant:         ipam.IDName{ID: subnetMeta.Spec.TenantID},
		Purpose:        subnetMeta.Spec.Purpose,
		DefaultGateway: subnetMeta.Spec.DefaultGateway,
		Sites:          sites,
	}

	return subnetAdd, nil
}

func compareSubnetMetaAPIESubnet(subnetMeta *k8sv1alpha1.SubnetMeta, apiSubnet *ipam.IPAM, u uniReconciler) bool {
	if apiSubnet.Name != subnetMeta.Spec.SubnetName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiSubnet.Name, "k8sValue", subnetMeta.Spec.SubnetName)
		return false
	}

	if apiSubnet.Prefix != subnetMeta.Spec.Prefix {
		u.DebugLogger.Info("Prefix changed", "netrisValue", apiSubnet.Prefix, "k8sValue", subnetMeta.Spec.Prefix)
		return false
	}

	if apiSubnet.Purpose != subnetMeta.Spec.Purpose {
		u.DebugLogger.Info("Purpose changed", "netrisValue", apiSubnet.Purpose, "k8sValue", subnetMeta.Spec.Purpose)
		return false
	}

	if apiSubnet.DefaultGateway != subnetMeta.Spec.DefaultGateway {
		u.DebugLogger.Info("DefaultGateway changed", "netrisValue", apiSubnet.DefaultGateway, "k8sValue", subnetMeta.Spec.DefaultGateway)
		return false
	}

	if ok := compareSubnetMetaSiteAPISubnetSite(subnetMeta.Spec.Sites, apiSubnet.Sites); !ok {
		u.DebugLogger.Info("Sites changed", "netrisValue", apiSubnet.Sites, "k8sValue", subnetMeta.Spec.Sites)
		return false
	}

	return true
}

func compareSubnetMetaSiteAPISubnetSite(subnetMetaSites []int, apiSubnetSites []ipam.IDName) bool {
	apiSites := []int{}

	for _, s := range apiSubnetSites {
		apiSites = append(apiSites, s.ID)
	}

	changelog, _ := diff.Diff(subnetMetaSites, apiSites)

	return len(changelog) <= 0
}
