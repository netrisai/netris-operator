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

package controllers

import (
	"fmt"
	"strconv"
	"time"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/configloader"
	"github.com/netrisai/netris-operator/netrisstorage"
	"github.com/netrisai/netriswebapi/v2/types/vnet"
)

func init() {
	if configloader.Root.RequeueInterval > 0 {
		requeueInterval = time.Duration(time.Duration(configloader.Root.RequeueInterval) * time.Second)
		contextTimeout = requeueInterval
	}
}

func (r *VNetReconciler) getPortsMeta(portNames []k8sv1alpha1.VNetSwitchPort) ([]k8sv1alpha1.VNetMetaMember, error) {
	members := []k8sv1alpha1.VNetMetaMember{}
	hwPorts := make(map[string]*vnet.VNetAddPort)
	for _, port := range portNames {
		vlanID := "1"
		if port.VlanID > 1 {
			vlanID = strconv.Itoa(port.VlanID)
		}

		state := "active"
		if len(port.State) > 0 {
			if port.State == "active" || port.State == "disabled" {
				state = port.State
			}
		}

		hwPorts[port.Name] = &vnet.VNetAddPort{
			Vlan:  vlanID,
			Lacp:  "off",
			State: state,
		}

	}
	for portName := range hwPorts {
		if port, yes := r.NStorage.PortsStorage.FindByName(portName); yes {
			hwPorts[portName].ID = port.ID
			hwPorts[portName].Name = portName
			hwPorts[portName].Lacp = "off"
		} else {
			return members, fmt.Errorf("port '%s' not found", portName)
		}
	}

	for _, member := range hwPorts {
		members = append(members, k8sv1alpha1.VNetMetaMember{
			Name:  member.Name,
			Lacp:  member.Lacp,
			State: member.State,
			ID:    member.ID,
			Vlan:  member.Vlan,
		})
	}
	return members, nil
}

func getSites(names []string, nStorage *netrisstorage.Storage) map[string]int {
	siteList := map[string]int{}
	for _, name := range names {
		siteList[name] = 0
	}
	for siteName := range siteList {
		if site, ok := nStorage.SitesStorage.FindByName(siteName); ok {
			siteList[siteName] = site.ID
		}
	}
	return siteList
}
