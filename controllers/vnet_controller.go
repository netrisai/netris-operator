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

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	api "github.com/netrisai/netrisapi"
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
	_ = r.Log.WithValues("VNet", req.NamespacedName)
	vnet := &k8sv1alpha1.VNet{}

	if err := r.Get(context.Background(), req.NamespacedName, vnet); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println(req.NamespacedName.String(), "Not found")
			return ctrl.Result{}, nil
		}
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
			return ctrl.Result{}, err
		}
	}

	if vnet.DeletionTimestamp != nil {
		fmt.Println("K8S: GO TO DELETE")
		return r.deleteVNet(vnet, vnetMeta)
	}

	if metaFound {
		fmt.Println("K8S: META FOUND")

		if vnet.GetGeneration() != vnetMeta.Spec.VnetCRGeneration {
			fmt.Println("K8S: Generating New Meta")
			vnetID := vnetMeta.Spec.ID
			newVnetMeta, err := r.VnetToVnetMeta(vnet)
			if err != nil {
				return ctrl.Result{}, err
			}
			vnetMeta.Spec = newVnetMeta.DeepCopy().Spec
			vnetMeta.Spec.ID = vnetID
			vnetMeta.Spec.VnetCRGeneration = vnet.GetGeneration()

			err = r.Update(context.Background(), vnetMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				fmt.Println(err)
			}
		}
	} else {
		fmt.Println("K8S: META NOT FOUND")
		if vnet.GetFinalizers() == nil {
			vnet.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})
			err := r.Patch(context.Background(), vnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		vnetMeta, err := r.VnetToVnetMeta(vnet)
		if err != nil {
			return ctrl.Result{}, err
		}

		vnetMeta.Spec.VnetCRGeneration = vnet.GetGeneration()

		if err := r.Create(context.Background(), vnetMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			fmt.Println(err)
		}
	}

	return ctrl.Result{}, nil
}

func updateVNet(vnet *api.APIVNetUpdate) (ctrl.Result, error) {
	reply, err := Cred.ValidateVNet(vnet)
	if err != nil {
		return ctrl.Result{}, err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf(resp.Message)
	}

	reply, err = Cred.UpdateVNet(vnet)
	if err != nil {
		return ctrl.Result{}, err
	}
	resp, err = api.ParseAPIResponse(reply.Data)
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil
}

func (r *VNetReconciler) deleteVNet(vnet *k8sv1alpha1.VNet, vnetMeta *k8sv1alpha1.VNetMeta) (ctrl.Result, error) {
	if vnetMeta != nil {
		_, err := r.deleteVnetMetaCR(vnetMeta)
		if err != nil {
			return ctrl.Result{}, err
		}

		if vnetMeta.Spec.ID > 0 {
			reply, err := Cred.DeleteVNet(vnetMeta.Spec.ID, []int{1})

			if err != nil {
				fmt.Println(err)
				return ctrl.Result{}, err
			}
			resp, err := api.ParseAPIResponse(reply.Data)
			if !resp.IsSuccess {
				if resp.Message != "Invalid circuit ID" {
					return ctrl.Result{}, fmt.Errorf(resp.Message)
				}
			}
		}

	}
	return r.deleteVnetCR(vnet)
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
