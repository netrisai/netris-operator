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

// VPCMetaSpec defines the desired state of VPCMeta
type VPCMetaSpec struct {
	Imported         bool     `json:"imported"`
	Reclaim          bool     `json:"reclaimPolicy"`
	VPCCRGeneration  int64    `json:"vpcGeneration"`
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	VPCName          string   `json:"vpcName"`
	AdminTenant      string   `json:"adminTenant"`
	AdminTenantID    int      `json:"adminTenantId"`
	GuestTenants     []string `json:"guestTenants"`
	GuestTenantIDs   []int    `json:"guestTenantIds"`
	Tags             []string `json:"tags"`
	IsSystem         bool     `json:"isSystem,omitempty"`
	IsDefault        bool     `json:"isDefault,omitempty"`
	VNI              int      `json:"vni,omitempty"`
}

// VPCMetaStatus defines the observed state of VPCMeta
type VPCMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VPCMeta is the Schema for the vpcmeta API
type VPCMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VPCMetaSpec   `json:"spec,omitempty"`
	Status VPCMetaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VPCMetaList contains a list of VPCMeta
type VPCMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VPCMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VPCMeta{}, &VPCMetaList{})
}

