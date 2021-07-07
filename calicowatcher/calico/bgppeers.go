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
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// BGPPeer contains information about a BGP peer resource that is a peer of a Calico
// compute node.
type BGPPeer struct {
	Name            string `json:"name"`
	metav1.TypeMeta `json:",inline"`

	// Metadata for a BGPPeer.
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification for a BGPPeer.
	Spec BGPPeerSpec `json:"spec,omitempty"`
}

// BGPPeerSpec contains the specification for a BGPPeer resource.
type BGPPeerSpec struct {
	// The AS Number of the peer.
	ASNumber int `json:"asNumber"`
	// The IP address of the peer.
	PeerIP net.IP `json:"peerIP" validate:"omitempty"`
}

// GetBGPConfiguration .
func GetBGPPeer(config *rest.Config) ([]*BGPPeer, error) {
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("{GetBGPPeer} %s", err)
	}

	bgpPeerResource := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgppeers",
	}

	list, err := dynClient.Resource(bgpPeerResource).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("{GetBGPPeer} %s", err)
	}

	var bgpPeers []*BGPPeer
	for _, peer := range list.Items {
		js, err := peer.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("{GetBGPPeer} %s", err)
		}

		var bgpPeer *BGPPeer
		err = json.Unmarshal(js, &bgpPeer)
		if err != nil {
			return nil, fmt.Errorf("{GetBGPPeer} %s", err)
		}
		bgpPeer.Name = bgpPeer.Metadata.Name
		bgpPeers = append(bgpPeers, bgpPeer)
	}
	return bgpPeers, nil
}

// DeleteBGPPeer .
func DeleteBGPPeer(peer *BGPPeer, config *rest.Config) error {
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("{DeleteBGPPeer} %s", err)
	}

	bgpPeerResource := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgppeers",
	}

	err = dynClient.Resource(bgpPeerResource).Delete(context.Background(), peer.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("{DeleteBGPPeer} %s", err)
	}
	return nil
}
