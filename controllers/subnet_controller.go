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
	"context"
	"fmt"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	"github.com/netrisai/netriswebapi/http"
	api "github.com/netrisai/netriswebapi/v2"
)

// SubnetReconciler reconciles a Subnet object
type SubnetReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=subnets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=subnets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=subnets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Subnet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *SubnetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	subnet := &k8sv1alpha1.Subnet{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	subnetCtx, subnetCancel := context.WithTimeout(cntxt, contextTimeout)
	defer subnetCancel()
	if err := r.Get(subnetCtx, req.NamespacedName, subnet); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	subnetMetaNamespaced := req.NamespacedName
	subnetMetaNamespaced.Name = string(subnet.GetUID())
	subnetMeta := &k8sv1alpha1.SubnetMeta{}
	metaFound := true

	subnetMetaCtx, subnetMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer subnetMetaCancel()
	if err := r.Get(subnetMetaCtx, subnetMetaNamespaced, subnetMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			subnetMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if subnet.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteSubnet(subnet, subnetMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteSubnet} %s", err), "")
			return u.patchSubnetStatus(subnet, "Failure", err.Error())
		}
		logger.Info("Subnet deleted")
		return ctrl.Result{}, nil
	}

	if subnetMustUpdateAnnotations(subnet) {
		debugLogger.Info("Setting default annotations")
		subnetUpdateDefaultAnnotations(subnet)
		subnetPatchCtx, subnetPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer subnetPatchCancel()
		err := r.Patch(subnetPatchCtx, subnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Subnet default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if subnetCompareFieldsForNewMeta(subnet, subnetMeta) {
			debugLogger.Info("Generating New Meta")
			subnetID := subnetMeta.Spec.ID
			newSubnetMeta, err := r.SubnetToSubnetMeta(subnet)
			if err != nil {
				logger.Error(fmt.Errorf("{SubnetToSubnetMeta} %s", err), "")
				return u.patchSubnetStatus(subnet, "Failure", err.Error())
			}
			subnetMeta.Spec = newSubnetMeta.DeepCopy().Spec
			subnetMeta.Spec.ID = subnetID
			subnetMeta.Spec.SubnetCRGeneration = subnet.GetGeneration()

			subnetMetaUpdateCtx, subnetMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer subnetMetaUpdateCancel()
			err = r.Update(subnetMetaUpdateCtx, subnetMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{subnetMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if subnet.GetFinalizers() == nil {
			subnet.SetFinalizers([]string{"vnet.k8s.netris.ai/delete"})

			subnetPatchCtx, subnetPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer subnetPatchCancel()
			err := r.Patch(subnetPatchCtx, subnet.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Subnet Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		subnetMeta, err := r.SubnetToSubnetMeta(subnet)
		if err != nil {
			logger.Error(fmt.Errorf("{SubnetToSubnetMeta} %s", err), "")
			return u.patchSubnetStatus(subnet, "Failure", err.Error())
		}

		subnetMeta.Spec.SubnetCRGeneration = subnet.GetGeneration()

		subnetMetaCreateCtx, subnetMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer subnetMetaCreateCancel()
		if err := r.Create(subnetMetaCreateCtx, subnetMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{subnetMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *SubnetReconciler) deleteSubnet(subnet *k8sv1alpha1.Subnet, subnetMeta *k8sv1alpha1.SubnetMeta) (ctrl.Result, error) {
	if subnetMeta != nil && subnetMeta.Spec.ID > 0 && !subnetMeta.Spec.Reclaim {
		reply, err := r.Cred.IPAM().Delete("subnet", subnetMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteSubnet} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess && resp.Meta.StatusCode != 400 {
			return ctrl.Result{}, fmt.Errorf("{deleteSubnet} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(subnet, subnetMeta)
}

func (r *SubnetReconciler) deleteCRs(subnet *k8sv1alpha1.Subnet, subnetMeta *k8sv1alpha1.SubnetMeta) (ctrl.Result, error) {
	if subnetMeta != nil {
		_, err := r.deleteSubnetMetaCR(subnetMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteSubnetCR(subnet)
}

func (r *SubnetReconciler) deleteSubnetCR(subnet *k8sv1alpha1.Subnet) (ctrl.Result, error) {
	subnet.ObjectMeta.SetFinalizers(nil)
	subnet.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, subnet.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteSubnetCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *SubnetReconciler) deleteSubnetMetaCR(subnetMeta *k8sv1alpha1.SubnetMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, subnetMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteSubnetMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubnetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Subnet{}).
		Complete(r)
}
