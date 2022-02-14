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
	"github.com/netrisai/netriswebapi/v2/types/link"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LinkToLinkMeta converts the Link resource to LinkMeta type and used for add the Link for Netris API.
func (r *LinkReconciler) LinkToLinkMeta(link *k8sv1alpha1.Link) (*k8sv1alpha1.LinkMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := link.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := link.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	local := 0
	remote := 0

	if o, ok := r.NStorage.PortsStorage.FindByName(string(link.Spec.Ports[0])); ok {
		local = o.ID
	} else {
		return nil, fmt.Errorf("Couldn't find port %s", link.Spec.Ports[0])
	}
	if d, ok := r.NStorage.PortsStorage.FindByName(string(link.Spec.Ports[1])); ok {
		remote = d.ID
	} else {
		return nil, fmt.Errorf("Couldn't find port %s", link.Spec.Ports[1])
	}

	linkMeta := &k8sv1alpha1.LinkMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(link.GetUID()),
			Namespace: link.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.LinkMetaSpec{
			Imported: imported,
			Reclaim:  reclaim,
			LinkName: link.Name,
			Local:    local,
			Remote:   remote,
		},
	}

	return linkMeta, nil
}

func linkCompareFieldsForNewMeta(link *k8sv1alpha1.Link, linkMeta *k8sv1alpha1.LinkMeta) bool {
	imported := false
	reclaim := false
	if i, ok := link.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := link.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return link.GetGeneration() != linkMeta.Spec.LinkCRGeneration || imported != linkMeta.Spec.Imported || reclaim != linkMeta.Spec.Reclaim
}

func linkMustUpdateAnnotations(link *k8sv1alpha1.Link) bool {
	update := false
	if i, ok := link.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := link.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func linkUpdateDefaultAnnotations(link *k8sv1alpha1.Link) {
	imported := "false"
	reclaim := "delete"
	if i, ok := link.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := link.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := link.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	link.SetAnnotations(annotations)
}

// LinkMetaToNetris converts the k8s Link resource to Netris type and used for add the Link for Netris API.
func LinkMetaToNetris(linkMeta *k8sv1alpha1.LinkMeta) (*link.Link, error) {
	linkAdd := &link.Link{
		Local:  link.LinkIDName{ID: linkMeta.Spec.Local},
		Remote: link.LinkIDName{ID: linkMeta.Spec.Remote},
	}

	return linkAdd, nil
}

// LinkMetaToNetrisUpdate converts the k8s Link resource to Netris type and used for update the Link for Netris API.
func LinkMetaToNetrisUpdate(linkMeta *k8sv1alpha1.LinkMeta) (*link.Link, error) {
	linkAdd := &link.Link{
		Local:  link.LinkIDName{ID: linkMeta.Spec.Local},
		Remote: link.LinkIDName{ID: linkMeta.Spec.Remote},
	}

	return linkAdd, nil
}
