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
	"fmt"
	"sync"

	"github.com/netrisai/netriswebapi/v2/types/port"
)

// PortsStorage .
type PortsStorage struct {
	sync.Mutex
	Ports []*port.Port
}

// NewPortStorage .
func NewPortStorage() *PortsStorage {
	return &PortsStorage{}
}

func (p *PortsStorage) storeAll(ports []*port.Port) {
	p.Ports = ports
}

// GetAll .
func (p *PortsStorage) GetAll() []*port.Port {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *PortsStorage) getAll() []*port.Port {
	return p.Ports
}

// FindByName .
func (p *PortsStorage) FindByName(name string) (*port.Port, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *PortsStorage) findByName(name string) (*port.Port, bool) {
	for _, port := range p.Ports {
		portName := fmt.Sprintf("%s@%s", port.Port, port.SwitchName)
		if portName == name {
			return port, true
		}
	}
	return nil, false
}

// Download .
func (p *PortsStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	ports, err := Cred.Port().Get()
	if err != nil {
		return err
	}
	p.storeAll(ports)
	return nil
}
