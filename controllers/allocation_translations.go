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
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netriswebapi/v2/types/ipam"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AllocationToAllocationMeta converts the Allocation resource to AllocationMeta type and used for add the Allocation for Netris API.
func (r *AllocationReconciler) AllocationToAllocationMeta(allocation *k8sv1alpha1.Allocation) (*k8sv1alpha1.AllocationMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := allocation.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := allocation.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	allocationMeta := &k8sv1alpha1.AllocationMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(allocation.GetUID()),
			Namespace: allocation.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.AllocationMetaSpec{
			Imported:       imported,
			Reclaim:        reclaim,
			AllocationName: allocation.Name,
			Prefix:         allocation.Spec.Prefix,
			Tenant:         allocation.Spec.Tenant,
		},
	}

	return allocationMeta, nil
}

func allocationCompareFieldsForNewMeta(allocation *k8sv1alpha1.Allocation, allocationMeta *k8sv1alpha1.AllocationMeta) bool {
	imported := false
	reclaim := false
	if i, ok := allocation.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := allocation.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return allocation.GetGeneration() != allocationMeta.Spec.AllocationCRGeneration || imported != allocationMeta.Spec.Imported || reclaim != allocationMeta.Spec.Reclaim
}

func allocationMustUpdateAnnotations(allocation *k8sv1alpha1.Allocation) bool {
	update := false
	if i, ok := allocation.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := allocation.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func allocationUpdateDefaultAnnotations(allocation *k8sv1alpha1.Allocation) {
	imported := "false"
	reclaim := "delete"
	if i, ok := allocation.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := allocation.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := allocation.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	allocation.SetAnnotations(annotations)
}

// AllocationMetaToNetris converts the k8s Allocation resource to Netris type and used for add the Allocation for Netris API.
func AllocationMetaToNetris(allocationMeta *k8sv1alpha1.AllocationMeta) (*ipam.Allocation, error) {
	allocationAdd := &ipam.Allocation{
		Name:   allocationMeta.Spec.AllocationName,
		Prefix: allocationMeta.Spec.Prefix,
		Tenant: ipam.IDName{Name: allocationMeta.Spec.Tenant},
	}

	return allocationAdd, nil
}

// AllocationMetaToNetrisUpdate converts the k8s Allocation resource to Netris type and used for update the Allocation for Netris API.
func AllocationMetaToNetrisUpdate(allocationMeta *k8sv1alpha1.AllocationMeta) (*ipam.Allocation, error) {
	allocationAdd := &ipam.Allocation{
		Name:   allocationMeta.Spec.AllocationName,
		Prefix: allocationMeta.Spec.Prefix,
		Tenant: ipam.IDName{Name: allocationMeta.Spec.Tenant},
	}

	return allocationAdd, nil
}

func compareAllocationMetaAPIEAllocation(allocationMeta *k8sv1alpha1.AllocationMeta, apiAllocation *ipam.IPAM, u uniReconciler) bool {
	if apiAllocation.Name != allocationMeta.Spec.AllocationName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiAllocation.Name, "k8sValue", allocationMeta.Spec.AllocationName)
		return false
	}
	if apiAllocation.Prefix != allocationMeta.Spec.Prefix {
		u.DebugLogger.Info("Prefix changed", "netrisValue", apiAllocation.Name, "k8sValue", allocationMeta.Spec.AllocationName)
		return false
	}

	return true
}
