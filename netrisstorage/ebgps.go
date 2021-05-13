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

// EBGPStorage .
type EBGPStorage struct {
	sync.Mutex
	EBGPs              []*api.APIEBGP
	EBGPSites          []*api.APIEBGPSite
	EBGPVNets          []*api.APIEBGPVNet
	EBGPRouteMaps      []*api.APIEBGPRouteMap
	EBGPOffloaders     map[int][]*api.APIEBGPOffloader
	EBGPPorts          map[int][]*api.APIEBGPPort
	EBGPSwitches       map[int][]*api.APIEBGPSwitch
	EBGPUpdatedSources []*api.APIEBGPUpdatedSource
}

// NewEBGPStorage .
func NewEBGPStoragee() *EBGPStorage {
	return &EBGPStorage{
		EBGPs:              []*api.APIEBGP{},
		EBGPSites:          []*api.APIEBGPSite{},
		EBGPVNets:          []*api.APIEBGPVNet{},
		EBGPRouteMaps:      []*api.APIEBGPRouteMap{},
		EBGPOffloaders:     make(map[int][]*api.APIEBGPOffloader),
		EBGPPorts:          make(map[int][]*api.APIEBGPPort),
		EBGPSwitches:       make(map[int][]*api.APIEBGPSwitch),
		EBGPUpdatedSources: []*api.APIEBGPUpdatedSource{},
	}
}

func (p *EBGPStorage) storeEBGPSites(items []*api.APIEBGPSite) {
	p.EBGPSites = items
}

func (p *EBGPStorage) storeEBGPVNets(items []*api.APIEBGPVNet) {
	p.EBGPVNets = items
}

func (p *EBGPStorage) storeEBGPRouteMaps(items []*api.APIEBGPRouteMap) {
	p.EBGPRouteMaps = items
}

func (p *EBGPStorage) storeEBGPOffloaders(siteID int, items []*api.APIEBGPOffloader) {
	p.EBGPOffloaders[siteID] = items
}

func (p *EBGPStorage) storeEBGPPorts(siteID int, items []*api.APIEBGPPort) {
	p.EBGPPorts[siteID] = items
}

func (p *EBGPStorage) storeEBGPSwitches(siteID int, items []*api.APIEBGPSwitch) {
	p.EBGPSwitches[siteID] = items
}

