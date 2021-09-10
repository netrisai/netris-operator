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
	"time"

	api "github.com/netrisai/netriswebapi/v2"
)

var Cred *api.Clientset

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
	*BGPStorage
	*L4LBStorage
	*SubnetsStorage
	*HWsStorage
}

// NewStorage .
func NewStorage(cred *api.Clientset) *Storage {
	Cred = cred
	return &Storage{
		PortsStorage:   NewPortStorage(),
		SitesStorage:   NewSitesStorage(),
		TenantsStorage: NewTenantsStorage(),
		VNetStorage:    NewVNetStorage(),
		BGPStorage:     NewBGPStoragee(),
		L4LBStorage:    NewL4LBStorage(),
		SubnetsStorage: NewSubnetsStorage(),
		HWsStorage:     NewHWsStorage(),
	}
}

// Download .
func (s *Storage) Download() error {
	s.Lock()
	defer s.Unlock()
	if err := s.PortsStorage.Download(); err != nil {
		fmt.Println("PortsStorage", err)
		return err
	}
	if err := s.SitesStorage.Download(); err != nil {
		fmt.Println("SitesStorage", err)
		return err
	}
	if err := s.TenantsStorage.Download(); err != nil {
		fmt.Println("TenantsStorage", err)
		return err
	}
	if err := s.VNetStorage.Download(); err != nil {
		fmt.Println("VNetStorage", err)
		return err
	}
	if err := s.BGPStorage.Download(); err != nil {
		fmt.Println("BGPStorage", err)
		return err
	}
	if err := s.L4LBStorage.Download(); err != nil {
		fmt.Println("L4LBStorage", err)
		return err
	}
	if err := s.SubnetsStorage.Download(); err != nil {
		fmt.Println("SubnetsStorage", err)
		return err
	}
	if err := s.HWsStorage.Download(); err != nil {
		fmt.Println("HWsStorage", err)
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
