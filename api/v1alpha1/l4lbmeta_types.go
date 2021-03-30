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

// L4LBMetaSpec defines the desired state of L4LBMeta
type L4LBMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Imported         bool   `json:"imported"`
	L4LBCRGeneration int64  `json:"l4lbGeneration"`
	ID               int    `json:"id"`
	L4LBName         string `json:"l4lbName"`

	Tenant    int  `json:"tenantId"`
	SiteID    int  `json:"siteId"`
	Automatic bool `json:"automatic"`
	Internal  int  `json:"internal"`

	KubenetInfoString string `json:"kubenet_info"`

	Protocol string `json:"protocol"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`

	Status string `json:"status"`

	HealthCheck string `json:"healthCheck"`
	Timeout     string `json:"timeOut"`
	RequestPath string `json:"requestPath"`

	Backend []L4LBMetaLBBackend `json:"backendIps"`
}

// L4LBMetaLBBackend .
type L4LBMetaLBBackend struct {
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	Maintenance bool   `json:"maintenance"`
}

// L4LBMetaStatus defines the observed state of L4LBMeta
type L4LBMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// L4LBMeta is the Schema for the l4lbmeta API
type L4LBMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   L4LBMetaSpec   `json:"spec,omitempty"`
	Status L4LBMetaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// L4LBMetaList contains a list of L4LBMeta
type L4LBMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []L4LBMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&L4LBMeta{}, &L4LBMetaList{})
}
