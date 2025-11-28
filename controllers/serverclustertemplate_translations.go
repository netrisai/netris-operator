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
	"encoding/json"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netriswebapi/v2/types/serverclustertemplate"
	"github.com/r3labs/diff/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func compareServerClusterTemplateMetaAPIServerClusterTemplate(sctMeta *k8sv1alpha1.ServerClusterTemplateMeta, apiSCT *serverclustertemplate.ServerClusterTemplate) bool {
	if sctMeta.Spec.ServerClusterTemplateName != apiSCT.Name {
		return false
	}

	// Compare Vnets by marshaling to JSON and comparing
	metaVnetsJSON, err := json.Marshal(sctMeta.Spec.Vnets)
	if err != nil {
		return false
	}
	apiVnetsJSON, err := json.Marshal(apiSCT.Vnets)
	if err != nil {
		return false
	}

	// Compare as JSON strings
	var metaVnets, apiVnets interface{}
	if err := json.Unmarshal(metaVnetsJSON, &metaVnets); err != nil {
		return false
	}
	if err := json.Unmarshal(apiVnetsJSON, &apiVnets); err != nil {
		return false
	}

	changelog, _ := diff.Diff(metaVnets, apiVnets)
	return len(changelog) <= 0
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

