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

package calico

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type IPIPConfiguration struct {
	// When enabled is true, ipip tunneling will be used to deliver packets to
	// destinations within this pool.
	Enabled bool `json:"enabled,omitempty"`

	// The IPIP mode.  This can be one of "always" or "cross-subnet".  A mode
	// of "always" will also use IPIP tunneling for routing to destination IP
	// addresses within this pool.  A mode of "cross-subnet" will only use IPIP
	// tunneling when the destination node is on a different subnet to the
	// originating node.  The default value (if not specified) is "always".
	Mode string `json:"mode,omitempty" validate:"ipIpMode"`
}

// IPPool contains information about an IPPool resource.
type IPPool struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the IPPool.
	Spec IPPoolSpec `json:"spec,omitempty"`
}

// IPPoolSpec contains the specification for an IPPool resource.
type IPPoolSpec struct {
	// The pool CIDR.
	CIDR string `json:"cidr" validate:"net"`

	// Contains configuration for VXLAN tunneling for this pool. If not specified,
	// then this is defaulted to "Never" (i.e. VXLAN tunneling is disabled).
	VXLANMode VXLANMode `json:"vxlanMode,omitempty" validate:"omitempty,vxlanMode"`

	// Contains configuration for IPIP tunneling for this pool. If not specified,
	// then this is defaulted to "Never" (i.e. IPIP tunneling is disabled).
	IPIPMode IPIPMode `json:"ipipMode,omitempty" validate:"omitempty,ipIpMode"`

	// When nat-outgoing is true, packets sent from Calico networked containers in
	// this pool to destinations outside of this pool will be masqueraded.
	NATOutgoing bool `json:"natOutgoing,omitempty"`

	// When disabled is true, Calico IPAM will not assign addresses from this pool.
	Disabled bool `json:"disabled,omitempty"`

	// The block size to use for IP address assignments from this pool. Defaults to 26 for IPv4 and 112 for IPv6.
	BlockSize int `json:"blockSize,omitempty"`

	// Allows IPPool to allocate for a specific node by label selector.
	NodeSelector string `json:"nodeSelector,omitempty" validate:"omitempty,selector"`

	// Deprecated: this field is only used for APIv1 backwards compatibility.
	// Setting this field is not allowed, this field is for internal use only.
	IPIP *IPIPConfiguration `json:"ipip,omitempty" validate:"omitempty,mustBeNil"`

	// Deprecated: this field is only used for APIv1 backwards compatibility.
	// Setting this field is not allowed, this field is for internal use only.
	NATOutgoingV1 bool `json:"nat-outgoing,omitempty" validate:"omitempty,mustBeFalse"`
}

type VXLANMode string

type IPIPMode string

// GetIPPool .
func GetIPPool(config *rest.Config) ([]*IPPool, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("{GetIPPool} %s", err)
	}

	bgpConfigurationResource := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "ippools",
	}

	list, err := dynClient.Resource(bgpConfigurationResource).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("{GetIPPool} %s", err)
	}

	var ipPools []*IPPool
	for _, peer := range list.Items {
		js, err := peer.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("{GetIPPool} %s", err)
		}

		var ipPool *IPPool
		err = json.Unmarshal(js, &ipPool)
		if err != nil {
			return nil, fmt.Errorf("{GetIPPool} %s", err)
		}
		ipPool.Name = ipPool.ObjectMeta.Name
		ipPools = append(ipPools, ipPool)
	}
	return ipPools, nil
}
