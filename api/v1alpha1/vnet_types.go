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

type VNetGateways struct {
	Id         int    `json:"id,omitempty"`
	Gateway    string `json:"gateway"`
	Gw_length  string `json:"gw_length"`
	Version    string `json:"version"`
	Va_vlan_id int    `json:"va_vlan_id,omitempty"`
}

// type VNetMembers struct {
// 	Port_id        int    `json:"port_id"`
// 	Vlan_id        string `json:"vlan_id"`
// 	Tenant_id      int    `json:"tenant_id"`
// 	ChildPort      int    `json:"childPort"`
// 	ParentPort     int    `json:"parentPort"`
// 	Member_state   string `json:"member_state"`
// 	Lacp           string `json:"lacp"`
// 	Port_name      string `json:"port_name"`
// 	PortIsUntagged bool   `json:"portIsUntagged"`
// }

// VNetSpec defines the desired state of VNet
type VNetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ID   int    `json:"id,omitempty"`
	Name string `json:"name"`
	// +kubebuilder:validation:Minimum=1
	Vxlan_id       int            `json:"vxlan_id,omitempty"`
	Mac_address    string         `json:"mac_address,omitempty"`
	MembersCount   int            `json:"membersCount,omitempty"`
	State          string         `json:"state"`
	Provisioning   int            `json:"provisioning"`
	Create_date    string         `json:"create_date,omitempty"`
	Modified_date  string         `json:"modifiedDate,omitempty"`
	Owner          int            `json:"owner"`
	Va_mode        bool           `json:"va_mode"`
	Va_native_vlan int            `json:"va_native_vlan"`
	Va_vlans       string         `json:"va_vlans"`
	Tenants        []int          `json:"tenants"`
	Sites          []int          `json:"sites"`
	Gateways       []VNetGateways `json:"gateways"`
	Members        string         `json:"members"`
}

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
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VNetSpec   `json:"spec"`
	Status VNetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VNetList contains a list of VNet
type VNetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VNet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VNet{}, &VNetList{})
}
