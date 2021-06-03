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

// BGPStorage .
type BGPStorage struct {
	sync.Mutex
	BGPs              []*api.APIEBGP
	BGPSites          []*api.APIEBGPSite
	BGPVNets          []*api.APIEBGPVNet
	BGPRouteMaps      []*api.APIEBGPRouteMap
	BGPOffloaders     map[int][]*api.APIEBGPOffloader
	BGPPorts          map[int][]*api.APIEBGPPort
	BGPSwitches       map[int][]*api.APIEBGPSwitch
	BGPUpdatedSources []*api.APIEBGPUpdatedSource
}

// NewBGPStorage .
func NewBGPStoragee() *BGPStorage {
	return &BGPStorage{
		BGPs:              []*api.APIEBGP{},
		BGPSites:          []*api.APIEBGPSite{},
		BGPVNets:          []*api.APIEBGPVNet{},
		BGPRouteMaps:      []*api.APIEBGPRouteMap{},
		BGPOffloaders:     make(map[int][]*api.APIEBGPOffloader),
		BGPPorts:          make(map[int][]*api.APIEBGPPort),
		BGPSwitches:       make(map[int][]*api.APIEBGPSwitch),
		BGPUpdatedSources: []*api.APIEBGPUpdatedSource{},
	}
}

func (p *BGPStorage) storeBGPSites(items []*api.APIEBGPSite) {
	p.BGPSites = items
}

func (p *BGPStorage) storeBGPVNets(items []*api.APIEBGPVNet) {
	p.BGPVNets = items
}

func (p *BGPStorage) storeBGPRouteMaps(items []*api.APIEBGPRouteMap) {
	p.BGPRouteMaps = items
}

func (p *BGPStorage) storeBGPOffloaders(siteID int, items []*api.APIEBGPOffloader) {
	p.BGPOffloaders[siteID] = items
}

func (p *BGPStorage) storeBGPPorts(siteID int, items []*api.APIEBGPPort) {
	p.BGPPorts[siteID] = items
}

func (p *BGPStorage) storeBGPSwitches(siteID int, items []*api.APIEBGPSwitch) {
	p.BGPSwitches[siteID] = items
}

func (p *BGPStorage) storeBGPUpdatedSources(items []*api.APIEBGPUpdatedSource) {
	p.BGPUpdatedSources = items
}

func (p *BGPStorage) storeAll(items []*api.APIEBGP) {
	p.BGPs = items
}

// GetAll .
func (p *BGPStorage) GetAll() []*api.APIEBGP {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *BGPStorage) getAll() []*api.APIEBGP {
	return p.BGPs
}

