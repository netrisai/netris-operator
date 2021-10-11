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

// BGPMetaSpec defines the desired state of BGPMeta
type BGPMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Imported        bool   `json:"imported"`
	Reclaim         bool   `json:"reclaimPolicy"`
	BGPCRGeneration int64  `json:"bgpGeneration"`
	ID              int    `json:"id"`
	BGPName         string `json:"bgpName"`

	AllowasIn          int     `json:"allowas_in"`
	HWID               int     `json:"hwid"`
	Port               string  `json:"port"`
	VnetID             int     `json:"vnet"`
	Site               string  `json:"site"`
	BgpPassword        string  `json:"bgp_password"`
	Community          string  `json:"community"`
	Description        string  `json:"description"`
	InboundRouteMap    int     `json:"inboundRouteMap"`
	Internal           string  `json:"internal"`
	IPVersion          string  `json:"ip_version"`
	LocalIP            string  `json:"local_ip"`
	LocalPreference    int     `json:"local_preference"`
	Multihop           int     `json:"multihop"`
	Name               string  `json:"name"`
	NeighborAddress    *string `json:"neighbor_address,omitempty"`
	NeighborAs         int     `json:"neighbor_as"`
	Originate          string  `json:"originate"`
	OutboundRouteMap   int     `json:"outboundRouteMap"`
	PrefixLength       int     `json:"prefix_length"`
	PrefixLimit        int     `json:"prefix_limit"`
	PrefixListInbound  string  `json:"prefix_list_inbound"`
	PrefixListOutbound string  `json:"prefix_list_outbound"`
	PrependInbound     int     `json:"prepend_inbound"`
	PrependOutbound    int     `json:"prepend_outbound"`
	RemoteIP           string  `json:"remote_ip"`
	Status             string  `json:"status"`
	UpdateSource       string  `json:"update_source"`
	Vlan               int     `json:"vlan"`
	Weight             int     `json:"weight"`
}

// BGPMetaStatus defines the observed state of BGPMeta
type BGPMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BGPMeta is the Schema for the bgpmeta API
type BGPMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BGPMetaSpec   `json:"spec,omitempty"`
	Status BGPMetaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BGPMetaList contains a list of BGPMeta
type BGPMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BGPMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BGPMeta{}, &BGPMetaList{})
}
