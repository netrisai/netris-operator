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
	api "github.com/netrisai/netrisapi"
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
			vnetMeta = nil
		} else {
			log.Printf("r.Get error: %v\n", err)
			return ctrl.Result{}, err
		}
	}

	if vnet.DeletionTimestamp != nil {
		fmt.Println("GO TO DELETE")
		return r.deleteVNet(vnet, vnetMeta)
	}

	if metaFound {
		fmt.Println("K8S: META FOUND")
		vnetID := vnetMeta.Spec.ID

		newVnetMeta, err := r.VnetToVnetMeta(vnet)
		if err != nil {
			return ctrl.Result{}, err
		}

		vnetMeta.Spec = newVnetMeta.DeepCopy().Spec
		vnetMeta.Spec.ID = vnetID

		err = r.Update(context.Background(), vnetMeta.DeepCopyObject(), &client.UpdateOptions{})
		if err != nil {
			fmt.Println(err)
		}

		time.Sleep(100 * time.Millisecond)

		if err := r.Get(context.Background(), vnetMetaNamespaced, vnetMeta); err != nil {
			if errors.IsNotFound(err) {
				fmt.Println(vnetMetaNamespaced.String(), "Didn't updated")
			} else {
				log.Printf("r.Get error: %v\n", err)
				return ctrl.Result{}, err
			}
		}

		if vnetMeta.Spec.ID == 0 {
			if _, err := r.createVNet(vnetMeta); err != nil {
				fmt.Println(err)
			}
		} else {
			vnets, err := Cred.GetVNetsByID(vnetMeta.Spec.ID)
			if err != nil {
				return ctrl.Result{}, err
			}
			if len(vnets) == 0 {
				fmt.Println("API: VNet not found")
				fmt.Println("API: Going to create VNet")
				if _, err := r.createVNet(vnetMeta); err != nil {
					fmt.Println(err)
				}
			} else {
				apiVnet := vnets[0]
				if ok := compareVNetMetaAPIVnet(vnetMeta, apiVnet); ok {
					fmt.Println("Nothing Changed")
				} else {
					fmt.Println("SOMETHING CHANGED. GO TO UPDATE")
				}
			}
		}
	} else {
		fmt.Println("K8S: META NOT FOUND")
		vnet.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})
		err := r.Update(context.Background(), vnet.DeepCopyObject(), &client.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, err
		}

		vnetMeta, err := r.VnetToVnetMeta(vnet)
		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(context.Background(), vnetMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			fmt.Println(err)
		}
		time.Sleep(100 * time.Millisecond)
		if err := r.Get(context.Background(), vnetMetaNamespaced, vnetMeta); err != nil {
			if errors.IsNotFound(err) {
				fmt.Println(vnetMetaNamespaced.String(), "Didn't created")
			} else {
				log.Printf("r.Get error: %v\n", err)
				return ctrl.Result{}, err
			}
		}

		if _, err := r.createVNet(vnetMeta); err != nil {
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

func compareVNetMetaAPIVnetGateways(vnetMetaGateways []k8sv1alpha1.VNetMetaGateway, apiVnetGateways []api.APIVNetGateway) bool {
	k8sGateways := make(map[string]k8sv1alpha1.VNetMetaGateway)
	for _, gateway := range vnetMetaGateways {
		k8sGateways[fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength)] = gateway
	}

	netrisGateways := make(map[string]api.APIVNetGateway)
	for _, gateway := range apiVnetGateways {
		netrisGateways[fmt.Sprintf("%s/%d", gateway.Gateway, gateway.GwLength)] = gateway
	}

	for address := range k8sGateways {
		if _, ok := netrisGateways[address]; !ok {
			return false
		}
	}

	return true
}

func compareVNetMetaAPIVnetMembers(vnetMetaMembers []k8sv1alpha1.VNetMetaMember, apiVnetMembers []api.APIVNetInfoMember) bool {
	k8sMembers := make(map[int]k8sv1alpha1.VNetMetaMember)
	for _, member := range vnetMetaMembers {
		k8sMembers[member.PortID] = member
	}

	netrisMembers := make(map[int]api.APIVNetInfoMember)
	for _, member := range apiVnetMembers {
		netrisMembers[member.PortID] = member
	}

	for portID := range k8sMembers {
		if _, ok := netrisMembers[portID]; !ok {
			return false
		}
	}

	return true
}

func compareVNetMetaAPIVnetSites(vnetMetaSites []k8sv1alpha1.VNetMetaSite, apiVnetSites []int) bool {

	k8sSites := make(map[int]string)
	for _, site := range vnetMetaSites {
		k8sSites[site.ID] = ""
	}

	for _, siteID := range apiVnetSites {
		if _, ok := k8sSites[siteID]; !ok {
			return false
		}
	}

	return true
}

func compareVNetMetaAPIVnet(vnetMeta *k8sv1alpha1.VNetMeta, apiVnet *api.APIVNetInfo) bool {
	fmt.Println("Comparing VNetMeta with Netris VNet")

	if ok := compareVNetMetaAPIVnetSites(vnetMeta.Spec.Sites, apiVnet.SitesID); !ok {
		return false
	}
	if ok := compareVNetMetaAPIVnetGateways(vnetMeta.Spec.Gateways, apiVnet.Gateways); !ok {
		return false
	}
	if ok := compareVNetMetaAPIVnetMembers(vnetMeta.Spec.Members, apiVnet.Members); !ok {
		return false
	}

	if vnetMeta.Spec.OwnerID != apiVnet.Owner {
		return false
	}

	apiVaMode := false
	if apiVnet.VaMode > 0 {
		apiVaMode = true
	}

	if vnetMeta.Spec.VaMode != apiVaMode {
		return false
	}

	if vnetMeta.Spec.VaVLANs != apiVnet.VaVlans {
		return false
	}

	return true
}

