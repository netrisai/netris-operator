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

// AllocationMetaSpec defines the desired state of AllocationMeta
type AllocationMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Imported               bool   `json:"imported"`
	Reclaim                bool   `json:"reclaimPolicy"`
	AllocationCRGeneration int64  `json:"allocationGeneration"`
	ID                     int    `json:"id"`
	AllocationName         string `json:"allocationName"`

	Prefix string `json:"prefix"`
	Tenant string `json:"tenant"`
}

// AllocationMetaStatus defines the observed state of AllocationMeta
type AllocationMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AllocationMeta is the Schema for the allocationmeta API
type AllocationMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AllocationMetaSpec   `json:"spec,omitempty"`
	Status AllocationMetaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AllocationMetaList contains a list of AllocationMeta
type AllocationMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AllocationMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AllocationMeta{}, &AllocationMetaList{})
}
