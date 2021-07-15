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

// SubnetsStorage .
type SubnetsStorage struct {
	sync.Mutex
	Subnets []*api.APISubnet
}

// NewVNetStorage .
func NewSubnetsStorage() *SubnetsStorage {
	return &SubnetsStorage{}
}

// GetAll .
func (p *SubnetsStorage) GetAll() []api.APISubnet {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *SubnetsStorage) getAll() []api.APISubnet {
	subnets := []api.APISubnet{}
	for _, subnet := range p.Subnets {
		subnets = append(subnets, *subnet)
	}
	return subnets
}

func (p *SubnetsStorage) storeAll(items []*api.APISubnet) {
	p.Subnets = items
}

// FindByName .
func (p *SubnetsStorage) FindByName(name string) (*api.APISubnet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *SubnetsStorage) findByName(name string) (*api.APISubnet, bool) {
	for _, item := range p.Subnets {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindByID .
func (p *SubnetsStorage) FindByID(id int) (*api.APISubnet, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *SubnetsStorage) findByID(id int) (*api.APISubnet, bool) {
	for _, item := range p.Subnets {
		subnetID, _ := strconv.Atoi(item.ID)
		if subnetID == id {
			return item, true
		}
	}
	return nil, false
}

// Download .
func (p *SubnetsStorage) download() error {
	items, err := Cred.GetSubnets()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *SubnetsStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
