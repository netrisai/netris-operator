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

package calicowatcher

import (
	"fmt"
	"net"
	"strconv"

	api "github.com/netrisai/netrisapi"
)

func (w *Watcher) findSiteByIP(ip string) (int, string, error) {
	siteID := 0
	subnets := w.NStorage.SubnetsStorage.GetAll()

	subnetChilds := []api.APISubnetChild{}
	for _, subnet := range subnets {
		subnetChilds = append(subnetChilds, subnet.Children...)
	}

	for _, subnet := range subnetChilds {
		ipAddr := net.ParseIP(ip)
		_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", subnet.Prefix, subnet.Length))
		if err != nil {
			return siteID, "", err
		}
		if ipNet.Contains(ipAddr) {
			sID, _ := strconv.Atoi(subnet.SiteID)
			if err != nil {
				return siteID, "", err
			}
			return sID, ipNet.String(), nil
		}
	}

	return siteID, "", fmt.Errorf("There are no sites  for specified IP address %s", ip)
}
