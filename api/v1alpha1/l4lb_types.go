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

// L4LBSpec defines the desired state of L4LB
type L4LBSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Enum=active;disabled
	State string `json:"state,omitempty"`

	Check       L4LBCheck `json:"check,omitempty"`
	OwnerTenant string    `json:"ownerTenant"`
	Site        string    `json:"site"`

	// +kubebuilder:validation:Enum=tcp;udp
	Protocol string `json:"protocol,omitempty"`

	Frontend L4LBFrontend  `json:"frontend"`
	Backend  []L4LBBackend `json:"backend"`
}

// L4LBCheck .
type L4LBCheck struct {
	// +kubebuilder:validation:Enum=tcp;http
	Type        string `json:"type,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
	RequestPath string `json:"requestPath,omitempty"`
}

// L4LBBackend .
// +kubebuilder:validation:Pattern=`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]):([1-9]|[1-9][0-9]{1,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-4])$`
type L4LBBackend string

// L4LBFrontend .
type L4LBFrontend struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65534
	Port int `json:"port"`

	// +kubebuilder:validation:Pattern=`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`
	IP string `json:"ip,omitempty"`

	// +kubebuilder:validation:Pattern=`^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\/([0-9]|[12]\d|3[0-2]))$`
	Subnet string `json:"subnet,omitempty"`
}

// L4LBStatus defines the observed state of L4LB
type L4LBStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// L4LB is the Schema for the l4lbs API
type L4LB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   L4LBSpec   `json:"spec,omitempty"`
	Status L4LBStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// L4LBList contains a list of L4LB
type L4LBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []L4LB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&L4LB{}, &L4LBList{})
}
