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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ControllerMetaSpec defines the desired state of ControllerMeta
type ControllerMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Imported               bool   `json:"imported"`
	Reclaim                bool   `json:"reclaimPolicy"`
	ControllerCRGeneration int64  `json:"controllerGeneration"`
	ID                     int    `json:"id"`
	ControllerName         string `json:"controllerName"`

	TenantID    int    `json:"tenant,omitempty"`
	Description string `json:"description,omitempty"`
	SiteID      int    `json:"site,omitempty"`
	MainIP      string `json:"mainIp,omitempty"`
}

// ControllerMetaStatus defines the observed state of ControllerMeta
type ControllerMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ControllerMeta is the Schema for the controllermeta API
type ControllerMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControllerMetaSpec   `json:"spec,omitempty"`
	Status ControllerMetaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ControllerMetaList contains a list of ControllerMeta
type ControllerMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControllerMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ControllerMeta{}, &ControllerMetaList{})
}
