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

// TenantsStorage .
type TenantsStorage struct {
	sync.Mutex
	Tenants []*api.APITenant
}

// NewTenantsStorage .
func NewTenantsStorage() *TenantsStorage {
	return &TenantsStorage{}
}

// GetAll .
func (p *TenantsStorage) GetAll() []*api.APITenant {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *TenantsStorage) getAll() []*api.APITenant {
	return p.Tenants
}

func (p *TenantsStorage) storeAll(items []*api.APITenant) {
	p.Tenants = items
}

// FindByName .
func (p *TenantsStorage) FindByName(name string) (*api.APITenant, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *TenantsStorage) findByName(name string) (*api.APITenant, bool) {
	for _, item := range p.Tenants {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindByID .
func (p *TenantsStorage) FindByID(id int) (*api.APITenant, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *TenantsStorage) findByID(id int) (*api.APITenant, bool) {
	for _, item := range p.Tenants {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// Download .
func (p *TenantsStorage) download() error {
	items, err := Cred.GetTenants()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *TenantsStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
