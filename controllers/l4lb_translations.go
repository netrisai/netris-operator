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
	"regexp"
	"strconv"
	"strings"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/calicowatcher"
	"github.com/netrisai/netriswebapi/v2/types/ipam"
	"github.com/netrisai/netriswebapi/v2/types/l4lb"
	"github.com/r3labs/diff/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// L4LBToL4LBMeta converts the VNet resource to VNetMeta type and used for add the VNet for Netris API.
func (r *L4LBReconciler) L4LBToL4LBMeta(l4lb *k8sv1alpha1.L4LB) (*k8sv1alpha1.L4LBMeta, error) {
	bReg := regexp.MustCompile(`^(?P<ip>(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])):(?P<port>([1-9]|[1-9][0-9]{1,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-4]))$`)

	tenantID := 0
	siteID := 0
	var state string
	var timeout string
	proto := "tcp"

	l4lbMetaBackends := []k8sv1alpha1.L4LBMetaBackend{}
	ipForTenant := ""

	for _, backend := range l4lb.Spec.Backend {
		valueMatch := bReg.FindStringSubmatch(string(backend))
		result := regParser(valueMatch, bReg.SubexpNames())
		port, err := strconv.Atoi(result["port"])
		if err != nil {
			return nil, err
		}
		ipForTenant = result["ip"]
		l4lbMetaBackends = append(l4lbMetaBackends, k8sv1alpha1.L4LBMetaBackend{
			IP:   result["ip"],
			Port: port,
		})
	}

	if l4lb.Spec.OwnerTenant == "" {
		if r.L4LBTenant != "" {
			l4lb.Spec.OwnerTenant = r.L4LBTenant
		} else {
			subnet, err := calicowatcher.FindIPAMByIP(ipForTenant, r.NStorage.SubnetsStorage.GetAll())
			if err != nil {
				return nil, err
			}

			tenantID = subnet.Tenant.ID
		}
	}

	if tenantID == 0 {
		tenant, ok := r.NStorage.TenantsStorage.FindByName(l4lb.Spec.OwnerTenant)
		if !ok {
			return nil, fmt.Errorf("Tenant '%s' not found", l4lb.Spec.OwnerTenant)
		}
		tenantID = tenant.ID
	}

	if l4lb.Spec.Site == "" {
		siteid, err := r.findSiteByIP(ipForTenant)
		if err != nil {
			return nil, err
		}
		siteID = siteid
	}

	if siteID == 0 {
		if site, ok := r.NStorage.SitesStorage.FindByName(l4lb.Spec.Site); ok {
			siteID = site.ID
		} else {
			return nil, fmt.Errorf("'%s' site not found", l4lb.Spec.Site)
		}
	}

	if l4lb.Spec.State == "" || l4lb.Spec.State == "active" {
		state = "enable"
	} else {
		state = l4lb.Spec.State
	}

	healthCheck := &k8sv1alpha1.L4LBMetaHealthCheck{}

	if l4lb.Spec.Protocol != "" {
		proto = l4lb.Spec.Protocol
	}

	if proto == "tcp" {
		if l4lb.Spec.Check.Timeout == 0 {
			timeout = "2000"
		} else {
			timeout = strconv.Itoa(l4lb.Spec.Check.Timeout)
		}

		if l4lb.Spec.Check.Type == "tcp" || l4lb.Spec.Check.Type == "" {
			healthCheck.TCP = &k8sv1alpha1.L4LBMetaHealthCheckTCP{
				Timeout: timeout,
			}
		} else if l4lb.Spec.Check.Type == "http" {
			healthCheck.HTTP = &k8sv1alpha1.L4LBMetaHealthCheckHTTP{
				Timeout:     timeout,
				RequestPath: l4lb.Spec.Check.RequestPath,
			}
		}
	}

	imported := false
	reclaim := false
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	automatic := false
	if l4lb.Spec.Frontend.IP == "" {
		automatic = true
	}

	l4lbMeta := &k8sv1alpha1.L4LBMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(l4lb.GetUID()),
			Namespace: l4lb.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.L4LBMetaSpec{
			Imported:    imported,
			Reclaim:     reclaim,
			L4LBName:    l4lb.Name,
			SiteID:      siteID,
			SiteName:    l4lb.Spec.Site,
			Tenant:      tenantID,
			Status:      state,
			Automatic:   automatic,
			Protocol:    strings.ToUpper(proto),
			Port:        l4lb.Spec.Frontend.Port,
			IP:          l4lb.Spec.Frontend.IP,
			Backend:     l4lbMetaBackends,
			HealthCheck: healthCheck,
		},
	}

	return l4lbMeta, nil
}

