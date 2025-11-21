/*
Copyright 2025. Netris, Inc.

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

	"github.com/netrisai/netriswebapi/v2/types/vpc"
)

// VPCStorage caches VPC objects retrieved from Netris API.
type VPCStorage struct {
	sync.Mutex
	VPCs []*vpc.VPC
}

// NewVPCStorage creates new VPC storage.
func NewVPCStorage() *VPCStorage {
	return &VPCStorage{}
}

// GetAll returns a copy of cached VPCs.
func (p *VPCStorage) GetAll() []vpc.VPC {
	p.Lock()
	defer p.Unlock()
	vpcs := []vpc.VPC{}
	for _, item := range p.VPCs {
		vpcs = append(vpcs, *item)
	}
	return vpcs
}

// FindByName returns VPC by name if present in cache, triggering refresh on miss.
func (p *VPCStorage) FindByName(name string) (*vpc.VPC, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByName(name)
	if !ok {
		_ = p.download()
		return p.findByName(name)
	}
	return item, ok
}

func (p *VPCStorage) findByName(name string) (*vpc.VPC, bool) {
	for _, item := range p.VPCs {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindByID returns VPC by ID, refreshing cache on miss.
func (p *VPCStorage) FindByID(id int) (*vpc.VPC, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *VPCStorage) findByID(id int) (*vpc.VPC, bool) {
	for _, item := range p.VPCs {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

func (p *VPCStorage) storeAll(items []*vpc.VPC) {
	p.VPCs = items
}

func (p *VPCStorage) download() error {
	items, err := Cred.VPC().Get()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download refreshes cached VPCs.
func (p *VPCStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
