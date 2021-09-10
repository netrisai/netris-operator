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

package netrisstorage

import (
	"sync"

	"github.com/netrisai/netriswebapi/v2/types/vnet"
)

// VNetStorage .
type VNetStorage struct {
	sync.Mutex
	VNets []*vnet.VNet
}

// NewVNetStorage .
func NewVNetStorage() *VNetStorage {
	return &VNetStorage{}
}

// GetAll .
func (p *VNetStorage) GetAll() []vnet.VNet {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *VNetStorage) getAll() []vnet.VNet {
	vnets := []vnet.VNet{}
	for _, vnet := range p.VNets {
		vnets = append(vnets, *vnet)
	}
	return vnets
}

func (p *VNetStorage) storeAll(items []*vnet.VNet) {
	p.VNets = items
}

// FindByName .
func (p *VNetStorage) FindByName(name string) (*vnet.VNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *VNetStorage) findByName(name string) (*vnet.VNet, bool) {
	for _, item := range p.VNets {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindByID .
func (p *VNetStorage) FindByID(id int) (*vnet.VNet, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *VNetStorage) findByID(id int) (*vnet.VNet, bool) {
	for _, item := range p.VNets {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindByGateway .
func (p *VNetStorage) FindByGateway(gateway string) (*vnet.VNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByGateway(gateway)
}

func (p *VNetStorage) findByGateway(gateway string) (*vnet.VNet, bool) {
	for _, item := range p.VNets {
		for _, gway := range item.Gateways {
			if gway.Prefix == gateway {
				return item, true
			}
		}
	}
	return nil, false
}

// Download .
func (p *VNetStorage) download() error {
	items, err := Cred.VNet().Get()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *VNetStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
