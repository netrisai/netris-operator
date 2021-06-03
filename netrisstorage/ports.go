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

	api "github.com/netrisai/netrisapi"
)

// PortsStorage .
type PortsStorage struct {
	sync.Mutex
	Ports []*api.APIPort
}

// NewPortStorage .
func NewPortStorage() *PortsStorage {
	return &PortsStorage{}
}

func (p *PortsStorage) storeAll(ports []*api.APIPort) {
	p.Ports = ports
}

// GetAll .
func (p *PortsStorage) GetAll() []*api.APIPort {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *PortsStorage) getAll() []*api.APIPort {
	return p.Ports
}

// FindByName .
func (p *PortsStorage) FindByName(name string) (*api.APIPort, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *PortsStorage) findByName(name string) (*api.APIPort, bool) {
	for _, port := range p.Ports {
		portName := fmt.Sprintf("%s@%s", port.SlavePortName, port.SwitchName)
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
	ports, err := Cred.GetPorts()
	if err != nil {
		return err
	}
	p.storeAll(ports)
	return nil
}
