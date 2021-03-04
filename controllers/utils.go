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
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	api "github.com/netrisai/netrisapi"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func ignoreDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			fmt.Println("UPDATE EVENT")
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			fmt.Println("DELETE EVENT")
			// Evaluates to false if the object has been confirmed deleted.
			return true
		},
	}
}

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

func getVNet(id int) (vnet *api.APIVNet, err error) {
	vnets, err := Cred.GetVNets()
	if err != nil {
		return vnet, err
	}
	for _, v := range vnets {
		vid, err := strconv.Atoi(v.ID)
		if err != nil {
			return vnet, err
		}
		if vid == id {
			return v, nil
		}
	}

	return vnet, fmt.Errorf("VNet not found in Netris")
}

func toJSON(s interface{}) string {
	js, _ := json.Marshal(s)
	return string(js)
}
