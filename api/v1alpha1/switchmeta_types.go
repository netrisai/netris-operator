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
	"github.com/netrisai/netriswebapi/v2/types/inventory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SwitchMetaSpec defines the desired state of SwitchMeta
type SwitchMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Imported           bool   `json:"imported"`
	Reclaim            bool   `json:"reclaimPolicy"`
	SwitchCRGeneration int64  `json:"switchGeneration"`
	ID                 int    `json:"id"`
	SwitchName         string `json:"switchName"`

	TenantID    int           `json:"tenant,omitempty"`
	Description string        `json:"description,omitempty"`
	NOS         inventory.NOS `json:"nos,omitempty"`
	SiteID      int           `json:"site,omitempty"`
	ASN         int           `json:"asn,omitempty"`
	ProfileID   int           `json:"profile,omitempty"`
	MainIP      string        `json:"mainIp,omitempty"`
	MgmtIP      string        `json:"mgmtIp,omitempty"`
	PortsCount  int           `json:"portsCount,omitempty"`
	MacAddress  string        `json:"macAddress,omitempty"`
}

// SwitchMetaStatus defines the observed state of SwitchMeta
type SwitchMetaStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SwitchMeta is the Schema for the switchmeta API
type SwitchMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwitchMetaSpec   `json:"spec,omitempty"`
	Status SwitchMetaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SwitchMetaList contains a list of SwitchMeta
type SwitchMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwitchMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwitchMeta{}, &SwitchMetaList{})
}
