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
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"encoding/json"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
			fmt.Println("GO TO DELETE RESOURSE")

		} else {
			log.Printf("r.Get 1: %v", err)
		}
	} else {
		fmt.Println("GO TO CREATE/UPDATE RESOURSE")
		reconciledResourceSpecJSON, err := json.Marshal(reconciledResource.Spec)
		if err != nil {
			log.Printf("reconciledResourceSpecJSON error: %v", err)
		}
		if reconciledResource.Spec.ID == 0 {
			fmt.Println("GO TO CREATE")
			fmt.Println("Create with params: ---", string(reconciledResourceSpecJSON))
			createVNet, err := AddVNet(cred, reconciledResourceSpecJSON)
			if err != nil {
				log.Printf("createVNet error: %v", err)
			}
			fmt.Println(string(createVNet.Data))
			if err := json.Unmarshal(createVNet.Data, &unmarshaledData); err != nil {
				log.Fatalf("vnetResponseUnmarshal error: %v", err)
			}
			isSuccess := unmarshaledData["isSuccess"].(bool)
			if isSuccess {
				vnetID := unmarshaledData["data"].(map[string]interface{})
				fmt.Println("vnetID - ", vnetID["circuitID"])
				vid, err := strconv.Atoi(fmt.Sprint(vnetID["circuitID"]))
				if err != nil {
					fmt.Println("id convert error")
				}
				reconciledResource.Spec.ID = vid
				err = r.Patch(context.Background(), reconciledResource.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					log.Printf("r.Patch( error: %v", err)
				}
			}

		} else {
			fmt.Println("GO TO UPDATE")
		}
	}
	// return ctrl.Result{}, nil
	return ctrl.Result{RequeueAfter: time.Second * 60}, nil

}

// SetupWithManager Resources
func (r *VNetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VNet{}).
		Complete(r)
}

// AddVNet new
func AddVNet(cred *HTTPCred, vnet []byte) (reply HTTPReply, err error) {
	address := cred.URL.String() + conductorAddresses.VNet
	reply, err = cred.Post(address, vnet)
	if err != nil {
		return reply, err
	}

	return reply, nil
}
