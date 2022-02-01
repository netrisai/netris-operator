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
	"encoding/json"
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
	"github.com/netrisai/netriswebapi/v2/types/ipam"
)

// SubnetMetaReconciler reconciles a SubnetMeta object
type SubnetMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=subnetmeta,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=subnetmeta/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=subnetmeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SubnetMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *SubnetMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	subnetMeta := &k8sv1alpha1.SubnetMeta{}
	subnetCR := &k8sv1alpha1.Subnet{}
	subnetMetaCtx, subnetMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer subnetMetaCancel()
	if err := r.Get(subnetMetaCtx, req.NamespacedName, subnetMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, subnetMeta.Spec.SubnetName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "Provisioning"

	subnetNN := req.NamespacedName
	subnetNN.Name = subnetMeta.Spec.SubnetName
	subnetNNCtx, subnetNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer subnetNNCancel()
	if err := r.Get(subnetNNCtx, subnetNN, subnetCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if subnetMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if subnetMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if subnetMeta.Spec.Imported {
			logger.Info("Importing subnet")
			debugLogger.Info("Imported yaml mode. Finding Subnet by name")
			if subnet, ok := r.NStorage.SubnetsStorage.FindByName(subnetMeta.Spec.SubnetName); ok {
				debugLogger.Info("Imported yaml mode. Subnet found")
				subnetMeta.Spec.ID = subnet.ID

				subnetMetaPatchCtx, subnetMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer subnetMetaPatchCancel()
				err := r.Patch(subnetMetaPatchCtx, subnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch subnetmeta.Spec.ID} %s", err), "")
					return u.patchSubnetStatus(subnetCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Subnet imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Subnet not found for import")
			debugLogger.Info("Imported yaml mode. Subnet not found")
		}

		logger.Info("Creating Subnet")
		if _, err, errMsg := r.createSubnet(subnetMeta); err != nil {
			logger.Error(fmt.Errorf("{createSubnet} %s", err), "")
			return u.patchSubnetStatus(subnetCR, "Failure", errMsg.Error())
		}
		logger.Info("Subnet Created")
	} else {
		if apiSubnet, ok := r.NStorage.SubnetsStorage.FindByID(subnetMeta.Spec.ID); ok {
			debugLogger.Info("Comparing SubnetMeta with Netris Subnet")
			if ok := compareSubnetMetaAPIESubnet(subnetMeta, apiSubnet, u); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Go to update Subnet in Netris")
				logger.Info("Updating Subnet")
				subnetUpdate, err := SubnetMetaToNetrisUpdate(subnetMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{SubnetMetaToNetrisUpdate} %s", err), "")
					return u.patchSubnetStatus(subnetCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(subnetUpdate)
				debugLogger.Info("subnetUpdate", "payload", string(js))

				_, err, errMsg := updateSubnet(subnetMeta.Spec.ID, subnetUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateSubnet} %s", err), "")
					return u.patchSubnetStatus(subnetCR, "Failure", errMsg.Error())
				}
				logger.Info("Subnet Updated")
			}
		} else {
			debugLogger.Info("Subnet not found in Netris")
			debugLogger.Info("Going to create Subnet")
			logger.Info("Creating Subnet")
			if _, err, errMsg := r.createSubnet(subnetMeta); err != nil {
				logger.Error(fmt.Errorf("{createSubnet} %s", err), "")
				return u.patchSubnetStatus(subnetCR, "Failure", errMsg.Error())
			}
			logger.Info("Subnet Created")
		}
	}
	return u.patchSubnetStatus(subnetCR, provisionState, "Success")
}

func (r *SubnetMetaReconciler) createSubnet(subnetMeta *k8sv1alpha1.SubnetMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", subnetMeta.Namespace, subnetMeta.Spec.SubnetName),
		"subnetName", subnetMeta.Spec.SubnetCRGeneration,
	).V(int(zapcore.WarnLevel))

	subnetAdd, err := SubnetMetaToNetris(subnetMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(subnetAdd)
	debugLogger.Info("subnetToAdd", "payload", string(js))

	reply, err := r.Cred.IPAM().AddSubnet(subnetAdd)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	idStruct := struct {
		ID int `json:"id"`
	}{}

	data, err := reply.Parse()
	if err != nil {
		return ctrl.Result{}, err, err
	}

	if reply.StatusCode != 200 {
		return ctrl.Result{}, fmt.Errorf(data.Message), fmt.Errorf(data.Message)
	}

	idStruct.ID = int(data.Data.(map[string]interface{})["id"].(float64))

	debugLogger.Info("Subnet Created", "id", idStruct.ID)

	subnetMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, subnetMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateSubnet(id int, subnet *ipam.Subnet, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.IPAM().UpdateSubnet(id, subnet)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateSubnet} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateSubnet} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubnetMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.SubnetMeta{}).
		Complete(r)
}
