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

	"github.com/netrisai/netriswebapi/v2/types/bgp"
)

// BGPStorage .
type BGPStorage struct {
	sync.Mutex
	BGPs              []*bgp.EBGP
	BGPSites          []*bgp.EBGPSite
	BGPVNets          []*bgp.EBGPVNet
	BGPRouteMaps      []*bgp.EBGPRouteMap
	BGPOffloaders     map[int][]*bgp.EBGPOffloader
	BGPPorts          map[int][]*bgp.EBGPPort
	BGPSwitches       map[int][]*bgp.EBGPSwitch
	BGPUpdatedSources []*bgp.EBGPUpdatedSource
}

// NewBGPStorage .
func NewBGPStoragee() *BGPStorage {
	return &BGPStorage{
		BGPs:              []*bgp.EBGP{},
		BGPSites:          []*bgp.EBGPSite{},
		BGPVNets:          []*bgp.EBGPVNet{},
		BGPRouteMaps:      []*bgp.EBGPRouteMap{},
		BGPOffloaders:     make(map[int][]*bgp.EBGPOffloader),
		BGPPorts:          make(map[int][]*bgp.EBGPPort),
		BGPSwitches:       make(map[int][]*bgp.EBGPSwitch),
		BGPUpdatedSources: []*bgp.EBGPUpdatedSource{},
	}
}

func (p *BGPStorage) storeBGPSites(items []*bgp.EBGPSite) {
	p.BGPSites = items
}

func (p *BGPStorage) storeBGPVNets(items []*bgp.EBGPVNet) {
	p.BGPVNets = items
}

func (p *BGPStorage) storeBGPRouteMaps(items []*bgp.EBGPRouteMap) {
	p.BGPRouteMaps = items
}

func (p *BGPStorage) storeBGPOffloaders(siteID int, items []*bgp.EBGPOffloader) {
	p.BGPOffloaders[siteID] = items
}

func (p *BGPStorage) storeBGPPorts(siteID int, items []*bgp.EBGPPort) {
	p.BGPPorts[siteID] = items
}

func (p *BGPStorage) storeBGPSwitches(siteID int, items []*bgp.EBGPSwitch) {
	p.BGPSwitches[siteID] = items
}

