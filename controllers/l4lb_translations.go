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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VnetToVnetMeta converts the VNet resource to VNetMeta type and used for add the VNet for Netris API.
func (r *L4LBReconciler) L4LBToL4LBMeta(l4lb *k8sv1alpha1.L4LB) (*k8sv1alpha1.L4LBMeta, error) {
	tenantID := 0
	siteID := 0
	var state string
	var timeout string
	path := l4lb.Spec.Check.RequestPath

	if site, ok := NStorage.SitesStorage.FindByName(l4lb.Spec.Site); ok {
		siteID = site.ID
	}

	tenant, ok := NStorage.TenantsStorage.FindByName(l4lb.Spec.OwnerTenant)
	if !ok {
		return nil, fmt.Errorf("Tenant '%s' not found", l4lb.Spec.OwnerTenant)
	}
	tenantID = tenant.ID

	if l4lb.Spec.State == "" || l4lb.Spec.State == "active" {
		state = "enable"
	}

	if l4lb.Spec.Check.Timeout == 0 {
		timeout = string(rune(l4lb.Spec.Check.Timeout))
	}

	if l4lb.Spec.Check.Type == "tcp" {
		path = ""
	}

	imported := false
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/import"]; ok {
		if i == "true" {
			imported = true
		}
	}

	l4lbMeta := &k8sv1alpha1.L4LBMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(l4lb.GetUID()),
			Namespace: "default",
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.L4LBMetaSpec{
			Imported: imported,
			L4LBName: l4lb.Name,
			SiteID:   siteID,
			Tenant:   tenantID,
			Status:   state,
			Protocol: l4lb.Spec.Protocol,
			Port:     l4lb.Spec.Frontend.Port,
			IP:       l4lb.Spec.Frontend.IP,
			// Backend: , ?
			// HealthCheck: , ?
			Timeout:     timeout,
			RequestPath: path,
		},
	}

	return l4lbMeta, nil
}
