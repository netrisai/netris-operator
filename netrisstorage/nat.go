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

	"github.com/netrisai/netriswebapi/v2/types/nat"
)

// NATStorage .
type NATStorage struct {
	sync.Mutex
	NAT []*nat.NAT
}

// NewNATStorage .
func NewNATStorage() *NATStorage {
	return &NATStorage{}
}

// GetAll .
func (p *NATStorage) GetAll() []*nat.NAT {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *NATStorage) getAll() []*nat.NAT {
	return p.NAT
}

func (p *NATStorage) storeAll(items []*nat.NAT) {
	p.NAT = items
}

// FindByName .
func (p *NATStorage) FindByName(name string) (*nat.NAT, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *NATStorage) findByName(name string) (*nat.NAT, bool) {
	for _, nat := range p.NAT {
		if nat.Name == name {
			return nat, true
		}
	}
	return nil, false
}

// FindByID .
func (p *NATStorage) FindByID(id int) (*nat.NAT, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *NATStorage) findByID(id int) (*nat.NAT, bool) {
	for _, nat := range p.NAT {
		if nat.ID == id {
			return nat, true
		}
	}
	return nil, false
}

// Download .
func (p *NATStorage) download() error {
	items, err := Cred.NAT().Get()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *NATStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
