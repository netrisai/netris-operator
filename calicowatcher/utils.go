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

	"github.com/netrisai/netriswebapi/v2/types/ipam"
)

func findIPAMByIP(ip string, subnets []*ipam.IPAM) (*ipam.IPAM, error) {
	for _, subnet := range subnets {
		ipAddr := net.ParseIP(ip)
		_, ipNet, err := net.ParseCIDR(subnet.Prefix)
		if err != nil {
			return nil, err
		}

		if ipNet.Contains(ipAddr) {
			if len(subnet.Children) > 0 {
				ip, err := findIPAMByIP(ip, subnet.Children)
				if ip != nil {
					return ip, err
				}
			}

			return subnet, nil

		}
	}

	return nil, fmt.Errorf("there are no subnet for specified IP address %s", ip)
}

func FindIPAMByIP(ip string, subnets []*ipam.IPAM) (*ipam.IPAM, error) {
	return findIPAMByIP(ip, subnets)
}
