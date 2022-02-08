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

// SoftgateMetaSpec defines the desired state of SoftgateMeta
type SoftgateMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Imported             bool   `json:"imported"`
	Reclaim              bool   `json:"reclaimPolicy"`
	SoftgateCRGeneration int64  `json:"softgateGeneration"`
	ID                   int    `json:"id"`
	SoftgateName         string `json:"softgateName"`

	TenantID    int    `json:"tenantid,omitempty"`
	Description string `json:"description,omitempty"`
	SiteID      int    `json:"siteid,omitempty"`
	ProfileID   int    `json:"profileid,omitempty"`
	MainIP      string `json:"mainIp,omitempty"`
	MgmtIP      string `json:"mgmtIp,omitempty"`
}

// SoftgateMetaStatus defines the observed state of SoftgateMeta
type SoftgateMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SoftgateMeta is the Schema for the softgatemeta API
type SoftgateMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SoftgateMetaSpec   `json:"spec,omitempty"`
	Status SoftgateMetaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SoftgateMetaList contains a list of SoftgateMeta
type SoftgateMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SoftgateMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SoftgateMeta{}, &SoftgateMetaList{})
}
