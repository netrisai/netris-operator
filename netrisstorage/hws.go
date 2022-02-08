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

	"github.com/netrisai/netriswebapi/v2/types/inventory"
)

// HWsStorage .
type HWsStorage struct {
	sync.Mutex
	HWs []*inventory.HW
}

// NewHWsStorage .
func NewHWsStorage() *HWsStorage {
	return &HWsStorage{}
}

// GetAll .
func (p *HWsStorage) GetAll() []*inventory.HW {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *HWsStorage) getAll() []*inventory.HW {
	return p.HWs
}

func (p *HWsStorage) storeAll(items []*inventory.HW) {
	p.HWs = items
}

// FindByName .
func (p *HWsStorage) FindByName(name string) (*inventory.HW, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *HWsStorage) findByName(name string) (*inventory.HW, bool) {
	for _, hw := range p.HWs {
		if hw.Name == name {
			return hw, true
		}
	}
	return nil, false
}

// FindByName .
func (p *HWsStorage) FindSoftgateByName(name string) (*inventory.HW, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSoftgateByName(name)
}

func (p *HWsStorage) findSoftgateByName(name string) (*inventory.HW, bool) {
	for _, hw := range p.HWs {
		if hw.Name == name && hw.Type == "softgate" {
			return hw, true
		}
	}
	return nil, false
}

// FindByName .
func (p *HWsStorage) FindSwitchByName(name string) (*inventory.HW, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSwitchByName(name)
}

func (p *HWsStorage) findSwitchByName(name string) (*inventory.HW, bool) {
	for _, hw := range p.HWs {
		if hw.Name == name && hw.Type == "switch" {
			return hw, true
		}
	}
	return nil, false
}

// FindByName .
func (p *HWsStorage) FindControllerByName(name string) (*inventory.HW, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findControllerByName(name)
}

func (p *HWsStorage) findControllerByName(name string) (*inventory.HW, bool) {
	for _, hw := range p.HWs {
		if hw.Name == name && hw.Type == "controller" {
			return hw, true
		}
	}
	return nil, false
}

// FindByID .
func (p *HWsStorage) FindByID(id int) (*inventory.HW, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *HWsStorage) findByID(id int) (*inventory.HW, bool) {
	for _, hw := range p.HWs {
		if hw.ID == id {
			return hw, true
		}
	}
	return nil, false
}

// FindByID .
func (p *HWsStorage) FindSoftgateByID(id int) (*inventory.HW, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findSoftgateByID(id)
	if !ok {
		_ = p.download()
		return p.findSoftgateByID(id)
	}
	return item, ok
}

func (p *HWsStorage) findSoftgateByID(id int) (*inventory.HW, bool) {
	for _, hw := range p.HWs {
		if hw.ID == id && hw.Type == "softgate" {
			return hw, true
		}
	}
	return nil, false
}

// FindByID .
func (p *HWsStorage) FindControllerByID(id int) (*inventory.HW, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findControllerByID(id)
	if !ok {
		_ = p.download()
		return p.findControllerByID(id)
	}
	return item, ok
}

func (p *HWsStorage) findControllerByID(id int) (*inventory.HW, bool) {
	for _, hw := range p.HWs {
		if hw.ID == id && hw.Type == "controller" {
			return hw, true
		}
	}
	return nil, false
}

// FindByID .
func (p *HWsStorage) FindSwitchByID(id int) (*inventory.HW, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findSwitchByID(id)
	if !ok {
		_ = p.download()
		return p.findSwitchByID(id)
	}
	return item, ok
}

func (p *HWsStorage) findSwitchByID(id int) (*inventory.HW, bool) {
	for _, hw := range p.HWs {
		if hw.ID == id && hw.Type == "switch" {
			return hw, true
		}
	}
	return nil, false
}

// FindHWsBySite .
func (p *HWsStorage) FindHWsBySite(siteID int) []inventory.HW {
	p.Lock()
	defer p.Unlock()
	return p.findHWsBySite(siteID)
}

func (p *HWsStorage) findHWsBySite(siteID int) []inventory.HW {
	hws := []inventory.HW{}
	for _, hw := range p.HWs {
		if hw.Site.ID == siteID {
			hws = append(hws, *hw)
		}
	}
	return hws
}

// FindSpineBySite .
func (p *HWsStorage) FindSpineBySite(siteID int) *inventory.HW {
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
	items, err := Cred.Inventory().Get()
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
