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

	"encoding/json"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/netrisai/netrisapi"

	"github.com/netrisai/netris-operator/api/v1alpha1"
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
)

// VNetReconciler reconciles a VNet object
type VNetReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// unmarshaledData struct
var unmarshaledData map[string]interface{}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnets/status,verbs=get;update;patch

// Reconcile vnet events
func (r *VNetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("vnet", req.NamespacedName)
	reconciledResource := &k8sv1alpha1.VNet{}
	err := r.Get(context.Background(), req.NamespacedName, reconciledResource)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Printf("r.Get error: %v\n", err)
		return ctrl.Result{}, err
	}
	if reconciledResource.Spec.ID == 0 {
		fmt.Println("GO TO CREATE")
		reconciledResourceSpecJSON, err := json.Marshal(reconciledResource.Spec)
		if err != nil {
			log.Println(err)
			return ctrl.Result{}, nil
		}
		fmt.Println("Reconcile", string(reconciledResourceSpecJSON))
		return r.createVNet(reconciledResource)
	}
	if reconciledResource.DeletionTimestamp != nil {
		fmt.Println("GO TO DELETE")
		return r.deleteVNet(reconciledResource)
	}
	fmt.Println("GO TO UPDATE")
	fmt.Println("Nothing changed")
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
	reply, err := cred.DeleteVNet(reconciledResource.Spec.ID, []int{1})

	if err != nil {
		fmt.Println(err)
		return ctrl.Result{}, err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if !resp.IsSuccess {
		fmt.Println(resp.Message)
		return ctrl.Result{}, fmt.Errorf(resp.Message)
	}

	reconciledResource.ObjectMeta.SetFinalizers(nil)
	reconciledResource.SetFinalizers(nil)

	err = r.Update(context.Background(), reconciledResource.DeepCopyObject(), &client.UpdateOptions{})
	if err != nil {
		fmt.Println(err)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// NetrisToK8SVnet converts the Netris API structure to k8s VNet resource.
func NetrisToK8SVnet(*api.APIVNetAdd) *v1alpha1.VNet {
	return &v1alpha1.VNet{}
}

// K8sToNetrisVnetAdd converts the k8s VNet resource to Netris API type and used for add the VNet for Netris API.
func (r *VNetReconciler) K8sToNetrisVnetAdd(reconciledResource *k8sv1alpha1.VNet) *api.APIVNetAdd {
	ports := []k8sv1alpha1.VNetSwitchPort{}
	siteNames := []string{}
	apiGateways := []api.APIVNetGateway{}

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

	tenantID := getTenantID(reconciledResource.Spec.Owner)
	// fmt.Printf("TenantID: %d\n", tenantID)

	vnetAdd := &api.APIVNetAdd{
		Name:     reconciledResource.Name,
		Sites:    siteIDs,
		Owner:    tenantID,
		Tenants:  []int{}, // AAAAAAA
		Gateways: apiGateways,
		Members:  prts.String(),

		VaMode:       false,
		VaNativeVLAN: 1,
		VaVLANs:      "",
	}

	return vnetAdd
}

func (r *VNetReconciler) createVNet(reconciledResource *k8sv1alpha1.VNet) (ctrl.Result, error) {
	vnetAdd := r.K8sToNetrisVnetAdd(reconciledResource)
	reply, err := cred.AddVNet(vnetAdd)
	if err != nil {
		fmt.Println(err)
		return ctrl.Result{}, err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if !resp.IsSuccess {
		fmt.Println(resp.Message)
		return ctrl.Result{}, fmt.Errorf(resp.Message)
	}

	idStruct := api.APIVNetAddReply{}

	api.CustomDecode(resp.Data, &idStruct)

	reconciledResource.Spec.ID = idStruct.CircuitID
	reconciledResource.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

	err = r.Update(context.Background(), reconciledResource.DeepCopyObject(), &client.UpdateOptions{}) // requeue
	if err != nil {
		fmt.Println(err)
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}