// FindByID .
func (p *BGPStorage) FindByID(id int) (*api.APIEBGP, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *BGPStorage) findByID(id int) (*api.APIEBGP, bool) {
	for _, item := range p.BGPs {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindByName .
func (p *BGPStorage) FindByName(name string) (*api.APIEBGP, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *BGPStorage) findByName(name string) (*api.APIEBGP, bool) {
	for _, item := range p.BGPs {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindSiteByID .
func (p *BGPStorage) FindSiteByID(id int) (*api.APIEBGPSite, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSiteByID(id)
}

func (p *BGPStorage) findSiteByID(id int) (*api.APIEBGPSite, bool) {
	for _, item := range p.BGPSites {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindSiteByName .
func (p *BGPStorage) FindSiteByName(name string) (*api.APIEBGPSite, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSiteByName(name)
}

func (p *BGPStorage) findSiteByName(name string) (*api.APIEBGPSite, bool) {
	for _, item := range p.BGPSites {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindOffloaderByID .
func (p *BGPStorage) FindOffloaderByID(siteID, id int) (*api.APIEBGPOffloader, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findOffloaderByID(siteID, id)
}

func (p *BGPStorage) findOffloaderByID(siteID, id int) (*api.APIEBGPOffloader, bool) {
	if offloaders, ok := p.BGPOffloaders[siteID]; ok {
		for _, item := range offloaders {
			if item.SwitchID == id {
				return item, true
			}
		}
	}
	return nil, false
}

// FindOffloaderByName .
func (p *BGPStorage) FindOffloaderByName(siteID int, name string) (*api.APIEBGPOffloader, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findOffloaderByName(siteID, name)
}

func (p *BGPStorage) findOffloaderByName(siteID int, name string) (*api.APIEBGPOffloader, bool) {
	if offloaders, ok := p.BGPOffloaders[siteID]; ok {
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
func (p *BGPStorage) FindPort(siteID int, portName string) (*api.APIEBGPPort, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findPort(siteID, portName)
}

func (p *BGPStorage) findPort(siteID int, portName string) (*api.APIEBGPPort, bool) {
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
func (p *BGPStorage) FindVNetByID(id int) (*api.APIEBGPVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findVNetByID(id)
}

func (p *BGPStorage) findVNetByID(id int) (*api.APIEBGPVNet, bool) {
	for _, vnet := range p.BGPVNets {
		if vnet.ID == id {
			return vnet, true
		}
	}
	return nil, false
}

// FindVNetByName .
func (p *BGPStorage) FindVNetByName(name string) (*api.APIEBGPVNet, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findVNetByName(name)
}

func (p *BGPStorage) findVNetByName(name string) (*api.APIEBGPVNet, bool) {
	for _, vnet := range p.BGPVNets {
		if vnet.Name == name {
			return vnet, true
		}
	}
	return nil, false
}

// FindSwitchByID .
func (p *BGPStorage) FindSwitchByID(siteID, id int) (*api.APIEBGPSwitch, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSwitchByID(siteID, id)
}

func (p *BGPStorage) findSwitchByID(siteID, id int) (*api.APIEBGPSwitch, bool) {
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
func (p *BGPStorage) FindSwitchByName(siteID int, name string) (*api.APIEBGPSwitch, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findSwitchByName(siteID, name)
}

func (p *BGPStorage) findSwitchByName(siteID int, name string) (*api.APIEBGPSwitch, bool) {
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
func (p *BGPStorage) FindRouteMapByID(id int) (*api.APIEBGPRouteMap, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findRouteMapByID(id)
}

func (p *BGPStorage) findRouteMapByID(id int) (*api.APIEBGPRouteMap, bool) {
	for _, item := range p.BGPRouteMaps {
		if item.ID == id {
			return item, true
		}
	}
	return nil, false
}

// FindRouteMapByName .
func (p *BGPStorage) FindRouteMapByName(name string) (*api.APIEBGPRouteMap, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findRouteMapByName(name)
}

func (p *BGPStorage) findRouteMapByName(name string) (*api.APIEBGPRouteMap, bool) {
	for _, item := range p.BGPRouteMaps {
		if item.Name == name {
			return item, true
		}
	}
	return nil, false
}

// FindUpdatedSource .
func (p *BGPStorage) FindUpdatedSource(source string) (*api.APIEBGPUpdatedSource, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findUpdatedSource(source)
}

func (p *BGPStorage) findUpdatedSource(source string) (*api.APIEBGPUpdatedSource, bool) {
	for _, item := range p.BGPUpdatedSources {
		if item.IPAddress == source {
			return item, true
		}
	}
	return nil, false
}

// Download .
func (p *BGPStorage) download() error {
	items, err := Cred.GetEBGPs()
	if err != nil {
		return err
	}
	p.storeAll(items)

	sites, err := Cred.GetEBGPSites()
	if err != nil {
		return err
	}
	p.storeBGPSites(sites)

	vnets, err := Cred.GetEBGPVNets()
	if err != nil {
		return err
	}
	p.storeBGPVNets(vnets)

	rmaps, err := Cred.GetEBGPRouteMaps()
	if err != nil {
		return err
	}
	p.storeBGPRouteMaps(rmaps)

	for _, site := range sites {
		offloaders, err := Cred.GetEBGPOffloaders(site.ID)
		if err != nil {
			return err
		}
		p.storeBGPOffloaders(site.ID, offloaders)

		ports, err := Cred.GetEBGPPorts(site.ID)
		if err != nil {
			return err
		}
		p.storeBGPPorts(site.ID, ports)

		sws, err := Cred.GetEBGPSwitches(site.ID)
		if err != nil {
			return err
		}
		p.storeBGPSwitches(site.ID, sws)
	}

	usources, err := Cred.GetEBGPUpdatedSources()
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
