package controllers

import (
	"fmt"
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
}

// NewStorage .
func NewStorage() *Storage {
	return &Storage{
		PortsStorage:   NewPortStorage(),
		SitesStorage:   NewSitesStorage(),
		TenantsStorage: NewTenantsStorage(),
	}
}

// Download .
func (s *Storage) Download() error {
	s.Lock()
	defer s.Unlock()
	err := s.PortsStorage.Download()
	err = s.SitesStorage.Download()
	err = s.TenantsStorage.Download()
	return err
}

// DownloadWithInterval .
func (s *Storage) DownloadWithInterval() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			err := s.Download()
			if err != nil {
				fmt.Println(err)
			}
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
