package controllers

import (
	"fmt"
	"log"

	api "github.com/netrisai/netrisapi"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/configloader"
)

var cred *api.HTTPCred

func init() {
	var err error
	cred, err = api.NewHTTPCredentials(configloader.Root.Controller.Host, configloader.Root.Controller.Login, configloader.Root.Controller.Password, 10)
	if err != nil {
		log.Panicf("newHTTPCredentials error %v", err)
	}
	err = cred.LoginUser()
	if err != nil {
		log.Printf("LoginUser error %v", err)
	}
	go cred.CheckAuthWithInterval()
}

type midPortType struct {
	Name           string
	ChildPort      int    `json:"childPort"`
	LACP           string `json:"lacp"`
	MemberState    string `json:"member_state"`
	ParentPort     int    `json:"parentPort"`
	PortIsUntagged bool   `json:"portIsUntagged"`
	PortID         int    `json:"port_id"`
	PortName       string `json:"port_name"`
	TenantID       int    `json:"tenant_id"`
	VLANID         int    `json:"vlan_id"`
}

func getPorts(portNames []k8sv1alpha1.VNetSwitchPort) *api.APIVNetMembers {
	hwPorts := make(map[string]*api.APIVNetMember)
	for _, port := range portNames {
		vlanID := 1
		if port.VlanID > 0 {
			vlanID = port.VlanID
		}
		hwPorts[port.Name] = &api.APIVNetMember{
			VLANID:         vlanID,
			PortIsUntagged: port.PortIsUntagged,
		}
	}
	ports, err := cred.GetPorts()
	if err != nil {
		fmt.Println(err)
	}
	for _, port := range ports {
		portName := fmt.Sprintf("%s@%s", port.SlavePortName, port.SwitchName)
		if _, ok := hwPorts[portName]; ok {
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
	sites, err := cred.GetSites()
	if err != nil {
		fmt.Println(err)
	}
	for _, site := range sites {
		if _, ok := siteList[site.Name]; ok {
			siteList[site.Name] = site.ID
		}
	}
	return siteList
}

func getTenantID(name string) int {
	tenants, err := cred.GetTenants()
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
