/*
Copyright 2020.

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

package controllers

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	api "github.com/netrisai/netrisapi"
)

/********************************************************************************
	Storage
*********************************************************************************/

// Storage .
type Storage struct {
	sync.Mutex
	*PortsStorage
	*SitesStorage
	*TenantsStorage
	*VNetStorage
	*EBGPStorage
}

// NewStorage .
func NewStorage() *Storage {
	return &Storage{
		PortsStorage:   NewPortStorage(),
		SitesStorage:   NewSitesStorage(),
		TenantsStorage: NewTenantsStorage(),
		VNetStorage:    NewVNetStorage(),
		EBGPStorage:    NewEBGPStoragee(),
	}
}

// Download .
func (s *Storage) Download() error {
	s.Lock()
	defer s.Unlock()
	if err := s.PortsStorage.Download(); err != nil {
		return err
	}
	if err := s.SitesStorage.Download(); err != nil {
		return err
	}
	if err := s.TenantsStorage.Download(); err != nil {
		return err
	}
	if err := s.VNetStorage.Download(); err != nil {
		return err
	}
	if err := s.EBGPStorage.Download(); err != nil {
		return err
	}
	return nil
}

// DownloadWithInterval .
func (s *Storage) DownloadWithInterval() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		<-ticker.C
		err := s.Download()
		if err != nil {
			fmt.Println(err)
		}
	}
}

/********************************************************************************
	Port Storage
*********************************************************************************/

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

/********************************************************************************
	Sites Storage
*********************************************************************************/

// SitesStorage .
type SitesStorage struct {
	sync.Mutex
	Sites []*api.APISite
}

// NewSitesStorage .
func NewSitesStorage() *SitesStorage {
	return &SitesStorage{}
}

// GetAll .
func (p *SitesStorage) GetAll() []*api.APISite {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *SitesStorage) getAll() []*api.APISite {
	return p.Sites
}

func (p *SitesStorage) storeAll(items []*api.APISite) {
	p.Sites = items
}

// FindByName .
func (p *SitesStorage) FindByName(name string) (*api.APISite, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *SitesStorage) findByName(name string) (*api.APISite, bool) {
	for _, site := range p.Sites {
		if site.Name == name {
			return site, true
		}
	}
	return nil, false
}

// Download .
func (p *SitesStorage) download() error {
	items, err := Cred.GetSites()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *SitesStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}

/********************************************************************************
	Tenants Storage
*********************************************************************************/

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
	return p.findByID(id)
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

/********************************************************************************
	VNet Storage
*********************************************************************************/

// VNetStorage .
type VNetStorage struct {
	sync.Mutex
	VNets []*api.APIVNet
}

// NewVNetStorage .
func NewVNetStorage() *VNetStorage {
	return &VNetStorage{}
}

// GetAll .
func (p *VNetStorage) GetAll() []*api.APIVNet {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *VNetStorage) getAll() []*api.APIVNet {
	return p.VNets
}

func (p *VNetStorage) storeAll(items []*api.APIVNet) {
	p.VNets = items
}

// FindByName .
func (p *VNetStorage) FindByName(name string) (*api.APIVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *VNetStorage) findByName(name string) (*api.APIVNet, bool) {
	for _, item := range p.VNets {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindByID .
func (p *VNetStorage) FindByID(id int) (*api.APIVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByID(id)
}

func (p *VNetStorage) findByID(id int) (*api.APIVNet, bool) {
	for _, item := range p.VNets {
		vnetID, _ := strconv.Atoi(item.ID)
		if vnetID == id {
			return item, true
		}
	}
	return nil, false
}

// Download .
func (p *VNetStorage) download() error {
	items, err := Cred.GetVNets()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *VNetStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}

/********************************************************************************
	EBGP Storage
*********************************************************************************/

// EBGPStorage .
type EBGPStorage struct {
	sync.Mutex
	EBGPs []*api.APIEBGP
}

// NewEBGPStorage .
func NewEBGPStoragee() *EBGPStorage {
	return &EBGPStorage{}
}

func (p *EBGPStorage) storeAll(items []*api.APIEBGP) {
	p.EBGPs = items
}

// GetAll .
func (p *EBGPStorage) GetAll() []*api.APIEBGP {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *EBGPStorage) getAll() []*api.APIEBGP {
	return p.EBGPs
}

// FindByID .
func (p *EBGPStorage) FindByID(id int) (*api.APIEBGP, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByID(id)
}

func (p *EBGPStorage) findByID(id int) (*api.APIEBGP, bool) {
	for _, item := range p.EBGPs {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// Download .
func (p *EBGPStorage) download() error {
	items, err := Cred.GetEBGPs()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *EBGPStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
