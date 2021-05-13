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
