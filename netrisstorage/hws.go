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

// HWsStorage .
type HWsStorage struct {
	sync.Mutex
	HWs []*api.APIInventory
}

// NewHWsStorage .
func NewHWsStorage() *HWsStorage {
	return &HWsStorage{}
}

// GetAll .
func (p *HWsStorage) GetAll() []*api.APIInventory {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *HWsStorage) getAll() []*api.APIInventory {
	return p.HWs
}

func (p *HWsStorage) storeAll(items []*api.APIInventory) {
	p.HWs = items
}

// FindByName .
func (p *HWsStorage) FindByName(name string) (*api.APIInventory, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *HWsStorage) findByName(name string) (*api.APIInventory, bool) {
	for _, hw := range p.HWs {
		if hw.SwitchName == name {
			return hw, true
		}
	}
	return nil, false
}

// FindByID .
func (p *HWsStorage) FindByID(id int) (*api.APIInventory, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *HWsStorage) findByID(id int) (*api.APIInventory, bool) {
	for _, hw := range p.HWs {
		if hw.ID == id {
			return hw, true
		}
	}
	return nil, false
}

// FindHWsBySite .
func (p *HWsStorage) FindHWsBySite(siteID int) []api.APIInventory {
	p.Lock()
	defer p.Unlock()
	return p.findHWsBySite(siteID)
}

func (p *HWsStorage) findHWsBySite(siteID int) []api.APIInventory {
	hws := []api.APIInventory{}
	for _, hw := range p.HWs {
		if hw.SiteID == siteID {
			hws = append(hws, *hw)
		}
	}
	return hws
}

// FindSpineBySite .
func (p *HWsStorage) FindSpineBySite(siteID int) *api.APIInventory {
	p.Lock()
	defer p.Unlock()
	for _, hw := range p.findHWsBySite(siteID) {
		if hw.Type == "spine" {
			sw := hw
			return &sw
		}
	}
	return nil
}

// Download .
func (p *HWsStorage) download() error {
	items, err := Cred.GetInventory()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *HWsStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
