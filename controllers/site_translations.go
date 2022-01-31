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
	"github.com/netrisai/netriswebapi/v1/types/site"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var routingProfiles = map[string]int{
	"default":     1,
	"default_agg": 2,
	"full":        3,
}

// SiteToSiteMeta converts the Site resource to SiteMeta type and used for add the Site for Netris API.
func (r *SiteReconciler) SiteToSiteMeta(site *k8sv1alpha1.Site) (*k8sv1alpha1.SiteMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := site.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := site.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	siteMeta := &k8sv1alpha1.SiteMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(site.GetUID()),
			Namespace: site.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.SiteMetaSpec{
			Imported:            imported,
			Reclaim:             reclaim,
			SiteName:            site.Name,
			PublicASN:           site.Spec.PublicASN,
			RohASN:              site.Spec.RohASN,
			VMASN:               site.Spec.VMASN,
			RohRoutingProfileID: routingProfiles[site.Spec.RohRoutingProfile],
			SiteMesh:            site.Spec.SiteMesh,
			ACLDefaultPolicy:    site.Spec.ACLDefaultPolicy,
		},
	}

	return siteMeta, nil
}

func siteCompareFieldsForNewMeta(site *k8sv1alpha1.Site, siteMeta *k8sv1alpha1.SiteMeta) bool {
	imported := false
	reclaim := false
	if i, ok := site.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := site.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return site.GetGeneration() != siteMeta.Spec.SiteCRGeneration || imported != siteMeta.Spec.Imported || reclaim != siteMeta.Spec.Reclaim
}

func siteMustUpdateAnnotations(site *k8sv1alpha1.Site) bool {
	update := false
	if i, ok := site.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := site.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func siteUpdateDefaultAnnotations(site *k8sv1alpha1.Site) {
	imported := "false"
	reclaim := "delete"
	if i, ok := site.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := site.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := site.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	site.SetAnnotations(annotations)
}

// SiteMetaToNetris converts the k8s Site resource to Netris type and used for add the Site for Netris API.
func SiteMetaToNetris(siteMeta *k8sv1alpha1.SiteMeta) (*site.SiteAdd, error) {
	siteAdd := &site.SiteAdd{
		Name:                siteMeta.Spec.SiteName,
		PublicASN:           siteMeta.Spec.PublicASN,
		PhysicalInstanceASN: siteMeta.Spec.RohASN,
		VirtualInstanceASN:  siteMeta.Spec.VMASN,
		VPN:                 siteMeta.Spec.SiteMesh,
		ACLPolicy:           siteMeta.Spec.ACLDefaultPolicy,
		RoutingProfileID:    siteMeta.Spec.RohRoutingProfileID,
	}

	return siteAdd, nil
}

// SiteMetaToNetrisUpdate converts the k8s Site resource to Netris type and used for update the Site for Netris API.
func SiteMetaToNetrisUpdate(siteMeta *k8sv1alpha1.SiteMeta) (*site.SiteAdd, error) {
	siteAdd := &site.SiteAdd{
		ID:                  siteMeta.Spec.ID,
		Name:                siteMeta.Spec.SiteName,
		PublicASN:           siteMeta.Spec.PublicASN,
		PhysicalInstanceASN: siteMeta.Spec.RohASN,
		VirtualInstanceASN:  siteMeta.Spec.VMASN,
		VPN:                 siteMeta.Spec.SiteMesh,
		ACLPolicy:           siteMeta.Spec.ACLDefaultPolicy,
		RoutingProfileID:    siteMeta.Spec.RohRoutingProfileID,
	}

	return siteAdd, nil
}

func compareSiteMetaAPIESite(siteMeta *k8sv1alpha1.SiteMeta, apiSite *site.Site, u uniReconciler) bool {
	if apiSite.Name != siteMeta.Spec.SiteName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiSite.Name, "k8sValue", siteMeta.Spec.SiteName)
		return false
	}
	if apiSite.PublicASN != siteMeta.Spec.PublicASN {
		u.DebugLogger.Info("PublicASN changed", "netrisValue", apiSite.PublicASN, "k8sValue", siteMeta.Spec.PublicASN)
		return false
	}
	if apiSite.PhysicalInstanceAsn != siteMeta.Spec.RohASN {
		u.DebugLogger.Info("RohASN changed", "netrisValue", apiSite.PhysicalInstanceAsn, "k8sValue", siteMeta.Spec.RohASN)
		return false
	}
	if apiSite.VirtualInstanceASN != siteMeta.Spec.VMASN {
		u.DebugLogger.Info("VMASN changed", "netrisValue", apiSite.VirtualInstanceASN, "k8sValue", siteMeta.Spec.VMASN)
		return false
	}
	if apiSite.RoutingProfileID != siteMeta.Spec.RohRoutingProfileID {
		u.DebugLogger.Info("RoutingProfile changed", "netrisValue", apiSite.RoutingProfileID, "k8sValue", siteMeta.Spec.RohRoutingProfileID)
		return false
	}
	if apiSite.VPN != siteMeta.Spec.SiteMesh {
		u.DebugLogger.Info("SiteMesh changed", "netrisValue", apiSite.VPN, "k8sValue", siteMeta.Spec.SiteMesh)
		return false
	}
	if apiSite.ACLPolicy != siteMeta.Spec.ACLDefaultPolicy {
		u.DebugLogger.Info("ACLDefaultPolicy changed", "netrisValue", apiSite.ACLPolicy, "k8sValue", siteMeta.Spec.ACLDefaultPolicy)
		return false
	}

	return true
}
