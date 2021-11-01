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

	"github.com/netrisai/netriswebapi/v1/types/site"
)

// SitesStorage .
type SitesStorage struct {
	sync.Mutex
	Sites []*site.Site
}

// NewSitesStorage .
func NewSitesStorage() *SitesStorage {
	return &SitesStorage{}
}

// GetAll .
func (p *SitesStorage) GetAll() []*site.Site {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *SitesStorage) getAll() []*site.Site {
	return p.Sites
}

func (p *SitesStorage) storeAll(items []*site.Site) {
	p.Sites = items
}

// FindByName .
func (p *SitesStorage) FindByName(name string) (*site.Site, bool) {
	p.Lock()
	defer p.Unlock()
	return p.findByName(name)
}

func (p *SitesStorage) findByName(name string) (*site.Site, bool) {
	for _, site := range p.Sites {
		if site.Name == name {
			return site, true
		}
	}
	return nil, false
}

// FindByID .
func (p *SitesStorage) FindByID(id int) (*site.Site, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.findByID(id)
	if !ok {
		_ = p.download()
		return p.findByID(id)
	}
	return item, ok
}

func (p *SitesStorage) findByID(id int) (*site.Site, bool) {
	for _, site := range p.Sites {
		if site.ID == id {
			return site, true
		}
	}
	return nil, false
}

// Download .
func (p *SitesStorage) download() error {
	items, err := Cred.Site().Get()
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
