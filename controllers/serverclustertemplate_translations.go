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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netriswebapi/v2/types/serverclustertemplate"
)

// ServerClusterTemplateToServerClusterTemplateMeta converts the ServerClusterTemplate resource to ServerClusterTemplateMeta type.
func (r *ServerClusterTemplateReconciler) ServerClusterTemplateToServerClusterTemplateMeta(sctCR *k8sv1alpha1.ServerClusterTemplate) (*k8sv1alpha1.ServerClusterTemplateMeta, error) {
	imported := false
	reclaim := false
	if i, ok := sctCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := sctCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	sctMeta := &k8sv1alpha1.ServerClusterTemplateMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(sctCR.GetUID()),
			Namespace: sctCR.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.ServerClusterTemplateMetaSpec{
			Imported:                        imported,
			Reclaim:                         reclaim,
			Name:                            string(sctCR.GetUID()),
			ServerClusterTemplateName:       sctCR.Name,
			Vnets:                           sctCR.Spec.Vnets,
			ServerClusterTemplateCRGeneration: sctCR.GetGeneration(),
		},
	}

	return sctMeta, nil
}

// ServerClusterTemplateMetaToNetris converts the ServerClusterTemplateMeta resource to Netris ServerClusterTemplateW type for adding.
func ServerClusterTemplateMetaToNetris(sctMeta *k8sv1alpha1.ServerClusterTemplateMeta) (*serverclustertemplate.ServerClusterTemplateW, error) {
	sctAdd := &serverclustertemplate.ServerClusterTemplateW{
		Name:  sctMeta.Spec.ServerClusterTemplateName,
		Vnets: sctMeta.Spec.Vnets,
	}
	return sctAdd, nil
}

// ServerClusterTemplateMetaToNetrisUpdate converts the ServerClusterTemplateMeta resource to Netris ServerClusterTemplateW type for updating.
func ServerClusterTemplateMetaToNetrisUpdate(sctMeta *k8sv1alpha1.ServerClusterTemplateMeta) (*serverclustertemplate.ServerClusterTemplateW, error) {
	sctUpdate := &serverclustertemplate.ServerClusterTemplateW{
		Name:  sctMeta.Spec.ServerClusterTemplateName,
		Vnets: sctMeta.Spec.Vnets,
	}
	return sctUpdate, nil
}

func compareServerClusterTemplateMetaAPIServerClusterTemplate(sctMeta *k8sv1alpha1.ServerClusterTemplateMeta, apiSCT *serverclustertemplate.ServerClusterTemplate, debugLogger logr.InfoLogger) bool {
	if sctMeta.Spec.ServerClusterTemplateName != apiSCT.Name {
		debugLogger.Info("Name changed", "metaName", sctMeta.Spec.ServerClusterTemplateName, "apiName", apiSCT.Name)
		return false
	}

	// Vnets field is ignored in comparison - API returns Vnets with IDs that are assigned by the API
	// and not present in the CR, causing false positives. Since Vnets cannot be updated via the API
	// when the template is in use, we skip comparing them.
	debugLogger.Info("Skipping Vnets comparison (field ignored)")
	
	return true
}

func serverClusterTemplateCompareFieldsForNewMeta(sctCR *k8sv1alpha1.ServerClusterTemplate, sctMeta *k8sv1alpha1.ServerClusterTemplateMeta) bool {
	imported := false
	reclaim := false
	if i, ok := sctCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := sctCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return sctCR.GetGeneration() != sctMeta.Spec.ServerClusterTemplateCRGeneration || imported != sctMeta.Spec.Imported || reclaim != sctMeta.Spec.Reclaim
}

func serverClusterTemplateMustUpdateAnnotations(sctCR *k8sv1alpha1.ServerClusterTemplate) bool {
	update := false
	if i, ok := sctCR.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := sctCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func serverClusterTemplateUpdateDefaultAnnotations(sctCR *k8sv1alpha1.ServerClusterTemplate) {
	imported := "false"
	reclaim := "delete"
	if i, ok := sctCR.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := sctCR.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := sctCR.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	sctCR.SetAnnotations(annotations)
}

