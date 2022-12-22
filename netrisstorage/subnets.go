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

	"github.com/netrisai/netriswebapi/v2/types/ipam"
)

// SubnetsStorage .
type SubnetsStorage struct {
	sync.Mutex
	Subnets []*ipam.IPAM
}

// NewSubnetsStorage .
func NewSubnetsStorage() *SubnetsStorage {
	return &SubnetsStorage{}
}

// GetAll .
func (p *SubnetsStorage) GetAll() []*ipam.IPAM {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *SubnetsStorage) getAll() []*ipam.IPAM {
	subnets := []*ipam.IPAM{}
	subnets = append(subnets, p.Subnets...)
	return subnets
}

func (p *SubnetsStorage) storeAll(items []*ipam.IPAM) {
	p.Subnets = items
}

// FindByName .
func (p *SubnetsStorage) FindByName(name string) (*ipam.IPAM, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *SubnetsStorage) findByName(name string) (*ipam.IPAM, bool) {
	for _, item := range p.Subnets {
		if item.Name == name {
			return item, true
		}
		if s, ok := p.findByNameInChildren(item, name); ok {
			return s, true
		}
	}
	return nil, false
}

func (p *SubnetsStorage) findByNameInChildren(ipam *ipam.IPAM, name string) (*ipam.IPAM, bool) {
	for _, item := range ipam.Children {
		if item.Name == name {
			return item, true
		}
		if s, ok := p.findByNameInChildren(item, name); ok {
			return s, true
		}
	}
	return nil, false
}

// FindByID .
func (p *SubnetsStorage) FindByID(id int, typo string) (*ipam.IPAM, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id, typo)
	if !ok {
		_ = p.download()
		return p.findByID(id, typo)
	}
	return item, ok
}

func (p *SubnetsStorage) findInChildren(ipam *ipam.IPAM, id int, typo string) (*ipam.IPAM, bool) {
	for _, item := range ipam.Children {
		if item.ID == id && item.Type == typo {
			return item, true
		}

		if s, ok := p.findInChildren(item, id, typo); ok {
			return s, true
		}
	}
	return nil, false
}

func (p *SubnetsStorage) findByID(id int, typo string) (*ipam.IPAM, bool) {
	for _, item := range p.Subnets {
		if item.ID == id && item.Type == typo {
			return item, true
		}
		if s, ok := p.findInChildren(item, id, typo); ok {
			return s, true
		}
	}
	return nil, false
}

// Download .
func (p *SubnetsStorage) download() error {
	items, err := Cred.IPAM().Get()
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
