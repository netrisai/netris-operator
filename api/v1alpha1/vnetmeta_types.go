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

// VNetMetaSpec defines the desired state of VNetMeta
type VNetMetaSpec struct {
	Gateways     []VNetMetaGateway `json:"gateways"`
	ID           int               `json:"id"`
	Members      []VNetMetaMember  `json:"members"`
	Name         string            `json:"name"`
	VnetName     string            `json:"vnetName"`
	OwnerID      int               `json:"ownerid"`
	Owner        string            `json:"owner"`
	Provisioning int               `json:"provisioning"`
	Sites        []VNetMetaSite    `json:"sites"`
	State        string            `json:"state"`
	Tenants      []int             `json:"tenants"`
	VaMode       bool              `json:"vaMode"`
	VaNativeVLAN int               `json:"vaNativeVlan"`
	VaVLANs      string            `json:"vaVlans"`
}

// VNetMetaSite .
type VNetMetaSite struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// VNetMetaMember .
type VNetMetaMember struct {
	ChildPort      int    `json:"childPort"`
	LACP           string `json:"lacp"`
	MemberState    string `json:"member_state"`
	ParentPort     int    `json:"parentPort"`
	PortIsUntagged bool   `json:"portIsUntagged"`
	PortID         int    `json:"port_id"`
	PortName       string `json:"port_name"`
	TenantID       int    `json:"tenant_id"`
	VLANID         int    `json:"vlan_id"`
}

// VNetMetaGateway .
type VNetMetaGateway struct {
	Gateway  string `json:"gateway"`
	GwLength int    `json:"gwLength"`
	ID       int    `json:"id,omitempty"`
	VaVLANID int    `json:"vaVlanId,omitempty"`
	Nos      string `json:"nos,omitempty"`
	Version  string `json:"version,omitempty"`
}

// VNetMetaStatus defines the observed state of VNetMeta
type VNetMetaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VNetMeta is the Schema for the vnetmeta API
type VNetMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VNetMetaSpec   `json:"spec,omitempty"`
	Status VNetMetaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VNetMetaList contains a list of VNetMeta
type VNetMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VNetMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VNetMeta{}, &VNetMetaList{})
}
