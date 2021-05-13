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

	api "github.com/netrisai/netrisapi"
)

// L4LBStorage .
type L4LBStorage struct {
	sync.Mutex
	L4LBs []*api.APILoadBalancer
}

// NewVNetStorage .
func NewL4LBStorage() *L4LBStorage {
	return &L4LBStorage{}
}

// GetAll .
func (p *L4LBStorage) GetAll() []*api.APILoadBalancer {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *L4LBStorage) getAll() []*api.APILoadBalancer {
	return p.L4LBs
}

func (p *L4LBStorage) storeAll(items []*api.APILoadBalancer) {
	p.L4LBs = items
}

// FindByName .
func (p *L4LBStorage) FindByName(name string) (*api.APILoadBalancer, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *L4LBStorage) findByName(name string) (*api.APILoadBalancer, bool) {
	for _, item := range p.L4LBs {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindByID .
func (p *L4LBStorage) FindByID(id int) (*api.APILoadBalancer, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *L4LBStorage) findByID(id int) (*api.APILoadBalancer, bool) {
	for _, item := range p.L4LBs {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// Download .
func (p *L4LBStorage) download() error {
	items, err := Cred.GetLoadBalancers()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *L4LBStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
