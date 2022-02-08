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

// ControllerSpec defines the desired state of Controller
type ControllerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Tenant      string `json:"tenant,omitempty"`
	Description string `json:"description,omitempty"`
	Site        string `json:"site,omitempty"`

	// +kubebuilder:validation:Pattern=`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`
	MainIP string `json:"mainIp,omitempty"`
}

// ControllerStatus defines the observed state of Controller
type ControllerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenant`
// +kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.site`
// +kubebuilder:printcolumn:name="Main IP",type=string,JSONPath=`.spec.mainIp`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Controller is the Schema for the controllers API
type Controller struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControllerSpec   `json:"spec,omitempty"`
	Status ControllerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ControllerList contains a list of Controller
type ControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Controller `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Controller{}, &ControllerList{})
}