func (p *BGPStorage) storeBGPUpdatedSources(items []*bgp.EBGPUpdatedSource) {
	p.BGPUpdatedSources = items
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

// FindSiteByID .
func (p *BGPStorage) FindSiteByID(id int) (*bgp.EBGPSite, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSiteByID(id)
}

func (p *BGPStorage) findSiteByID(id int) (*bgp.EBGPSite, bool) {
	for _, item := range p.BGPSites {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindSiteByName .
func (p *BGPStorage) FindSiteByName(name string) (*bgp.EBGPSite, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSiteByName(name)
}

func (p *BGPStorage) findSiteByName(name string) (*bgp.EBGPSite, bool) {
	for _, item := range p.BGPSites {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindOffloaderByID .
func (p *BGPStorage) FindOffloaderByID(siteID, id int) (*bgp.EBGPOffloader, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findOffloaderByID(siteID, id)
}

func (p *BGPStorage) findOffloaderByID(siteID, id int) (*bgp.EBGPOffloader, bool) {
	if offloaders, ok := p.BGPOffloaders[siteID]; ok {
		for _, item := range offloaders {
			if item.ID == id {
				return item, true
			}
		}
	}
	return nil, false
}

// FindOffloaderByName .
func (p *BGPStorage) FindOffloaderByName(siteID int, name string) (*bgp.EBGPOffloader, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findOffloaderByName(siteID, name)
}

func (p *BGPStorage) findOffloaderByName(siteID int, name string) (*bgp.EBGPOffloader, bool) {
	if offloaders, ok := p.BGPOffloaders[siteID]; ok {
		for _, item := range offloaders {
			if item.Name == name {
				return item, true
			}
		}
	}
	return nil, false
}

/* FindPort .
Example: FindPort(swp1@switch1)
*/
func (p *BGPStorage) FindPort(siteID int, portName string) (*bgp.EBGPPort, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findPort(siteID, portName)
}

func (p *BGPStorage) findPort(siteID int, portName string) (*bgp.EBGPPort, bool) {
	if ports, ok := p.BGPPorts[siteID]; ok {
		for _, port := range ports {
			if fmt.Sprintf("%s@%s", port.Port, port.SwitchName) == portName {
				return port, true
			}
		}
	}
	return nil, false
}

// FindVNetByID .
func (p *BGPStorage) FindVNetByID(id int) (*bgp.EBGPVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findVNetByID(id)
}

func (p *BGPStorage) findVNetByID(id int) (*bgp.EBGPVNet, bool) {
	for _, vnet := range p.BGPVNets {
		if vnet.ID == id {
			return vnet, true
		}
	}
	return nil, false
}

// FindVNetByName .
func (p *BGPStorage) FindVNetByName(name string) (*bgp.EBGPVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findVNetByName(name)
}

func (p *BGPStorage) findVNetByName(name string) (*bgp.EBGPVNet, bool) {
	for _, vnet := range p.BGPVNets {
		if vnet.Name == name {
			return vnet, true
		}
	}
	return nil, false
}

// FindSwitchByID .
func (p *BGPStorage) FindSwitchByID(siteID, id int) (*bgp.EBGPSwitch, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSwitchByID(siteID, id)
}

func (p *BGPStorage) findSwitchByID(siteID, id int) (*bgp.EBGPSwitch, bool) {
	if switches, ok := p.BGPSwitches[siteID]; ok {
		for _, item := range switches {
			if item.SwitchID == id {
				return item, true
			}
		}
	}
	return nil, false
}

// FindSwitchByName .
func (p *BGPStorage) FindSwitchByName(siteID int, name string) (*bgp.EBGPSwitch, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSwitchByName(siteID, name)
}

func (p *BGPStorage) findSwitchByName(siteID int, name string) (*bgp.EBGPSwitch, bool) {
	if offloaders, ok := p.BGPSwitches[siteID]; ok {
		for _, item := range offloaders {
			if item.Location == name {
				return item, true
			}
		}
	}
	return nil, false
}

// FindRouteMapByID .
func (p *BGPStorage) FindRouteMapByID(id int) (*bgp.EBGPRouteMap, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findRouteMapByID(id)
}

func (p *BGPStorage) findRouteMapByID(id int) (*bgp.EBGPRouteMap, bool) {
	for _, item := range p.BGPRouteMaps {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindRouteMapByName .
func (p *BGPStorage) FindRouteMapByName(name string) (*bgp.EBGPRouteMap, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findRouteMapByName(name)
}

func (p *BGPStorage) findRouteMapByName(name string) (*bgp.EBGPRouteMap, bool) {
	for _, item := range p.BGPRouteMaps {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindUpdatedSource .
func (p *BGPStorage) FindUpdatedSource(source string) (*bgp.EBGPUpdatedSource, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findUpdatedSource(source)
}

func (p *BGPStorage) findUpdatedSource(source string) (*bgp.EBGPUpdatedSource, bool) {
	for _, item := range p.BGPUpdatedSources {
		if item.IPAddress == source {
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

	sites, err := Cred.BGP().GetSites()
	if err != nil {
		return err
	}
	p.storeBGPSites(sites)

	vnets, err := Cred.BGP().GetVNets()
	if err != nil {
		return err
	}
	p.storeBGPVNets(vnets)

	rmaps, err := Cred.BGP().GetRouteMaps()
	if err != nil {
		return err
	}
	p.storeBGPRouteMaps(rmaps)

	for _, site := range sites {
		offloaders, err := Cred.BGP().GetOffloaders(site.ID)
		if err != nil {
			return err
		}
		p.storeBGPOffloaders(site.ID, offloaders)

		ports, err := Cred.BGP().GetPorts(site.ID)
		if err != nil {
			return err
		}
		p.storeBGPPorts(site.ID, ports)

		sws, err := Cred.BGP().GetSwitches(site.ID)
		if err != nil {
			return err
		}
		p.storeBGPSwitches(site.ID, sws)
	}

	usources, err := Cred.BGP().GetUpdateSources()
	if err != nil {
		return err
	}
	p.storeBGPUpdatedSources(usources)

	return nil
}

// Download .
func (p *BGPStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
