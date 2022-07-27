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
	"strings"

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
	Status       string      `json:"status,omitempty"`
	Message      string      `json:"message,omitempty"`
	State        string      `json:"state,omitempty"`
	Gateways     string      `json:"gateways,omitempty"`
	Sites        string      `json:"sites,omitempty"`
	ModifiedDate metav1.Time `json:"modified,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Gateways",type=string,JSONPath=`.status.gateways`
// +kubebuilder:printcolumn:name="Sites",type=string,JSONPath=".status.sites"
// +kubebuilder:printcolumn:name="Modified",type=date,JSONPath=`.status.modified`,priority=1
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
	State string `json:"state,omitempty"`

	GuestTenants []string   `json:"guestTenants"`
	Sites        []VNetSite `json:"sites"`
	VlanID       string     `json:"vlanid"`
}

// VNetSite .
type VNetSite struct {
	Name string `json:"name"`

	Gateways    []VNetGateway    `json:"gateways,omitempty"`
	SwitchPorts []VNetSwitchPort `json:"switchPorts,omitempty"`
}

// +kubebuilder:validation:Pattern=`(^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\/([0-9]|[12]\d|3[0-2]))?$)|(^((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?(\/([1-9]|[1-5][0-9]|6[0-4]))?$)`

// VNetGateway .
type VNetGateway string

func (v *VNetGateway) String() string {
	if v == nil {
		return ""
	}
	return string(*v)
}

// VNetSwitchPort .
type VNetSwitchPort struct {
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9]+@[a-zA-Z0-9-]+$`
	Name string `json:"name"`

	VlanID string `json:"vlanId,omitempty"`
	State  string `json:"state,omitempty"`
}

func init() {
	SchemeBuilder.Register(&VNet{}, &VNetList{})
}

// GatewaysString returns stringified gateways list
func (vnet *VNet) GatewaysString() string {
	str := ""
	strArr := []string{}
	for _, site := range vnet.Spec.Sites {
		for _, gateway := range site.Gateways {
			strArr = append(strArr, gateway.String())
		}
	}
	str = strings.Join(strArr, ", ")
	return str
}

// SitesString returns stringified site names list
func (vnet *VNet) SitesString() string {
	str := ""
	strArr := []string{}
	for _, site := range vnet.Spec.Sites {
		strArr = append(strArr, site.Name)
	}
	str = strings.Join(strArr, ", ")
	return str
}
