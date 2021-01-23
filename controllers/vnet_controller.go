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
	"context"
	"fmt"
	"log"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VNetReconciler reconciles a VNet object
type VNetReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnets/status,verbs=get;update;patch

// Reconcile vnet events
func (r *VNetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("vnet", req.NamespacedName)
	vnet := &k8sv1alpha1.VNet{}
	if err := r.Get(context.Background(), req.NamespacedName, vnet); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println(req.NamespacedName.String(), "Not found")
			return ctrl.Result{}, nil
		}
		log.Printf("r.Get error: %v\n", err)
		return ctrl.Result{}, err
	}

	vnetMetaNamespaced := req.NamespacedName
	vnetMetaNamespaced.Name = string(vnet.GetUID())
	vnetMeta := &k8sv1alpha1.VNetMeta{}
	metaFound := true
	if err := r.Get(context.Background(), vnetMetaNamespaced, vnetMeta); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println(vnetMetaNamespaced.String(), "Not found")
			metaFound = false
		} else {
			log.Printf("r.Get error: %v\n", err)
			return ctrl.Result{}, err
		}
	}

	if metaFound {
		if vnet.DeletionTimestamp != nil {
			fmt.Println("GO TO DELETE")

			vnetMeta.ObjectMeta.SetFinalizers(nil)
			vnetMeta.SetFinalizers(nil)
			if err := r.Update(context.Background(), vnetMeta.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.Delete(context.Background(), vnetMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
				return ctrl.Result{}, err
			}

			vnet.ObjectMeta.SetFinalizers(nil)
			vnet.SetFinalizers(nil)
			if err := r.Update(context.Background(), vnet.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil

			// return r.deleteVNet(vnet)
		}
	} else {
		vnet.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

		err := r.Update(context.Background(), vnet.DeepCopyObject(), &client.UpdateOptions{}) // requeue
		if err != nil {
			return ctrl.Result{}, err
		}
		vnetMeta, err := r.K8sToVnetMeta(vnet)
		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(context.Background(), vnetMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			fmt.Println(err)
		}
	}

	// vnets, err := Cred.GetVNetsByID(reconciledResource.Spec.ID)
	// if err != nil {
	// 	log.Println(err)
	// 	return ctrl.Result{}, nil
	// }
	// if len(vnets) > 0 {
	// 	// NetrisToK8SVnet(vnets[0])
	// }

	// fmt.Println("GO TO UPDATE")
	// fmt.Println("Nothing changed")
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager Resources
func (r *VNetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VNet{}).
		WithEventFilter(ignoreDeletionPredicate()).
		Complete(r)
}

func (r *VNetReconciler) deleteVNet(reconciledResource *k8sv1alpha1.VNet) (ctrl.Result, error) {
	// reply, err := Cred.DeleteVNet(reconciledResource.Spec.ID, []int{1})

	// if err != nil {
	// 	fmt.Println(err)
	// 	return ctrl.Result{}, err
	// }
	// resp, err := api.ParseAPIResponse(reply.Data)
	// if !resp.IsSuccess {
	// 	fmt.Println(resp.Message)
	// 	return ctrl.Result{}, fmt.Errorf(resp.Message)
	// }

	reconciledResource.ObjectMeta.SetFinalizers(nil)
	reconciledResource.SetFinalizers(nil)

	err := r.Update(context.Background(), reconciledResource.DeepCopyObject(), &client.UpdateOptions{})
	if err != nil {
		fmt.Println(err)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// NetrisToK8SVnet converts the Netris API structure to k8s VNet resource.
// func NetrisToK8SVnet(vnet *api.APIVNetInfo) *v1alpha1.VNet {
// 	k8svnet := &v1alpha1.VNet{}
// 	k8svnet.Name = vnet.Name
// 	k8svnet.Spec.OwnerID = vnet.Owner

// 	var apiSites []*api.APIVNETSite
// 	json.Unmarshal([]byte(vnet.Sites), &apiSites)
// 	sites := make(map[int]*v1alpha1.VNetSite)
// 	for _, site := range apiSites {
// 		for _, member := range vnet.Members {
// 			portName := fmt.Sprintf("%s@%s", member.Port, member.SwitchName)
// 			switchPort := k8sv1alpha1.VNetSwitchPort{
// 				Name:        portName,
// 				VlanID:      member.VlanID,
// 				PortID:      member.PortID,
// 				TenantID:    member.TenantID,
// 				ChildPort:   member.ChildPort,
// 				ParentPort:  member.ParentPort,
// 				MemberState: member.MemberState,
// 			}
// 			if _, ok := sites[site.SiteID]; ok {
// 				if member.SiteID == site.SiteID {
// 					sites[site.SiteID].SwitchPorts = append(sites[site.SiteID].SwitchPorts, switchPort)
// 				}
// 			} else {
// 				sites[site.SiteID] = &v1alpha1.VNetSite{
// 					Name:        site.SiteName,
// 					ID:          site.SiteID,
// 					Gateways:    []k8sv1alpha1.VNetGateway{},
// 					SwitchPorts: []k8sv1alpha1.VNetSwitchPort{switchPort},
// 				}
// 			}
// 		}
// 	}

// 	// js, _ := json.Marshal(sites)
// 	// fmt.Printf("%s\n", js)

// 	return &v1alpha1.VNet{}
// }

// K8sToVnetMeta converts the k8s VNet resource to VNetMeta type and used for add the VNet for Netris API.
func (r *VNetReconciler) K8sToVnetMeta(reconciledResource *k8sv1alpha1.VNet) (*k8sv1alpha1.VNetMeta, error) {
	ports := []k8sv1alpha1.VNetSwitchPort{}
	siteNames := []string{}
	apiGateways := []k8sv1alpha1.VNetMetaGateway{}

	for _, site := range reconciledResource.Spec.Sites {
		siteNames = append(siteNames, site.Name)
		for _, port := range site.SwitchPorts {
			ports = append(ports, port)
		}
		for _, gateway := range site.Gateways {
			apiGateways = append(apiGateways, makeGateway(gateway))
		}
	}

	// fmt.Println(ports)

	prts := getPorts(ports)
	// js, _ := json.Marshal(prts)
	// fmt.Printf("Ports: %s\n", js)

	sites := getSites(siteNames)
	// js, _ = json.Marshal(sites)
	// fmt.Printf("Sites: %s\n", js)

	siteIDs := []int{}
	for _, id := range sites {
		siteIDs = append(siteIDs, id)
	}

	tenantID := 0

	tenant, ok := NStorage.TenantsStorage.FindByName(reconciledResource.Spec.Owner)
	if !ok {
		return nil, fmt.Errorf("Tenant '%s' not found", reconciledResource.Spec.Owner)
	}
	tenantID = tenant.ID

	// fmt.Printf("TenantID: %d\n", tenantID)

	vnetMeta := &k8sv1alpha1.VNetMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(reconciledResource.GetUID()),
			Namespace: "default",
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.VNetMetaSpec{
			Name:     string(reconciledResource.GetUID()),
			Sites:    siteIDs,
			Owner:    tenantID,
			Tenants:  []int{}, // AAAAAAA
			Gateways: apiGateways,
			Members:  prts.String(),

			VaMode:       false,
			VaNativeVLAN: 1,
			VaVLANs:      "",
		},
	}

	vnetMeta.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

	return vnetMeta, nil
}

func (r *VNetReconciler) createVNet(reconciledResource *k8sv1alpha1.VNet) (ctrl.Result, error) {
	vnetMeta, err := r.K8sToVnetMeta(reconciledResource)
	if err != nil {
		fmt.Println(err)
		return ctrl.Result{}, err
	}

	fmt.Println(vnetMeta)
	// reply, err := Cred.AddVNet(vnetAdd)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return ctrl.Result{}, err
	// }
	// resp, err := api.ParseAPIResponse(reply.Data)
	// if !resp.IsSuccess {
	// 	fmt.Println(resp.Message)
	// 	return ctrl.Result{}, fmt.Errorf(resp.Message)
	// }

	// idStruct := api.APIVNetAddReply{}

	// api.CustomDecode(resp.Data, &idStruct)

	// reconciledResource.Spec.ID = idStruct.CircuitID
	// reconciledResource.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

	// err = r.Update(context.Background(), reconciledResource.DeepCopyObject(), &client.UpdateOptions{}) // requeue
	// if err != nil {
	// 	fmt.Println(err)
	// 	return ctrl.Result{}, err
	// }
	return ctrl.Result{}, nil
}
