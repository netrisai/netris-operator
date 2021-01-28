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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	api "github.com/netrisai/netrisapi"
)

// VNetMetaReconciler reconciles a VNetMeta object
type VNetMetaReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnetmeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=vnetmeta/status,verbs=get;update;patch

// Reconcile .
func (r *VNetMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("vnetMeta", req.NamespacedName)

	vnetMeta := &k8sv1alpha1.VNetMeta{}
	if err := r.Get(context.Background(), req.NamespacedName, vnetMeta); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println(req.NamespacedName.String(), "Not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if vnetMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if vnetMeta.Spec.ID == 0 {
		fmt.Println("K8S: ID Not found in meta. Creating VNet")
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
			fmt.Println("K8S: Comparing VnetMeta with Netris Vnet")
			if ok := compareVNetMetaAPIVnet(vnetMeta, apiVnet); ok {
				fmt.Println("K8S: Nothing Changed")
			} else {
				fmt.Println("K8S: SOMETHING CHANGED. GO TO UPDATE")
				updateVnet, err := VnetMetaToNetrisUpdate(vnetMeta)
				if err != nil {
					fmt.Println(err)
				}
				_, err = updateVNet(updateVnet)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager .
func (r *VNetMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.VNetMeta{}).
		Complete(r)
}

func (r *VNetMetaReconciler) createVNet(vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
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

	fmt.Printf("API: VNet Created: ID: %d\n", idStruct.CircuitID)

	vnetMeta.Spec.ID = idStruct.CircuitID
	vnetMeta.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

	err = r.Patch(context.Background(), vnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err
	}

	fmt.Println("K8S: ID patched to meta")
	return ctrl.Result{}, nil
}
