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
	"github.com/netrisai/netriswebapi/v2/types/vpc"
	"github.com/r3labs/diff/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VPCToVPCMeta converts the VPC resource to VPCMeta type and used for add the VPC for Netris API.
func (r *VPCReconciler) VPCToVPCMeta(vpcCR *k8sv1alpha1.VPC) (*k8sv1alpha1.VPCMeta, error) {
	adminTenantID := 0
	if tenant, ok := r.NStorage.TenantsStorage.FindByName(vpcCR.Spec.AdminTenant); ok {
		adminTenantID = tenant.ID
	} else {
		return nil, fmt.Errorf("'%s' admin tenant not found", vpcCR.Spec.AdminTenant)
	}

	guestTenantIDs := []int{}
	guestTenantNames := []string{}
	for _, tenantName := range vpcCR.Spec.GuestTenants {
		if tenant, ok := r.NStorage.TenantsStorage.FindByName(tenantName); ok {
			guestTenantIDs = append(guestTenantIDs, tenant.ID)
			guestTenantNames = append(guestTenantNames, tenant.Name)
		} else {
			return nil, fmt.Errorf("'%s' guest tenant not found", tenantName)
		}
	}

	imported := false
	reclaim := false
	if i, ok := vpcCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := vpcCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	vpcMeta := &k8sv1alpha1.VPCMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(vpcCR.GetUID()),
			Namespace: vpcCR.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.VPCMetaSpec{
			Imported:        imported,
			Reclaim:         reclaim,
			Name:            string(vpcCR.GetUID()),
			VPCName:         vpcCR.Name,
			AdminTenant:     vpcCR.Spec.AdminTenant,
			AdminTenantID:   adminTenantID,
			GuestTenants:    guestTenantNames,
			GuestTenantIDs:  guestTenantIDs,
			Tags:            normalizeVPCTags(vpcCR.Spec.Tags),
		},
	}

	return vpcMeta, nil
}

// VPCMetaToNetris converts the k8s VPC resource to Netris type and used for add the VPC for Netris API.
func (r *VPCMetaReconciler) VPCMetaToNetris(vpcMeta *k8sv1alpha1.VPCMeta) (*vpc.VPCw, error) {
	adminTenant := vpc.AdminTenant{ID: vpcMeta.Spec.AdminTenantID, Name: vpcMeta.Spec.AdminTenant}

	guestTenants := []vpc.GuestTenant{}
	for i, tenantID := range vpcMeta.Spec.GuestTenantIDs {
		guestTenants = append(guestTenants, vpc.GuestTenant{
			ID:   tenantID,
			Name: vpcMeta.Spec.GuestTenants[i],
		})
	}

	vpcAdd := &vpc.VPCw{
		Name:        vpcMeta.Spec.VPCName,
		AdminTenant: adminTenant,
		GuestTenant: guestTenants,
		Tags:        normalizeVPCTags(vpcMeta.Spec.Tags),
	}

	return vpcAdd, nil
}

// VPCMetaToNetrisUpdate converts the k8s VPC resource to Netris type and used for update the VPC for Netris API.
func VPCMetaToNetrisUpdate(vpcMeta *k8sv1alpha1.VPCMeta) (*vpc.VPCw, error) {
	adminTenant := vpc.AdminTenant{ID: vpcMeta.Spec.AdminTenantID, Name: vpcMeta.Spec.AdminTenant}

	guestTenants := []vpc.GuestTenant{}
	for i, tenantID := range vpcMeta.Spec.GuestTenantIDs {
		guestTenants = append(guestTenants, vpc.GuestTenant{
			ID:   tenantID,
			Name: vpcMeta.Spec.GuestTenants[i],
		})
	}

	vpcUpdate := &vpc.VPCw{
		Name:        vpcMeta.Spec.VPCName,
		AdminTenant: adminTenant,
		GuestTenant: guestTenants,
		Tags:        normalizeVPCTags(vpcMeta.Spec.Tags),
	}

	return vpcUpdate, nil
}

func compareVPCMetaAPIVPC(vpcMeta *k8sv1alpha1.VPCMeta, apiVPC *vpc.VPC) bool {
	if vpcMeta.Spec.VPCName != apiVPC.Name {
		return false
	}

	if vpcMeta.Spec.AdminTenantID != apiVPC.AdminTenant.ID {
		return false
	}

	if ok := compareVPCMetaAPIVPCGuestTenants(vpcMeta.Spec.GuestTenantIDs, apiVPC.GuestTenant); !ok {
		return false
	}

	if ok := compareVPCMetaAPIVPCTags(vpcMeta.Spec.Tags, apiVPC.Tags); !ok {
		return false
	}

	return true
}

func compareVPCMetaAPIVPCGuestTenants(vpcMetaTenantIDs []int, apiVPCTenants []vpc.GuestTenant) bool {
	tenantIDList := []int{}
	for _, tenant := range apiVPCTenants {
		tenantIDList = append(tenantIDList, tenant.ID)
	}
	changelog, _ := diff.Diff(vpcMetaTenantIDs, tenantIDList)
	return len(changelog) <= 0
}

func compareVPCMetaAPIVPCTags(vpcMetaTags []string, apiVPCTags []string) bool {
	normalizedMetaTags := normalizeVPCTags(vpcMetaTags)
	normalizedAPITags := normalizeVPCTags(apiVPCTags)
	changelog, _ := diff.Diff(normalizedMetaTags, normalizedAPITags)
	return len(changelog) <= 0
}

func normalizeVPCTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func vpcCompareFieldsForNewMeta(vpcCR *k8sv1alpha1.VPC, vpcMeta *k8sv1alpha1.VPCMeta) bool {
	imported := false
	reclaim := false
	if i, ok := vpcCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := vpcCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return vpcCR.GetGeneration() != vpcMeta.Spec.VPCCRGeneration || imported != vpcMeta.Spec.Imported || reclaim != vpcMeta.Spec.Reclaim
}

func vpcMustUpdateAnnotations(vpcCR *k8sv1alpha1.VPC) bool {
	update := false
	if i, ok := vpcCR.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := vpcCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func vpcUpdateDefaultAnnotations(vpcCR *k8sv1alpha1.VPC) {
	imported := "false"
	reclaim := "delete"
	if i, ok := vpcCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := vpcCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := vpcCR.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	vpcCR.SetAnnotations(annotations)
}

