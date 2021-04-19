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
	"net"
	"time"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
)

var ModifiedDateFormat = "02/Jan/06 15:04:05"

func makeGateway(gateway k8sv1alpha1.VNetGateway) k8sv1alpha1.VNetMetaGateway {
	version := ""
	ip, ipNet, err := net.ParseCIDR(gateway.String())
	if err != nil {
		fmt.Println(err)
		return k8sv1alpha1.VNetMetaGateway{}
	}

	if len(ip.To4()) == net.IPv4len {
		version = "ipv4"
	} else {
		version = "ipv6"
	}

	gwLength, _ := ipNet.Mask.Size()
	apiGateway := k8sv1alpha1.VNetMetaGateway{
		Gateway:  ip.String(),
		GwLength: gwLength,
		Version:  version,
	}
	return apiGateway
}

func regParser(valueMatch []string, subexpNames []string) map[string]string {
	result := make(map[string]string)
	for i, name := range subexpNames {
		if i != 0 && name != "" {
			result[name] = valueMatch[i]
		}
	}
	return result
}

func fromTimestampToString(timestamp int) string {
	return time.Unix(int64(timestamp/1000), 0).Format(ModifiedDateFormat)
}
