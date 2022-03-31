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

	"github.com/netrisai/netriswebapi/v1/types/inventoryprofile"
)

// InventoryProfileStorage .
type InventoryProfileStorage struct {
	sync.Mutex
	InventoryProfile []*inventoryprofile.Profile
}

// NewInventoryProfileStorage .
func NewInventoryProfileStorage() *InventoryProfileStorage {
	return &InventoryProfileStorage{}
}

// GetAll .
func (p *InventoryProfileStorage) GetAll() []*inventoryprofile.Profile {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *InventoryProfileStorage) getAll() []*inventoryprofile.Profile {
	return p.InventoryProfile
}

func (p *InventoryProfileStorage) storeAll(items []*inventoryprofile.Profile) {
	p.InventoryProfile = items
}

// FindByName .
func (p *InventoryProfileStorage) FindByName(name string) (*inventoryprofile.Profile, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *InventoryProfileStorage) findByName(name string) (*inventoryprofile.Profile, bool) {
	for _, inventoryProfile := range p.InventoryProfile {
		if inventoryProfile.Name == name {
			return inventoryProfile, true
		}
	}
	return nil, false
}

// FindByID .
func (p *InventoryProfileStorage) FindByID(id int) (*inventoryprofile.Profile, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *InventoryProfileStorage) findByID(id int) (*inventoryprofile.Profile, bool) {
	for _, inventoryProfile := range p.InventoryProfile {
		if inventoryProfile.ID == id {
			return inventoryProfile, true
		}
	}
	return nil, false
}

// Download .
func (p *InventoryProfileStorage) download() error {
	items, err := Cred.InventoryProfile().Get()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *InventoryProfileStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
