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

// SubnetMetaSpec defines the desired state of SubnetMeta
type SubnetMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Imported           bool   `json:"imported"`
	Reclaim            bool   `json:"reclaimPolicy"`
	SubnetCRGeneration int64  `json:"subnetGeneration"`
	ID                 int    `json:"id"`
	SubnetName         string `json:"subnetName"`

	Prefix         string `json:"prefix,omitempty"`
	TenantID       int    `json:"tenantid,omitempty"`
	Purpose        string `json:"purpose,omitempty"`
	DefaultGateway string `json:"defaultGateway,omitempty"`
	Sites          []int  `json:"sites,omitempty"`
}

// SubnetMetaStatus defines the observed state of SubnetMeta
type SubnetMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SubnetMeta is the Schema for the subnetmeta API
type SubnetMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetMetaSpec   `json:"spec,omitempty"`
	Status SubnetMetaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SubnetMetaList contains a list of SubnetMeta
type SubnetMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SubnetMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SubnetMeta{}, &SubnetMetaList{})
}
