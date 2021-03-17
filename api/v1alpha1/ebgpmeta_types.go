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

// EBGPMetaSpec defines the desired state of EBGPMeta
type EBGPMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Imported         bool   `json:"imported"`
	EBGPCRGeneration int64  `json:"ebgpGeneration"`
	ID               int    `json:"id"`
	EBGPName         string `json:"ebgpName"`

	AllowasIn          int    `json:"allowas_in"`
	BgpPassword        string `json:"bgp_password"`
	Community          string `json:"community"`
	Description        string `json:"description"`
	InboundRouteMap    int    `json:"inboundRouteMap"`
	Internal           string `json:"internal"`
	IPVersion          string `json:"ip_version"`
	LocalIP            string `json:"local_ip"`
	LocalPreference    int    `json:"local_preference"`
	Multihop           int    `json:"multihop"`
	Name               string `json:"name"`
	NeighborAddress    string `json:"neighbor_address"`
	NeighborAs         int    `json:"neighbor_as"`
	NfvID              int    `json:"nfv_id"`
	NfvPortID          int    `json:"nfv_port_id"`
	Originate          string `json:"originate"`
	OutboundRouteMap   int    `json:"outboundRouteMap"`
	PrefixLength       int    `json:"prefix_length"`
	PrefixLimit        int    `json:"prefix_limit"`
	PrefixListInbound  string `json:"prefix_list_inbound"`
	PrefixListOutbound string `json:"prefix_list_outbound"`
	PrependInbound     int    `json:"prepend_inbound"`
	PrependOutbound    int    `json:"prepend_outbound"`
	RcircuitID         int    `json:"rcircuit_id"`
	RemoteIP           string `json:"remote_ip"`
	SiteID             int    `json:"site_id"`
	Status             string `json:"status"`
	SwitchID           int    `json:"switch_id"`
	SwitchName         string `json:"switch_name"`
	SwitchPortID       int    `json:"switch_port_id"`
	TermSwitchID       int    `json:"term_switch_id"`
	TermSwitchName     string `json:"term_switch_name"`
	TerminateOnSwitch  string `json:"terminate_on_switch"`
	UpdateSource       string `json:"update_source"`
	Vlan               int    `json:"vlan"`
	Weight             int    `json:"weight"`
}

// EBGPMetaStatus defines the observed state of EBGPMeta
type EBGPMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EBGPMeta is the Schema for the ebgpmeta API
type EBGPMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EBGPMetaSpec   `json:"spec,omitempty"`
	Status EBGPMetaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EBGPMetaList contains a list of EBGPMeta
type EBGPMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EBGPMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EBGPMeta{}, &EBGPMetaList{})
}