func compareL4LBMetaAPIL4LBHealthCheck(l4lbMetaHealthCheck k8sv1alpha1.L4LBMetaHealthCheck, apiL4LBHealthCheck l4lb.LBHealthCheck) bool {
	var convertedAPIHealthCheck k8sv1alpha1.L4LBMetaHealthCheck

	if apiL4LBHealthCheck.TCP.Timeout != "" {
		convertedAPIHealthCheck.TCP = &k8sv1alpha1.L4LBMetaHealthCheckTCP{
			Timeout:     apiL4LBHealthCheck.TCP.Timeout,
			RequestPath: apiL4LBHealthCheck.TCP.RequestPath,
		}
	} else if apiL4LBHealthCheck.HTTP.Timeout != "" {
		convertedAPIHealthCheck.HTTP = &k8sv1alpha1.L4LBMetaHealthCheckHTTP{
			Timeout:     apiL4LBHealthCheck.HTTP.Timeout,
			RequestPath: apiL4LBHealthCheck.HTTP.RequestPath,
		}
	}

	changelog, _ := diff.Diff(l4lbMetaHealthCheck, convertedAPIHealthCheck)
	return len(changelog) <= 0
}

func compareL4LBMetaAPIL4LBBackend(l4lbMetaBackends []k8sv1alpha1.L4LBMetaBackend, apiL4LBBackends []l4lb.LBBackend) bool {
	type member struct {
		Port string `diff:"port"`
		IP   string `diff:"ip"`
	}

	l4lbBackends := []member{}
	apiBackends := []member{}

	for _, m := range l4lbMetaBackends {
		l4lbBackends = append(l4lbBackends, member{
			Port: strconv.Itoa(m.Port),
			IP:   m.IP,
		})
	}

	for _, m := range apiL4LBBackends {
		apiBackends = append(apiBackends, member{
			Port: m.Port,
			IP:   m.IP,
		})
	}

	changelog, _ := diff.Diff(l4lbBackends, apiBackends)
	return len(changelog) <= 0
}

func compareL4LBMetaAPIL4LB(l4lbMeta *k8sv1alpha1.L4LBMeta, apiL4LB *l4lb.LoadBalancer) bool {
	if l4lbMeta.Spec.L4LBName != apiL4LB.Name {
		return false
	}
	if l4lbMeta.Spec.IP != apiL4LB.IP {
		return false
	}
	if l4lbMeta.Spec.Automatic != apiL4LB.Automatic {
		return false
	}
	if l4lbMeta.Spec.Port != apiL4LB.Port {
		return false
	}
	if l4lbMeta.Spec.Protocol != apiL4LB.Protocol {
		return false
	}
	if l4lbMeta.Spec.SiteID != apiL4LB.SiteID {
		return false
	}
	if l4lbMeta.Spec.Tenant != apiL4LB.TenantID {
		return false
	}
	if l4lbMeta.Spec.Status != apiL4LB.Status {
		return false
	}
	if ok := compareL4LBMetaAPIL4LBHealthCheck(*l4lbMeta.Spec.HealthCheck, apiL4LB.HealthCheck); !ok {
		return false
	}
	if ok := compareL4LBMetaAPIL4LBBackend(l4lbMeta.Spec.Backend, apiL4LB.BackendIPs); !ok {
		return false
	}

	return true
}

// L4LBMetaToNetris converts the k8s L4LB resource to Netris type and used for add the L4LB for Netris API.
func L4LBMetaToNetris(l4lbMeta *k8sv1alpha1.L4LBMeta) (*l4lb.LoadBalancerAdd, error) {
	healthCheck := ""
	requestPath := ""
	timeOut := ""

	if l4lbMeta.Spec.Protocol == "TCP" {
		healthCheck = "None"
	}

	if l4lbMeta.Spec.HealthCheck.HTTP != nil {
		healthCheck = "HTTP"
		requestPath = l4lbMeta.Spec.HealthCheck.HTTP.RequestPath
		timeOut = l4lbMeta.Spec.HealthCheck.HTTP.Timeout
	} else if l4lbMeta.Spec.HealthCheck.TCP != nil {
		healthCheck = "TCP"
		requestPath = l4lbMeta.Spec.HealthCheck.TCP.RequestPath
		timeOut = l4lbMeta.Spec.HealthCheck.TCP.Timeout
	}
	lbBackends := []l4lb.LBAddBackend{}

	for _, backend := range l4lbMeta.Spec.Backend {
		lbBackends = append(lbBackends, l4lb.LBAddBackend{
			IP:   backend.IP,
			Port: backend.Port,
		})
	}

	ip := ""
	if !l4lbMeta.Spec.Automatic {
		ip = l4lbMeta.Spec.IP
	}

	l4lbAdd := &l4lb.LoadBalancerAdd{
		Name:        l4lbMeta.Spec.L4LBName,
		Tenant:      l4lbMeta.Spec.Tenant,
		SiteID:      l4lbMeta.Spec.SiteID,
		Automatic:   l4lbMeta.Spec.Automatic,
		Protocol:    l4lbMeta.Spec.Protocol,
		IP:          ip,
		Port:        l4lbMeta.Spec.Port,
		Status:      l4lbMeta.Spec.Status,
		RequestPath: requestPath,
		Timeout:     timeOut,
		Backend:     lbBackends,
	}

	if healthCheck != "" {
		l4lbAdd.HealthCheck = healthCheck
	}

	return l4lbAdd, nil
}

