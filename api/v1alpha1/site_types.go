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

// SiteStatus defines the observed state of Site
type SiteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// SiteSpec defines the desired state of Site
type SiteSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65534
	PublicASN int `json:"publicAsn"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65534
	RohASN int `json:"rohAsn"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65534
	VmASN int `json:"vmAsn"`

	// +kubebuilder:validation:Enum=default;default_agg;full
	RohRoutingProfile string `json:"rohRoutingProfile"`

	// +kubebuilder:validation:Enum=disabled;hub;spoke;dspoke
	SiteMesh string `json:"siteMesh"`

	// +kubebuilder:validation:Enum=permit;deny
	AclDefaultPolicy string `json:"aclDefaultPolicy"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Public ASN",type=integer,JSONPath=`.spec.publicAsn`
// +kubebuilder:printcolumn:name="ROH ASN",type=integer,JSONPath=`.spec.rohAsn`
// +kubebuilder:printcolumn:name="VM ASN",type=integer,JSONPath=`.spec.vmAsn`
// +kubebuilder:printcolumn:name="ROH Routing Profile",type=string,JSONPath=`.spec.rohRoutingProfile`
// +kubebuilder:printcolumn:name="Site Mesh",type=string,JSONPath=`.spec.siteMesh`
// +kubebuilder:printcolumn:name="ACL Default Policy",type=string,JSONPath=`.spec.aclDefaultPolicy`

// Site is the Schema for the sites API
type Site struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SiteSpec   `json:"spec,omitempty"`
	Status SiteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SiteList contains a list of Site
type SiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Site `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Site{}, &SiteList{})
}
