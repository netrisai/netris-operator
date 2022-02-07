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
	"github.com/netrisai/netris-operator/api/v1alpha1"
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/netrisstorage"
	"github.com/netrisai/netriswebapi/http"
	api "github.com/netrisai/netriswebapi/v2"
	"github.com/netrisai/netriswebapi/v2/types/inventory"
)

// SwitchMetaReconciler reconciles a SwitchMeta object
type SwitchMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=switchmeta,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=switchmeta/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=switchmeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SwitchMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *SwitchMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	switchMeta := &k8sv1alpha1.SwitchMeta{}
	switchCR := &k8sv1alpha1.Switch{}
	switchMetaCtx, switchMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer switchMetaCancel()
	if err := r.Get(switchMetaCtx, req.NamespacedName, switchMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, switchMeta.Spec.SwitchName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "OK"

	switchNN := req.NamespacedName
	switchNN.Name = switchMeta.Spec.SwitchName
	switchNNCtx, switchNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer switchNNCancel()
	if err := r.Get(switchNNCtx, switchNN, switchCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if switchMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if switchMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if switchMeta.Spec.Imported {
			logger.Info("Importing switch")
			debugLogger.Info("Imported yaml mode. Finding Switch by name")
			if switchH, ok := r.NStorage.HWsStorage.FindSwitchByName(switchMeta.Spec.SwitchName); ok {
				debugLogger.Info("Imported yaml mode. Switch found")
				switchMeta.Spec.ID = switchH.ID
				switchMeta.Spec.MainIP = switchH.MainIP.Address
				switchMeta.Spec.MgmtIP = switchH.MgmtIP.Address
				switchMeta.Spec.ASN = switchH.Asn

				switchMetaPatchCtx, switchMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer switchMetaPatchCancel()
				err := r.Patch(switchMetaPatchCtx, switchMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch switchmeta.Spec.ID} %s", err), "")
					return u.patchSwitchStatus(switchCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Switch imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Switch not found for import")
			debugLogger.Info("Imported yaml mode. Switch not found")
		}

		logger.Info("Creating Switch")
		if _, err, errMsg := r.createSwitch(switchMeta); err != nil {
			logger.Error(fmt.Errorf("{createSwitch} %s", err), "")
			return u.patchSwitchStatus(switchCR, "Failure", errMsg.Error())
		}
		logger.Info("Switch Created")
	} else {
		if apiSwitch, ok := r.NStorage.HWsStorage.FindSwitchByID(switchMeta.Spec.ID); ok {
			debugLogger.Info("Comparing SwitchMeta with Netris Switch")

			if ok := compareSwitchMetaAPIESwitch(switchMeta, apiSwitch, u); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Go to update Switch in Netris")
				logger.Info("Updating Switch")
				switchUpdate, err := SwitchMetaToNetrisUpdate(switchMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{SwitchMetaToNetrisUpdate} %s", err), "")
					return u.patchSwitchStatus(switchCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(switchUpdate)
				debugLogger.Info("switchUpdate", "payload", string(js))

				_, err, errMsg := updateSwitch(switchMeta.Spec.ID, switchUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateSwitch} %s", err), "")
					return u.patchSwitchStatus(switchCR, "Failure", errMsg.Error())
				}
				logger.Info("Switch Updated")
			}

			switchMeta.Spec.MainIP = apiSwitch.MainIP.Address
			switchMeta.Spec.MgmtIP = apiSwitch.MgmtIP.Address
			switchMeta.Spec.ASN = apiSwitch.Asn

		} else {
			debugLogger.Info("Switch not found in Netris")
			debugLogger.Info("Going to create Switch")
			logger.Info("Creating Switch")
			if _, err, errMsg := r.createSwitch(switchMeta); err != nil {
				logger.Error(fmt.Errorf("{createSwitch} %s", err), "")
				return u.patchSwitchStatus(switchCR, "Failure", errMsg.Error())
			}
			logger.Info("Switch Created")
		}
	}

	if _, err := u.updateSwitchIfNeccesarry(switchCR, *switchMeta); err != nil {
		logger.Error(fmt.Errorf("{updateSwitchIfNeccesarry} %s", err), "")
		return u.patchSwitchStatus(switchCR, "Failure", err.Error())
	}

	return u.patchSwitchStatus(switchCR, provisionState, "Success")
}

func (r *SwitchMetaReconciler) createSwitch(switchMeta *k8sv1alpha1.SwitchMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", switchMeta.Namespace, switchMeta.Spec.SwitchName),
		"switchName", switchMeta.Spec.SwitchCRGeneration,
	).V(int(zapcore.WarnLevel))

	switchAdd, err := SwitchMetaToNetris(switchMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(switchAdd)
	debugLogger.Info("switchToAdd", "payload", string(js))

	reply, err := r.Cred.Inventory().AddSwitch(switchAdd)
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

	debugLogger.Info("Switch Created", "id", idStruct.ID)

	switchMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, switchMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateSwitch(id int, switchH *inventory.HWSwitchUpdate, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.Inventory().UpdateSwitch(id, switchH)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateSwitch} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateSwitch} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwitchMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.SwitchMeta{}).
		Complete(r)
}

func (u *uniReconciler) updateSwitchIfNeccesarry(switchCR *v1alpha1.Switch, switchMeta v1alpha1.SwitchMeta) (ctrl.Result, error) {
	shouldUpdateCR := false
	if switchCR.Spec.MainIP == "" && switchCR.Spec.MainIP != switchMeta.Spec.MainIP {
		switchCR.Spec.MainIP = switchMeta.Spec.MainIP
		shouldUpdateCR = true
	}
	if switchCR.Spec.MgmtIP == "" && switchCR.Spec.MgmtIP != switchMeta.Spec.MgmtIP {
		switchCR.Spec.MgmtIP = switchMeta.Spec.MgmtIP
		shouldUpdateCR = true
	}
	if switchCR.Spec.ASN == 0 && switchCR.Spec.ASN != switchMeta.Spec.ASN {
		switchCR.Spec.ASN = switchMeta.Spec.ASN
		shouldUpdateCR = true
	}
	if shouldUpdateCR {
		u.DebugLogger.Info("Updating Switch CR")
		if _, err := u.patchSwitch(switchCR); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
