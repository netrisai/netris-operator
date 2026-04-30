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
	"github.com/netrisai/netriswebapi/v2/types/servercluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServerClusterToServerClusterMeta converts the ServerCluster resource to ServerClusterMeta type.
func (r *ServerClusterReconciler) ServerClusterToServerClusterMeta(scCR *k8sv1alpha1.ServerCluster) (*k8sv1alpha1.ServerClusterMeta, error) {
	imported := false
	reclaim := false
	if i, ok := scCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := scCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	adminID := 0
	if tenant, ok := r.NStorage.TenantsStorage.FindByName(scCR.Spec.Admin); ok {
		adminID = tenant.ID
	} else {
		return nil, fmt.Errorf("'%s' admin tenant not found", scCR.Spec.Admin)
	}

	siteID := 0
	if site, ok := r.NStorage.SitesStorage.FindByName(scCR.Spec.Site); ok {
		siteID = site.ID
	} else {
		return nil, fmt.Errorf("'%s' site not found", scCR.Spec.Site)
	}

	vpcID := 0
	if vpc, ok := r.NStorage.VPCStorage.FindByName(scCR.Spec.VPC); ok {
		vpcID = vpc.ID
	} else {
		return nil, fmt.Errorf("'%s' vpc not found", scCR.Spec.VPC)
	}

	templateID := 0
	templates, err := r.Cred.ServerClusterTemplate().Get()
	if err != nil {
		return nil, err
	}
	for _, template := range templates {
		if template.Name == scCR.Spec.Template {
			templateID = template.ID
			break
		}
	}
	if templateID == 0 && scCR.Spec.Template != "" {
		return nil, fmt.Errorf("'%s' template not found", scCR.Spec.Template)
	}

	scMeta := &k8sv1alpha1.ServerClusterMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(scCR.GetUID()),
			Namespace: scCR.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.ServerClusterMetaSpec{
			Imported:                  imported,
			Reclaim:                   reclaim,
			Name:                      string(scCR.GetUID()),
			ServerClusterName:         scCR.Name,
			AdminID:                   adminID,
			Admin:                     scCR.Spec.Admin,
			SiteID:                    siteID,
			Site:                      scCR.Spec.Site,
			VPCID:                     vpcID,
			VPC:                       scCR.Spec.VPC,
			TemplateID:                templateID,
			Template:                  scCR.Spec.Template,
			Tags:                      normalizeTags(scCR.Spec.Tags),
			ServerClusterCRGeneration: scCR.GetGeneration(),
		},
	}

	return scMeta, nil
}

// ServerClusterMetaToNetris converts the ServerClusterMeta resource to Netris ServerClusterW type for adding.
func ServerClusterMetaToNetris(scMeta *k8sv1alpha1.ServerClusterMeta) (*servercluster.ServerClusterW, error) {
	scAdd := &servercluster.ServerClusterW{
		Name:               scMeta.Spec.ServerClusterName,
		Admin:              servercluster.IDName{ID: scMeta.Spec.AdminID, Name: scMeta.Spec.Admin},
		Site:               servercluster.IDName{ID: scMeta.Spec.SiteID, Name: scMeta.Spec.Site},
		VPC:                servercluster.IDName{ID: scMeta.Spec.VPCID, Name: scMeta.Spec.VPC},
		SrvClusterTemplate: servercluster.IDName{ID: scMeta.Spec.TemplateID, Name: scMeta.Spec.Template},
		Tags:               normalizeTags(scMeta.Spec.Tags),
		Servers:            []servercluster.Servers{},
	}
	return scAdd, nil
}

// ServerClusterMetaToNetrisUpdate converts the ServerClusterMeta resource to Netris ServerClusterU type for updating.
func ServerClusterMetaToNetrisUpdate(scMeta *k8sv1alpha1.ServerClusterMeta) (*servercluster.ServerClusterU, error) {
	scUpdate := &servercluster.ServerClusterU{
		Tags:    normalizeTags(scMeta.Spec.Tags),
		Servers: []servercluster.Servers{},
	}
	return scUpdate, nil
}

func compareServerClusterMetaAPIServerCluster(scMeta *k8sv1alpha1.ServerClusterMeta, apiSC *servercluster.ServerCluster) bool {
	// Note: Update API only supports Tags and Servers, so we only compare those
	// Admin, Site, VPC, Template are set on creation and cannot be changed via Update API

	// Compare Tags
	apiTags := normalizeTags(apiSC.Tags)
	metaTags := normalizeTags(scMeta.Spec.Tags)

	if len(apiTags) != len(metaTags) {
		return false
	}

	// Check if all metaTags are in apiTags
	for _, tag := range metaTags {
		found := false
		for _, apiTag := range apiTags {
			if tag == apiTag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check if all apiTags are in metaTags
	for _, apiTag := range apiTags {
		found := false
		for _, tag := range metaTags {
			if apiTag == tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func serverClusterCompareFieldsForNewMeta(scCR *k8sv1alpha1.ServerCluster, scMeta *k8sv1alpha1.ServerClusterMeta) bool {
	imported := false
	reclaim := false
	if i, ok := scCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := scCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return scCR.GetGeneration() != scMeta.Spec.ServerClusterCRGeneration || imported != scMeta.Spec.Imported || reclaim != scMeta.Spec.Reclaim
}

func serverClusterMustUpdateAnnotations(scCR *k8sv1alpha1.ServerCluster) bool {
	update := false
	if i, ok := scCR.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := scCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func serverClusterUpdateDefaultAnnotations(scCR *k8sv1alpha1.ServerCluster) {
	imported := "false"
	reclaim := "delete"
	if i, ok := scCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := scCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := scCR.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	scCR.SetAnnotations(annotations)
}