// L4LBMetaToNetrisUpdate converts the k8s L4LB resource to Netris type and used for update the L4LB for Netris API.
func L4LBMetaToNetrisUpdate(l4lbMeta *k8sv1alpha1.L4LBMeta) (*l4lb.LoadBalancerUpdate, error) {
	healthCheck := ""
	requestPath := ""
	timeOut := ""

	if l4lbMeta.Spec.Protocol == "TCP" {
		healthCheck = "None"
	}

	if l4lbMeta.Spec.HealthCheck.HTTP != nil {
		healthCheck = "HTTP"
		requestPath = l4lbMeta.Spec.HealthCheck.HTTP.RequestPath
		timeOut = l4lbMeta.Spec.HealthCheck.HTTP.Timeout
	} else if l4lbMeta.Spec.HealthCheck.TCP != nil {
		healthCheck = "TCP"
		requestPath = l4lbMeta.Spec.HealthCheck.TCP.RequestPath
		timeOut = l4lbMeta.Spec.HealthCheck.TCP.Timeout
	}
	lbBackends := []l4lb.LBBackend{}

	for _, backend := range l4lbMeta.Spec.Backend {
		lbBackends = append(lbBackends, l4lb.LBBackend{
			IP:   backend.IP,
			Port: strconv.Itoa(backend.Port),
		})
	}

	l4lbUpdate := &l4lb.LoadBalancerUpdate{
		Name:        l4lbMeta.Spec.L4LBName,
		TenantID:    l4lbMeta.Spec.Tenant,
		SiteID:      l4lbMeta.Spec.SiteID,
		SiteName:    l4lbMeta.Spec.SiteName,
		Automatic:   l4lbMeta.Spec.Automatic,
		Protocol:    l4lbMeta.Spec.Protocol,
		IP:          l4lbMeta.Spec.IP,
		Port:        l4lbMeta.Spec.Port,
		Status:      l4lbMeta.Spec.Status,
		RequestPath: requestPath,
		Timeout:     timeOut,
		BackendIPs:  lbBackends,
	}

	if healthCheck != "" {
		l4lbUpdate.HealthCheck = healthCheck
	}

	return l4lbUpdate, nil
}

func l4lbCompareFieldsForNewMeta(l4lb *k8sv1alpha1.L4LB, l4lbMeta *k8sv1alpha1.L4LBMeta) bool {
	imported := false
	reclaim := false
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return l4lb.GetGeneration() != l4lbMeta.Spec.L4LBCRGeneration || imported != l4lbMeta.Spec.Imported || reclaim != l4lbMeta.Spec.Reclaim
}

func l4lbMustUpdateAnnotations(l4lb *k8sv1alpha1.L4LB) bool {
	update := false
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func l4lbUpdateDefaultAnnotations(l4lb *k8sv1alpha1.L4LB) {
	imported := "false"
	reclaim := "delete"
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := l4lb.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := l4lb.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	l4lb.SetAnnotations(annotations)
}

func (r *L4LBReconciler) findTenantByIP(ip string) (int, error) {
	tenantID := 0
	subnets, err := r.Cred.IPAM().Get()
	if err != nil {
		return tenantID, err
	}

	subnetChilds := []*ipam.IPAM{}
	for _, subnet := range subnets {
		subnetChilds = append(subnetChilds, subnet.Children...)
	}

	for _, subnet := range subnetChilds {
		ipAddr := net.ParseIP(ip)
		_, ipNet, err := net.ParseCIDR(subnet.Prefix)
		if err != nil {
			return tenantID, err
		}
		if ipNet.Contains(ipAddr) {
			return subnet.Tenant.ID, nil
		}
	}

	return tenantID, fmt.Errorf("There are no subnets for specified IP address %s", ip)
}

func (r *L4LBReconciler) findSiteByIP(ip string) (int, error) {
	siteID := 0
	subnets, err := r.Cred.IPAM().Get()
	if err != nil {
		return siteID, err
	}

	subnetChilds := []*ipam.IPAM{}
	for _, subnet := range subnets {
		subnetChilds = append(subnetChilds, subnet.Children...)
	}

	for _, subnet := range subnetChilds {
		ipAddr := net.ParseIP(ip)
		_, ipNet, err := net.ParseCIDR(subnet.Prefix)
		if err != nil {
			return siteID, err
		}
		if ipNet.Contains(ipAddr) {
			if len(subnet.Sites) > 0 {
				return subnet.Sites[0].ID, nil
			}
		}
	}

	return siteID, fmt.Errorf("There are no sites for specified IP address %s", ip)
}
