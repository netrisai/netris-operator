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
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/controllers"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var logger = logrus.New()

func getClientset() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(ctrl.GetConfigOrDie())
}

func start(mgr manager.Manager) {
	clientset, err := getClientset()
	if err != nil {
		logger.Error(err)
	}
	cl := mgr.GetClient()
	loadBalancerProcess(clientset, cl)
}

func Start(mgr manager.Manager) {
	ticker := time.NewTicker(10 * time.Second)
	start(mgr)
	for {
		<-ticker.C
		start(mgr)
	}
}

func filterL4LBs(LBs []k8sv1alpha1.L4LB) []k8sv1alpha1.L4LB {
	lbList := []k8sv1alpha1.L4LB{}
	for _, lb := range LBs {
		if lb.GetAnnotations()["servicename"] != "" && lb.GetAnnotations()["servicenamespace"] != "" && lb.GetAnnotations()["serviceuid"] != "" {
			lbList = append(lbList, lb)
		}
	}
	return lbList
}

func loadBalancerProcess(clientset *kubernetes.Clientset, cl client.Client) {
	logger.Debug("Generating load balancers from k8s...")
	var errors []error = nil
	lbTimeout := "2000"
	serviceLBs, err := generateLoadBalancers(clientset, lbTimeout)
	if err != nil {
		logger.Error(err)
	}

	l4lbs, err := getL4LBs(cl)
	if err != nil {
		logger.Error(err)
	}

	if l4lbs == nil {
		logger.Error("CRD Not found")
		return
	}

	filteerdL4LBs := filterL4LBs(l4lbs.Items)

	ipAuto := make(map[string]string)
	for _, lb := range filteerdL4LBs {
		if uid, ok := lb.GetAnnotations()["serviceuid"]; uid != "" && ok {
			ipAuto[uid] = lb.Status.IP
		}
	}

	lbsToCreate, lbsToUpdate, lbsToDelete, ingressIPsMap := compareLoadBalancers(filteerdL4LBs, serviceLBs)

	js, _ := json.Marshal(lbsToCreate)
	logger.Debugf("Load balancers for create: %s\n", js)
	js, _ = json.Marshal(lbsToUpdate)
	logger.Debugf("Load balancers for update: %s\n", js)
	js, _ = json.Marshal(lbsToDelete)
	logger.Debugf("Load balancers for delete: %s\n", js)
	js, _ = json.Marshal(ingressIPsMap)
	logger.Debugf("Ingress addresses for k8s: %s\n", js)

	lbsByUID := make(map[string][]*k8sv1alpha1.L4LB)
	for _, lb := range lbsToCreate {
		lbsByUID[lb.GetAnnotations()["serviceuid"]] = append(lbsByUID[lb.GetAnnotations()["serviceuid"]], lb)
	}

	errors = append(errors, deleteL4LBs(cl, lbsToDelete)...)

	errs := updateL4LBs(cl, lbsToUpdate)
	errors = append(errors, errs...)

	for _, lbs := range lbsByUID {
		errs = createL4LBs(cl, lbs, ipAuto)
		errors = append(errors, errs...)
	}

	for _, serviceLB := range serviceLBs {
		ingressIPs := []string{}
		if ingress, ok := ingressIPsMap[serviceLB.GetAnnotations()["serviceuid"]]; ok {
			for ip := range ingress {
				ingressIPs = append(ingressIPs, ip)
			}
			_, err := assignIngress(clientset, ingressIPs, serviceLB.GetAnnotations()["servicenamespace"], serviceLB.GetAnnotations()["servicename"])
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	for _, err := range errors {
		logger.Error(err)
	}
}

func deleteL4LBs(cl client.Client, lbs []k8sv1alpha1.L4LB) []error {
	var errors []error
	for _, lb := range lbs {
		err := deleteL4LB(cl, lb)
		if err != nil {
			errors = append(errors, fmt.Errorf("{deleteL4LBs} %s", err))
		}
	}
	return errors
}

func deleteL4LB(cl client.Client, lb k8sv1alpha1.L4LB) error {
	return cl.Delete(context.Background(), lb.DeepCopyObject(), &client.DeleteOptions{})
}

func updateL4LBs(cl client.Client, lbs []k8sv1alpha1.L4LB) []error {
	var errors []error
	for _, lb := range lbs {
		err := updateL4LB(cl, lb)
		if err != nil {
			errors = append(errors, fmt.Errorf("{updateL4LBs} %s", err))
		}
	}
	return errors
}

func createL4LB(cl client.Client, lb *k8sv1alpha1.L4LB) error {
	return cl.Create(context.Background(), lb.DeepCopyObject(), &client.CreateOptions{})
}

func createL4LBs(cl client.Client, lbs []*k8sv1alpha1.L4LB, ipAuto map[string]string) []error {
	var errors []error
	for _, lb := range lbs {
		if ip, ok := ipAuto[lb.GetAnnotations()["serviceuid"]]; ok && lb.Spec.Frontend.IP == "" {
			if ip == "" {
				break
			} else {
				lb.Spec.Frontend.IP = ip
				err := createL4LB(cl, lb)
				if err != nil {
					errors = append(errors, fmt.Errorf("{createL4LBs} %s", err))
				}
			}
		} else {
			err := createL4LB(cl, lb)
			if err != nil {
				errors = append(errors, fmt.Errorf("{createL4LBs} %s", err))
			}
			if lb.Spec.Frontend.IP == "" {
				break
			}
		}
	}
	return errors
}

func updateL4LB(cl client.Client, lb k8sv1alpha1.L4LB) error {
	return cl.Update(context.Background(), lb.DeepCopyObject(), &client.UpdateOptions{})
}

func compareLoadBalancers(LBs []k8sv1alpha1.L4LB, serviceLBs []*k8sv1alpha1.L4LB) ([]*k8sv1alpha1.L4LB, []k8sv1alpha1.L4LB, []k8sv1alpha1.L4LB, map[string]map[string]int) {
	LBsMap := map[string]k8sv1alpha1.L4LB{}
	IPsMap := make(map[string]string)
	serviceIngressMap := map[string]map[string]int{}
	lbIngressMap := map[string]map[string]int{}

	serviceLBsMap := map[string]*k8sv1alpha1.L4LB{}

	lbsToCreate := []*k8sv1alpha1.L4LB{}
	lbsToDelete := []k8sv1alpha1.L4LB{}
	lbsToUpdate := []k8sv1alpha1.L4LB{}

	for _, serviceLB := range serviceLBs {
		serviceLBsMap[serviceLB.Name] = serviceLB
		if _, ok := serviceIngressMap[serviceLB.GetAnnotations()["serviceuid"]]; !ok {
			serviceIngressMap[serviceLB.GetAnnotations()["serviceuid"]] = make(map[string]int)
		}
		ingressList := strings.Split(serviceLB.GetAnnotations()["serviceingressips"], ",")
		for _, ingress := range ingressList {
			serviceIngressMap[serviceLB.GetAnnotations()["serviceuid"]][ingress] = 1
		}
	}

	for _, lb := range LBs {
		LBsMap[lb.Name] = lb
		if l, ok := serviceLBsMap[lb.Name]; ok {
			IPsMap[l.GetAnnotations()["serviceuid"]] = lb.Spec.Frontend.IP
		}
	}

	if len(serviceLBsMap) > 0 {
		for _, serviceLB := range serviceLBsMap {
			if lb, ok := LBsMap[serviceLB.Name]; ok {
				anns := lb.GetAnnotations()
				anns["servicenamespace"] = serviceLB.GetAnnotations()["servicenamespace"]
				anns["serviceuid"] = serviceLB.GetAnnotations()["serviceuid"]
				anns["servicename"] = serviceLB.GetAnnotations()["servicename"]
				lb.SetAnnotations(anns)
				update := false

				if _, ok := lbIngressMap[serviceLB.GetAnnotations()["serviceuid"]]; !ok {
					lbIngressMap[serviceLB.GetAnnotations()["serviceuid"]] = make(map[string]int)
				}

				lbIngressMap[serviceLB.GetAnnotations()["serviceuid"]][lb.Status.IP] = 1

				if serviceLB.Spec.Frontend.IP != "" {
					lbIngressMap[serviceLB.GetAnnotations()["serviceuid"]] = map[string]int{}
				}

				if serviceLB.Spec.Frontend.IP != "" && serviceLB.Spec.Frontend.IP != lb.Spec.Frontend.IP {
					lb.Spec.Frontend.IP = serviceLB.Spec.Frontend.IP
					update = true
				}

				if serviceLB.Spec.Check.Timeout != lb.Spec.Check.Timeout {
					lb.Spec.Check.Timeout = serviceLB.Spec.Check.Timeout
					update = true
				}

				if !compareBackends(serviceLB.Spec.Backend, lb.Spec.Backend) {
					lb.Spec.Backend = serviceLB.Spec.Backend
					update = true
				}
				if update {
					lbsToUpdate = append(lbsToUpdate, lb)
				}
			} else {
				if ip, ok := IPsMap[serviceLB.GetAnnotations()["serviceuid"]]; ok {
					serviceLB.Spec.Frontend.IP = ip
				}
				lbsToCreate = append(lbsToCreate, serviceLB)
			}
		}
	}

	if len(LBs) > 0 {
		for _, LB := range LBs {
			if _, ok := serviceLBsMap[LB.Name]; !ok {
				lbsToDelete = append(lbsToDelete, LB)
			}
		}
	}

	ingressToUpdate := map[string]map[string]int{}

	for uid, serviceIngress := range serviceIngressMap {
		if !reflect.DeepEqual(serviceIngress, lbIngressMap[uid]) {
			ingressToUpdate[uid] = lbIngressMap[uid]
		}
	}

	return lbsToCreate, lbsToUpdate, lbsToDelete, ingressToUpdate
}

func compareBackends(lbBackends []k8sv1alpha1.L4LBBackend, serviceLBBackends []k8sv1alpha1.L4LBBackend) bool {
	lbBackendMap := make(map[string]int)
	serviceLBBackendMap := make(map[string]int)
	for _, lb := range lbBackends {
		lbBackendMap[string(lb)] = 1
	}

	for _, lb := range serviceLBBackends {
		serviceLBBackendMap[string(lb)] = 1
	}

	return reflect.DeepEqual(lbBackendMap, serviceLBBackendMap)
}

func getL4LBs(cl client.Client) (*k8sv1alpha1.L4LBList, error) {
	l4lb := &k8sv1alpha1.L4LBList{}

	err := cl.List(context.Background(), l4lb, &client.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("{getL4LBs} %s", err)
	}

	return l4lb, nil
}

func generateLoadBalancers(clientset *kubernetes.Clientset, lbTimeout string) ([]*k8sv1alpha1.L4LB, error) {
	lbList := []*k8sv1alpha1.L4LB{}
	serviceList, err := getServices(clientset, "")
	if err != nil {
		return lbList, fmt.Errorf("{generateLoadBalancers} %s", err)
	}

	logger.Debug("Getting k8s pods...")
	podList, err := getPods(clientset, "")
	if err != nil {
		return lbList, fmt.Errorf("{generateLoadBalancers} %s", err)
	}

	timeout, err := strconv.Atoi(lbTimeout)
	if err != nil {
		return lbList, fmt.Errorf("{generateLoadBalancers} %s", err)
	}

	tenant, ok := controllers.NStorage.TenantsStorage.FindByID(1)
	if !ok {
		return lbList, fmt.Errorf("{generateLoadBalancers} Default tenant not found")
	}

	site, ok := controllers.NStorage.SitesStorage.FindByID(1)
	if !ok {
		return lbList, fmt.Errorf("{generateLoadBalancers} Default site not found")
	}

	for _, svc := range serviceList.Items {
		if svc.Spec.Type == "LoadBalancer" {
			selectors := []selector{}
			hostIPs := map[string]int{}
			for key, value := range svc.Spec.Selector {
				selectors = append(selectors, selector{
					Key:   key,
					Value: value,
				})
			}

			pods := []v1.Pod{}
			for _, sel := range selectors {
				pods = append(pods, filterPodsBySelector(podList, sel.Key, sel.Value)...)
			}

			for _, pod := range pods {
				hostIPs[pod.Status.HostIP] = 1
			}

			var lbIPs []lbIP

			var ingressIPs []string

			for _, ingress := range svc.Status.LoadBalancer.Ingress {
				ingressIPs = append(ingressIPs, ingress.IP)
			}
			ingressIPsString := strings.Join(ingressIPs, ",")

			for _, port := range svc.Spec.Ports {
				lbIP := lbIP{
					Name:     port.Name,
					IP:       svc.Spec.LoadBalancerIP,
					Port:     int(port.Port),
					NodePort: int(port.NodePort),
					Protocol: string(port.Protocol),
				}
				if len(svc.Spec.LoadBalancerIP) > 0 {
					lbIP.IP = svc.Spec.LoadBalancerIP
				} else {
					lbIP.Automatic = true
				}
				lbIPs = append(lbIPs, lbIP)
			}

			var hostIPS []string

			for hostIP := range hostIPs {
				hostIPS = append(hostIPS, hostIP)
			}

			if len(lbIPs) > 0 && len(hostIPS) > 0 {
				for _, lbIP := range lbIPs {
					backends := []k8sv1alpha1.L4LBBackend{}
					for _, hostIP := range hostIPS {
						backend := fmt.Sprintf("%s:%d", hostIP, lbIP.NodePort)
						backends = append(backends, k8sv1alpha1.L4LBBackend(backend))
					}

					lb := &k8sv1alpha1.L4LB{
						ObjectMeta: metav1.ObjectMeta{
							Name:      strings.ToLower(fmt.Sprintf("%s-%s-%d", svc.GetUID(), lbIP.Protocol, lbIP.Port)),
							Namespace: svc.GetNamespace(),
						},
						TypeMeta: metav1.TypeMeta{
							Kind:       "L4LB",
							APIVersion: "k8s.netris.ai/v1alpha1",
						},
						Spec: k8sv1alpha1.L4LBSpec{
							OwnerTenant: tenant.Name,
							Site:        site.Name,
							Protocol:    strings.ToLower(lbIP.Protocol),
							Frontend: k8sv1alpha1.L4LBFrontend{
								Port: lbIP.Port,
								IP:   lbIP.IP,
							},
							State: "active",
							Check: k8sv1alpha1.L4LBCheck{
								Type:    "tcp",
								Timeout: timeout,
							},
							Backend: backends,
						},
					}

					anns := make(map[string]string)
					anns["servicenamespace"] = svc.GetNamespace()
					anns["servicename"] = svc.GetName()
					anns["serviceuid"] = string(svc.UID)
					anns["serviceingressips"] = ingressIPsString
					lb.SetAnnotations(anns)

					lbList = append(lbList, lb)
				}
			}
		}
	}
	return lbList, nil
}
