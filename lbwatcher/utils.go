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

package lbwatcher

import (
	"fmt"
	"net"

	"github.com/netrisai/netriswebapi/v2/types/ipam"
)

func (w *Watcher) findSiteByIP(ip string) (ipam.IDName, string, error) {
	var site ipam.IDName
	subnets := w.NStorage.SubnetsStorage.GetAll()

	subnetChilds := []*ipam.IPAM{}
	for _, subnet := range subnets {
		subnetChilds = append(subnetChilds, subnet.Children...)
	}

	for _, subnet := range subnetChilds {
		ipAddr := net.ParseIP(ip)
		_, ipNet, err := net.ParseCIDR(subnet.Prefix)
		if err != nil {
			return site, "", err
		}
		if ipNet.Contains(ipAddr) {
			if len(subnet.Sites) > 0 {
				return subnet.Sites[0], ipNet.String(), nil
			}
		}
	}

	return site, "", fmt.Errorf("There are no sites for specified IP address %s", ip)
}
