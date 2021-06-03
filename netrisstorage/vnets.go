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
	"strconv"
	"sync"

	api "github.com/netrisai/netrisapi"
)

// VNetStorage .
type VNetStorage struct {
	sync.Mutex
	VNets []*api.APIVNet
}

// NewVNetStorage .
func NewVNetStorage() *VNetStorage {
	return &VNetStorage{}
}

// GetAll .
func (p *VNetStorage) GetAll() []*api.APIVNet {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *VNetStorage) getAll() []*api.APIVNet {
	return p.VNets
}

func (p *VNetStorage) storeAll(items []*api.APIVNet) {
	p.VNets = items
}

// FindByName .
func (p *VNetStorage) FindByName(name string) (*api.APIVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *VNetStorage) findByName(name string) (*api.APIVNet, bool) {
	for _, item := range p.VNets {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindByID .
func (p *VNetStorage) FindByID(id int) (*api.APIVNet, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *VNetStorage) findByID(id int) (*api.APIVNet, bool) {
	for _, item := range p.VNets {
		vnetID, _ := strconv.Atoi(item.ID)
		if vnetID == id {
			return item, true
		}
	}
	return nil, false
}

// Download .
func (p *VNetStorage) download() error {
	items, err := Cred.GetVNets()
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
