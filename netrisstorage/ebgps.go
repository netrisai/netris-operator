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

	"github.com/netrisai/netriswebapi/v2/types/bgp"
)

// BGPStorage .
type BGPStorage struct {
	sync.Mutex
	BGPs []*bgp.EBGP
}

// NewBGPStorage .
func NewBGPStorage() *BGPStorage {
	return &BGPStorage{
		BGPs: []*bgp.EBGP{},
	}
}

func (p *BGPStorage) storeAll(items []*bgp.EBGP) {
	p.BGPs = items
}

// GetAll .
func (p *BGPStorage) GetAll() []*bgp.EBGP {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *BGPStorage) getAll() []*bgp.EBGP {
	return p.BGPs
}

// FindByID .
func (p *BGPStorage) FindByID(id int) (*bgp.EBGP, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *BGPStorage) findByID(id int) (*bgp.EBGP, bool) {
	for _, item := range p.BGPs {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindByName .
func (p *BGPStorage) FindByName(name string) (*bgp.EBGP, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *BGPStorage) findByName(name string) (*bgp.EBGP, bool) {
	for _, item := range p.BGPs {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// Download .
func (p *BGPStorage) download() error {
	items, err := Cred.BGP().Get()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *BGPStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
