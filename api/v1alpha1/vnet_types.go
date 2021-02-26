/*
Copyright 2020.

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

// type VNetTenants struct {
// 	Tenant_id   int    `json:"tenant_id"`
// 	Tenant_name string `json:"tenant_name"`
// }

// VNetStatus defines the observed state of VNet
type VNetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Gateways",type=string,JSONPath=`.spec.sites[*].switchPorts[*].name`
// +kubebuilder:printcolumn:name="Sites",type=string,JSONPath=".spec.sites[*].name"
// +kubebuilder:printcolumn:name="Modified",type=date,JSONPath=`.metadata.managedFields[0].time`,priority=1
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.ownerTenant`
// +kubebuilder:printcolumn:name="Guest Tenants",type=string,JSONPath=`.spec.guestTenants`,priority=1
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// VNet is the Schema for the vnets API
type VNet struct {
	// APIVersion        string `json:"apiVersion"`
	// Kind              string `json:"kind"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VNetSpec   `json:"spec"`
	Status            VNetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VNetList contains a list of VNet
type VNetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VNet `json:"items"`
}

// VNetSpec .
type VNetSpec struct {
	Owner string `json:"ownerTenant"`

	// +kubebuilder:validation:Enum=active;disabled
	State        string     `json:"state,omitempty"`
	GuestTenants []string   `json:"guestTenants"`
	Sites        []VNetSite `json:"sites"`
}

// VNetSite .
type VNetSite struct {
	Name        string           `json:"name"`
	Gateways    []string         `json:"gateways,omitempty"`
	SwitchPorts []VNetSwitchPort `json:"switchPorts,omitempty"`
}

// VNetSwitchPort .
type VNetSwitchPort struct {
	Name string `json:"name"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4094
	VlanID int    `json:"vlanId,omitempty"`
	State  string `json:"state,omitempty"`
}

func init() {
	SchemeBuilder.Register(&VNet{}, &VNetList{})
}
