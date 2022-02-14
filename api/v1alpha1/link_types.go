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

// LinkSpec defines the desired state of Link
type LinkSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinItems=2
	// +kubebuilder:validation:MaxItems=2
	Ports []LinkSpecPort `json:"ports"`
}

// LinkSpecPort .
type LinkSpecPort string

func (l LinkSpecPort) String() string {
	return string(l)
}

// LinkStatus defines the observed state of Link
type LinkStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Ports   string `json:"ports,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ports",type=string,JSONPath=`.status.ports`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Link is the Schema for the links API
type Link struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LinkSpec   `json:"spec,omitempty"`
	Status LinkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LinkList contains a list of Link
type LinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Link `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Link{}, &LinkList{})
}
