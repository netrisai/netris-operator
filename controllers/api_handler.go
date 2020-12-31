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
	"log"

	api "github.com/netrisai/netrisapi"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/configloader"
)

// Cred .
var Cred *api.HTTPCred

// NStorage .
var NStorage = NewStorage()

func init() {
	var err error
	Cred, err = api.NewHTTPCredentials(configloader.Root.Controller.Host, configloader.Root.Controller.Login, configloader.Root.Controller.Password, 10)
	if err != nil {
		log.Panicf("newHTTPCredentials error %v", err)
	}
	err = Cred.LoginUser()
	if err != nil {
		log.Printf("LoginUser error %v", err)
	}
	go Cred.CheckAuthWithInterval()
	err = NStorage.Download()
	if err != nil {
		log.Printf("Storage.Download() error %v", err)
	}
	go NStorage.DownloadWithInterval()
}

func getPorts(portNames []k8sv1alpha1.VNetSwitchPort) *api.APIVNetMembers {
	hwPorts := make(map[string]*api.APIVNetMember)
	for _, port := range portNames {
		vlanID := 1
		if port.VlanID > 0 {
			vlanID = port.VlanID
		}
		if port.PortIsUntagged {
			vlanID = 1
		}
		hwPorts[port.Name] = &api.APIVNetMember{
			VLANID:         vlanID,
			PortIsUntagged: port.PortIsUntagged,
		}
	}
	for portName := range hwPorts {
		if port, yes := NStorage.PortsStorage.FindByName(portName); yes {
			hwPorts[portName].PortID = port.ID
			hwPorts[portName].PortName = port.PortNameFull
			hwPorts[portName].TenantID = port.TenantID
			hwPorts[portName].MemberState = port.MemberState
			hwPorts[portName].LACP = "off"
			hwPorts[portName].ParentPort = port.ParentPort
			// hwPorts[portName].Name = port.SlavePortName
		}
	}
	members := &api.APIVNetMembers{}
	for _, member := range hwPorts {
		members.Add(*member)
	}
	return members
}

func getSites(names []string) map[string]int {
	siteList := map[string]int{}
	for _, name := range names {
		siteList[name] = 0
	}
	for siteName := range siteList {
		if site, ok := NStorage.SitesStorage.FindByName(siteName); ok {
			siteList[siteName] = site.ID
		}
	}
	return siteList
}

func getTenantID(name string) int {
	tenants, err := Cred.GetTenants()
	if err != nil {
		fmt.Println(err)
	}
	for _, tenant := range tenants {
		if tenant.Name == name {
			return tenant.ID
		}
	}
	return 0
}
