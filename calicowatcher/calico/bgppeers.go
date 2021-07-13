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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
	PeerIP string `json:"peerIP" validate:"omitempty"`
}

// GetBGPPeers .
func (c *Calico) GetBGPPeers(config *rest.Config) ([]*BGPPeer, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("{GetBGPPeer} %s", err)
	}

	bgpPeerResource := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgppeers",
	}

	list, err := dynClient.Resource(bgpPeerResource).List(ctx, metav1.ListOptions{})
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

// GetBGPPeer .
func (c *Calico) GetBGPPeer(name string, config *rest.Config) (*BGPPeer, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("{GetBGPPeer} %s", err)
	}

	bgpPeerResource := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgppeers",
	}

	peer, err := dynClient.Resource(bgpPeerResource).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("{GetBGPPeer} %s", err)
	}

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

	return bgpPeer, nil
}

// DeleteBGPPeer .
func (c *Calico) DeleteBGPPeer(peer *BGPPeer, config *rest.Config) error {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("{DeleteBGPPeer} %s", err)
	}

	bgpPeerResource := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgppeers",
	}

	err = dynClient.Resource(bgpPeerResource).Delete(ctx, peer.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("{DeleteBGPPeer} %s", err)
	}
	return nil
}

// CreateBGPPeer .
func (c *Calico) CreateBGPPeer(peer *BGPPeer, config *rest.Config) error {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("{CreateBGPPeer} %s", err)
	}

	bgpPeerResource := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgppeers",
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(peer)
	if err != nil {
		return fmt.Errorf("{CreateBGPPeer} %s", err)
	}

	obj := &unstructured.Unstructured{
		Object: m,
	}

	_, err = dynClient.Resource(bgpPeerResource).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("{CreateBGPPeer} %s", err)
	}
	return nil
}

// UpdateBGPPeer .
func (c *Calico) UpdateBGPPeer(peer *BGPPeer, config *rest.Config) error {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("{UpdateBGPPeer} %s", err)
	}

	bgpPeerResource := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgppeers",
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(peer)
	if err != nil {
		return fmt.Errorf("{UpdateBGPPeer} %s", err)
	}

	obj := &unstructured.Unstructured{
		Object: m,
	}

	_, err = dynClient.Resource(bgpPeerResource).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("{UpdateBGPPeer} %s", err)
	}
	return nil
}

// GenerateBGPPeer .
func (c *Calico) GenerateBGPPeer(name, namespace, ip string, asn int) *BGPPeer {
	nmspace := "default"
	if len(namespace) > 0 {
		nmspace = namespace
	}
	return &BGPPeer{
		Metadata: metav1.ObjectMeta{
			Name:      name,
			Namespace: nmspace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "BGPPeer",
			APIVersion: "crd.projectcalico.org/v1",
		},
		Spec: BGPPeerSpec{
			ASNumber: asn,
			PeerIP:   ip,
		},
	}
}
