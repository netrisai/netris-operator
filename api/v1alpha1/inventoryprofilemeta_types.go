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

// InventoryProfileMetaSpec defines the desired state of InventoryProfileMeta
type InventoryProfileMetaSpec struct {
	Imported                     bool   `json:"imported"`
	Reclaim                      bool   `json:"reclaimPolicy"`
	InventoryProfileCRGeneration int64  `json:"inventoryProfileGeneration"`
	ID                           int    `json:"id"`
	InventoryProfileName         string `json:"inventoryProfileName"`

	Description string `json:"description,omitempty"`

	Timezone string `json:"timezone"`

	AllowSSHFromIPv4 []string                     `json:"allowSshFromIpv4,omitempty"`
	AllowSSHFromIPv6 []string                     `json:"allowSshFromIpv6,omitempty"`
	NTPServers       []string                     `json:"ntpServers,omitempty"`
	DNSServers       []string                     `json:"dnsServers,omitempty"`
	CustomRules      []InventoryProfileCustomRule `json:"customRules,omitempty"`
}

// InventoryProfileMetaStatus defines the observed state of InventoryProfileMeta
type InventoryProfileMetaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InventoryProfileMeta is the Schema for the inventoryprofilemeta API
type InventoryProfileMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InventoryProfileMetaSpec   `json:"spec,omitempty"`
	Status InventoryProfileMetaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InventoryProfileMetaList contains a list of InventoryProfileMeta
type InventoryProfileMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InventoryProfileMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InventoryProfileMeta{}, &InventoryProfileMetaList{})
}
