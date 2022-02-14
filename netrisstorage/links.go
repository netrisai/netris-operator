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

	"github.com/netrisai/netriswebapi/v2/types/link"
)

// LinksStorage .
type LinksStorage struct {
	sync.Mutex
	Links []*link.Link
}

// NewLinksStorage .
func NewLinksStorage() *LinksStorage {
	return &LinksStorage{}
}

// GetAll .
func (p *LinksStorage) GetAll() []*link.Link {
	p.Lock()
	defer p.Unlock()
	return p.getAll()
}

func (p *LinksStorage) getAll() []*link.Link {
	return p.Links
}

func (p *LinksStorage) storeAll(items []*link.Link) {
	p.Links = items
}

// Find .
func (p *LinksStorage) Find(local, remote int) (*link.Link, bool) {
	p.Lock()
	defer p.Unlock()
	item, ok := p.find(local, remote)
	if !ok {
		_ = p.download()
		return p.find(local, remote)
	}
	return item, ok
}

func (p *LinksStorage) find(local, remote int) (*link.Link, bool) {
	for _, link := range p.Links {
		if (link.Local.ID == local && link.Remote.ID == remote) || (link.Local.ID == remote && link.Remote.ID == local) {
			return link, true
		}
	}
	return nil, false
}

// Download .
func (p *LinksStorage) download() error {
	items, err := Cred.Link().Get()
	if err != nil {
		return err
	}
	p.storeAll(items)
	return nil
}

// Download .
func (p *LinksStorage) Download() error {
	p.Lock()
	defer p.Unlock()
	return p.download()
}
