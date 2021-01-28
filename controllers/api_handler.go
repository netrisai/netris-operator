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
	Cred.InsecureVerify(configloader.Root.Controller.Insecure)
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

func getPortsMeta(portNames []k8sv1alpha1.VNetSwitchPort) []k8sv1alpha1.VNetMetaMember {
	hwPorts := make(map[string]*api.APIVNetMember)
	portIsUntagged := false
	for _, port := range portNames {
		vlanID := 1
		if port.VlanID > 0 {
			vlanID = port.VlanID
		}
		if vlanID == 1 {
			portIsUntagged = true
		}
		hwPorts[port.Name] = &api.APIVNetMember{
			VLANID:         vlanID,
			PortIsUntagged: portIsUntagged,
		}
	}
	for portName := range hwPorts {
		if port, yes := NStorage.PortsStorage.FindByName(portName); yes {
			hwPorts[portName].PortID = port.ID
			hwPorts[portName].PortName = portName
			hwPorts[portName].TenantID = port.TenantID
			hwPorts[portName].MemberState = port.MemberState
			hwPorts[portName].LACP = "off"
			hwPorts[portName].ParentPort = port.ParentPort
			// hwPorts[portName].Name = port.SlavePortName
		}
	}
	members := []k8sv1alpha1.VNetMetaMember{}
	for _, member := range hwPorts {
		members = append(members, k8sv1alpha1.VNetMetaMember{
			ChildPort:      member.ChildPort,
			LACP:           member.LACP,
			MemberState:    member.MemberState,
			ParentPort:     member.ParentPort,
			PortIsUntagged: member.PortIsUntagged,
			PortID:         member.PortID,
			PortName:       member.PortName,
			TenantID:       member.TenantID,
			VLANID:         member.VLANID,
		})
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