func (p *EBGPStorage) storeEBGPUpdatedSources(items []*api.APIEBGPUpdatedSource) {
	p.EBGPUpdatedSources = items
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
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *EBGPStorage) findByID(id int) (*api.APIEBGP, bool) {
	for _, item := range p.EBGPs {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindSiteByID .
func (p *EBGPStorage) FindSiteByID(id int) (*api.APIEBGPSite, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSiteByID(id)
}

func (p *EBGPStorage) findSiteByID(id int) (*api.APIEBGPSite, bool) {
	for _, item := range p.EBGPSites {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindSiteByName .
func (p *EBGPStorage) FindSiteByName(name string) (*api.APIEBGPSite, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSiteByName(name)
}

func (p *EBGPStorage) findSiteByName(name string) (*api.APIEBGPSite, bool) {
	for _, item := range p.EBGPSites {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindOffloaderByID .
func (p *EBGPStorage) FindOffloaderByID(siteID, id int) (*api.APIEBGPOffloader, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findOffloaderByID(siteID, id)
}

func (p *EBGPStorage) findOffloaderByID(siteID, id int) (*api.APIEBGPOffloader, bool) {
	if offloaders, ok := p.EBGPOffloaders[siteID]; ok {
		for _, item := range offloaders {
			if item.SwitchID == id {
				return item, true
			}
		}
	}
	return nil, false
}

// FindOffloaderByName .
func (p *EBGPStorage) FindOffloaderByName(siteID int, name string) (*api.APIEBGPOffloader, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findOffloaderByName(siteID, name)
}

func (p *EBGPStorage) findOffloaderByName(siteID int, name string) (*api.APIEBGPOffloader, bool) {
	if offloaders, ok := p.EBGPOffloaders[siteID]; ok {
		for _, item := range offloaders {
			if item.Location == name {
				return item, true
			}
		}
	}
	return nil, false
}

/* FindPort .
Example: FindPort(swp1@switch1)
*/
func (p *EBGPStorage) FindPort(siteID int, portName string) (*api.APIEBGPPort, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findPort(siteID, portName)
}

func (p *EBGPStorage) findPort(siteID int, portName string) (*api.APIEBGPPort, bool) {
	if ports, ok := p.EBGPPorts[siteID]; ok {
		for _, port := range ports {
			if fmt.Sprintf("%s@%s", port.Port, port.SwitchName) == portName {
				return port, true
			}
		}
	}
	return nil, false
}

// FindVNetByID .
func (p *EBGPStorage) FindVNetByID(id int) (*api.APIEBGPVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findVNetByID(id)
}

func (p *EBGPStorage) findVNetByID(id int) (*api.APIEBGPVNet, bool) {
	for _, vnet := range p.EBGPVNets {
		if vnet.ID == id {
			return vnet, true
		}
	}
	return nil, false
}

// FindVNetByName .
func (p *EBGPStorage) FindVNetByName(name string) (*api.APIEBGPVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findVNetByName(name)
}

func (p *EBGPStorage) findVNetByName(name string) (*api.APIEBGPVNet, bool) {
	for _, vnet := range p.EBGPVNets {
		if vnet.Name == name {
			return vnet, true
		}
	}
	return nil, false
}

// FindSwitchByID .
func (p *EBGPStorage) FindSwitchByID(siteID, id int) (*api.APIEBGPSwitch, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSwitchByID(siteID, id)
}

func (p *EBGPStorage) findSwitchByID(siteID, id int) (*api.APIEBGPSwitch, bool) {
	if switches, ok := p.EBGPSwitches[siteID]; ok {
		for _, item := range switches {
			if item.SwitchID == id {
				return item, true
			}
		}
	}
	return nil, false
}

// FindSwitchByName .
func (p *EBGPStorage) FindSwitchByName(siteID int, name string) (*api.APIEBGPSwitch, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSwitchByName(siteID, name)
}

func (p *EBGPStorage) findSwitchByName(siteID int, name string) (*api.APIEBGPSwitch, bool) {
	if offloaders, ok := p.EBGPSwitches[siteID]; ok {
		for _, item := range offloaders {
			if item.Location == name {
				return item, true
			}
		}
	}
	return nil, false
}

// FindRouteMapByID .
func (p *EBGPStorage) FindRouteMapByID(id int) (*api.APIEBGPRouteMap, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findRouteMapByID(id)
}

func (p *EBGPStorage) findRouteMapByID(id int) (*api.APIEBGPRouteMap, bool) {
	for _, item := range p.EBGPRouteMaps {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindRouteMapByName .
func (p *EBGPStorage) FindRouteMapByName(name string) (*api.APIEBGPRouteMap, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findRouteMapByName(name)
}

func (p *EBGPStorage) findRouteMapByName(name string) (*api.APIEBGPRouteMap, bool) {
	for _, item := range p.EBGPRouteMaps {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindUpdatedSource .
func (p *EBGPStorage) FindUpdatedSource(source string) (*api.APIEBGPUpdatedSource, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findUpdatedSource(source)
}

func (p *EBGPStorage) findUpdatedSource(source string) (*api.APIEBGPUpdatedSource, bool) {
	for _, item := range p.EBGPUpdatedSources {
		if item.IPAddress == source {
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

	sites, err := Cred.GetEBGPSites()
	if err != nil {
		return err
	}
	p.storeEBGPSites(sites)

	vnets, err := Cred.GetEBGPVNets()
	if err != nil {
		return err
	}
	p.storeEBGPVNets(vnets)

	rmaps, err := Cred.GetEBGPRouteMaps()
	if err != nil {
		return err
	}
	p.storeEBGPRouteMaps(rmaps)

	for _, site := range sites {
		offloaders, err := Cred.GetEBGPOffloaders(site.ID)
		if err != nil {
			return err
		}
		p.storeEBGPOffloaders(site.ID, offloaders)

		ports, err := Cred.GetEBGPPorts(site.ID)
		if err != nil {
			return err
		}
		p.storeEBGPPorts(site.ID, ports)

		sws, err := Cred.GetEBGPSwitches(site.ID)
		if err != nil {
			return err
		}
		p.storeEBGPSwitches(site.ID, sws)
	}

	usources, err := Cred.GetEBGPUpdatedSources()
	if err != nil {
		return err
	}
	p.storeEBGPUpdatedSources(usources)

	return nil
}

// Download .
func (p *EBGPStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