func (r *VNetReconciler) deleteVNet(vnet *k8sv1alpha1.VNet, vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	if vnetMeta != nil && vnetMeta.Spec.ID > 0 {
		reply, err := Cred.DeleteVNet(vnetMeta.Spec.ID, []int{1})

		if err != nil {
			fmt.Println(err)
			return ctrl.Result{}, err
		}
		resp, err := api.ParseAPIResponse(reply.Data)
		if !resp.IsSuccess {
			fmt.Println(resp.Message)
			return ctrl.Result{}, fmt.Errorf(resp.Message)
		}
	}
	return r.deleteCRs(vnet, vnetMeta)
}

func (r *VNetReconciler) deleteCRs(vnet *k8sv1alpha1.VNet, vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	if vnetMeta != nil {
		_, err := r.deleteVnetMetaCR(vnetMeta)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return r.deleteVnetCR(vnet)
}

func (r *VNetReconciler) deleteVnetCR(vnet *k8sv1alpha1.VNet) (ctrl.Result, error) {
	vnet.ObjectMeta.SetFinalizers(nil)
	vnet.SetFinalizers(nil)
	if err := r.Update(context.Background(), vnet.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VNetReconciler) deleteVnetMetaCR(vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	vnetMeta.ObjectMeta.SetFinalizers(nil)
	vnetMeta.SetFinalizers(nil)
	if err := r.Update(context.Background(), vnetMeta.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Delete(context.Background(), vnetMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager Resources
func (r *VNetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VNet{}).
		WithEventFilter(ignoreDeletionPredicate()).
		Complete(r)
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

// VnetToVnetMeta converts the VNet resource to VNetMeta type and used for add the VNet for Netris API.
func (r *VNetReconciler) VnetToVnetMeta(vnet *k8sv1alpha1.VNet) (*k8sv1alpha1.VNetMeta, error) {
	ports := []k8sv1alpha1.VNetSwitchPort{}
	siteNames := []string{}
	apiGateways := []k8sv1alpha1.VNetMetaGateway{}

	for _, site := range vnet.Spec.Sites {
		siteNames = append(siteNames, site.Name)
		for _, port := range site.SwitchPorts {
			ports = append(ports, port)
		}
		for _, gateway := range site.Gateways {
			apiGateways = append(apiGateways, makeGateway(gateway))
		}
	}
	prts := getPortsMeta(ports)

	sites := getSites(siteNames)
	sitesList := []k8sv1alpha1.VNetMetaSite{}

	for name, id := range sites {
		sitesList = append(sitesList, k8sv1alpha1.VNetMetaSite{
			Name: name,
			ID:   id,
		})
	}

	tenantID := 0

	tenant, ok := NStorage.TenantsStorage.FindByName(vnet.Spec.Owner)
	if !ok {
		return nil, fmt.Errorf("Tenant '%s' not found", vnet.Spec.Owner)
	}
	tenantID = tenant.ID

	vnetMeta := &k8sv1alpha1.VNetMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(vnet.GetUID()),
			Namespace: "default",
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.VNetMetaSpec{
			Name:     string(vnet.GetUID()),
			VnetName: vnet.Name,
			Sites:    sitesList,
			OwnerID:  tenantID,
			Tenants:  []int{}, // AAAAAAA
			Gateways: apiGateways,
			Members:  prts,

			VaMode:       false,
			VaNativeVLAN: 1,
			VaVLANs:      "",
		},
	}

	vnetMeta.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

	return vnetMeta, nil
}

// VnetMetaToNetris converts the k8s VNet resource to Netris type and used for add the VNet for Netris API.
func VnetMetaToNetris(vnetMeta *k8sv1alpha1.VNetMeta) (*api.APIVNetAdd, error) {
	ports := []k8sv1alpha1.VNetMetaMember{}
	siteNames := []string{}
	apiGateways := []api.APIVNetGateway{}

	for _, site := range vnetMeta.Spec.Sites {
		siteNames = append(siteNames, site.Name)
	}
	for _, port := range vnetMeta.Spec.Members {
		ports = append(ports, port)
	}
	for _, gateway := range vnetMeta.Spec.Gateways {
		apiGateways = append(apiGateways, api.APIVNetGateway{
			Gateway:  gateway.Gateway,
			GwLength: gateway.GwLength,
			ID:       gateway.ID,
			Version:  gateway.Version,
		})
	}

	prts := getPorts(ports)

	sites := getSites(siteNames)
	siteIDs := []int{}
	for _, id := range sites {
		siteIDs = append(siteIDs, id)
	}

	vnetAdd := &api.APIVNetAdd{
		Name:         vnetMeta.Spec.VnetName,
		Sites:        siteIDs,
		Owner:        vnetMeta.Spec.OwnerID,
		Tenants:      []int{}, // AAAAAAA
		Gateways:     apiGateways,
		Members:      prts.String(),
		VaMode:       false,
		VaNativeVLAN: 1,
		VaVLANs:      "",
	}

	return vnetAdd, nil
}

func (r *VNetReconciler) createVNet(vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	vnetAdd, err := VnetMetaToNetris(vnetMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	reply, err := Cred.AddVNet(vnetAdd)
	if err != nil {
		return ctrl.Result{}, err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf(resp.Message)
	}

	idStruct := api.APIVNetAddReply{}
	api.CustomDecode(resp.Data, &idStruct)

	fmt.Printf("VNet Created. ID: %d\n", idStruct.CircuitID)

	vnetMeta.Spec.ID = idStruct.CircuitID
	vnetMeta.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

	err = r.Update(context.Background(), vnetMeta.DeepCopyObject(), &client.UpdateOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
