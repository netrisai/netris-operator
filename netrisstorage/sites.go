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

// FindByID .
func (p *SitesStorage) FindByID(id int) (*api.APISite, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *SitesStorage) findByID(id int) (*api.APISite, bool) {
	for _, site := range p.Sites {
		if site.ID == id {
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
