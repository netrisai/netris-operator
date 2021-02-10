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
	Status string `json:"status"`
	Type   string `json:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VNet is the Schema for the vnets API
type VNet struct {
	// APIVersion        string `json:"apiVersion"`
	// Kind              string `json:"kind"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VNetSpec `json:"spec"`
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
	Owner        string     `json:"ownerTenant"`
	State        string     `json:"state,omitempty"`
	GuestTenants []string   `json:"guestTenants"`
	Sites        []VNetSite `json:"sites"`
}

// VNetSite .
type VNetSite struct {
	Name        string           `json:"name"`
	Gateways    []VNetGateway    `json:"gateways,omitempty"`
	SwitchPorts []VNetSwitchPort `json:"switchPorts,omitempty"`
}

// VNetGateway .
type VNetGateway struct {
	Gateway4 string `json:"gateway4,omitempty"`
	Gateway6 string `json:"gateway6,omitempty"`
}

// VNetSwitchPort .
type VNetSwitchPort struct {
	Name   string `json:"name"`
	VlanID int    `json:"vlanId,omitempty"`
	State  string `json:"state,omitempty"`
}

func init() {
	SchemeBuilder.Register(&VNet{}, &VNetList{})
}
