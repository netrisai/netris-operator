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
	"strconv"
	"time"

	"github.com/go-logr/logr"
	api "github.com/netrisai/netrisapi"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
)

// BGPMetaReconciler reconciles a BGPMeta object
type BGPMetaReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=bgpmeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=bgpmeta/status,verbs=get;update;patch

func (r *BGPMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	bgpMeta := &k8sv1alpha1.BGPMeta{}
	bgpCR := &k8sv1alpha1.BGP{}
	if err := r.Get(context.Background(), req.NamespacedName, bgpMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, bgpMeta.Spec.BGPName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
	}

	provisionState := "Provisioning"

	bgpNN := req.NamespacedName
	bgpNN.Name = bgpMeta.Spec.BGPName
	if err := r.Get(context.Background(), bgpNN, bgpCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if bgpMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if bgpMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if bgpMeta.Spec.Imported {
			logger.Info("Importing bgp")
			debugLogger.Info("Imported yaml mode. Finding BGP by name")
			if bgp, ok := NStorage.BGPStorage.FindByName(bgpMeta.Spec.BGPName); ok {
				debugLogger.Info("Imported yaml mode. BGP found")
				bgpMeta.Spec.ID = bgp.ID
				bgpCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(bgp.ModifiedDate/1000), 0))
				bgpCR.Status.BGPState = fmt.Sprintf("bgp: %s; prefix: %s; time: %s", bgp.BgpState, bgp.BgpPrefixes, bgp.BgpUptime)
				bgpCR.Status.PortState = bgp.PortStatus
				bgpCR.Status.TerminateOnSwitch = bgp.TermSwName
				if bgp.Vlan > 1 {
					bgpCR.Status.VLANID = strconv.Itoa(bgp.Vlan)
				} else {
					bgpCR.Status.VLANID = "untagged"
				}

				err := r.Patch(context.Background(), bgpMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch bgpmeta.Spec.ID} %s", err), "")
					return u.patchBGPStatus(bgpCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("BGP imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("BGP not found for import")
			debugLogger.Info("Imported yaml mode. BGP not found")
		}

		logger.Info("Creating BGP")
		if _, err, errMsg := r.createBGP(bgpMeta); err != nil {
			logger.Error(fmt.Errorf("{createBGP} %s", err), "")
			return u.patchBGPStatus(bgpCR, "Failure", errMsg.Error())
		}
		logger.Info("BGP Created")
	} else {
		if apiBGP, ok := NStorage.BGPStorage.FindByID(bgpMeta.Spec.ID); ok {
			bgpCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(apiBGP.ModifiedDate/1000), 0))
			bgpCR.Status.BGPState = fmt.Sprintf("bgp: %s; prefix: %s; time: %s", apiBGP.BgpState, apiBGP.BgpPrefixes, apiBGP.BgpUptime)
			bgpCR.Status.PortState = apiBGP.PortStatus
			bgpCR.Status.TerminateOnSwitch = apiBGP.TermSwName
			if apiBGP.Vlan > 1 {
				bgpCR.Status.VLANID = strconv.Itoa(apiBGP.Vlan)
			} else {
				bgpCR.Status.VLANID = "untagged"
			}
			debugLogger.Info("Comparing BGPMeta with Netris BGP")
			if ok := compareBGPMetaAPIEBGP(bgpMeta, apiBGP); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Something changed")
				debugLogger.Info("Go to update BGP in Netris")
				logger.Info("Updating BGP")
				bgpUpdate, err := BGPMetaToNetrisUpdate(bgpMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{BGPMetaToNetrisUpdate} %s", err), "")
					return u.patchBGPStatus(bgpCR, "Failure", err.Error())
				}
				_, err, errMsg := updateBGP(bgpUpdate)
				if err != nil {
					logger.Error(fmt.Errorf("{updateBGP} %s", err), "")
					return u.patchBGPStatus(bgpCR, "Failure", errMsg.Error())
				}
				logger.Info("BGP Updated")
			}
		} else {
			debugLogger.Info("BGP not found in Netris")
			debugLogger.Info("Going to create BGP")
			logger.Info("Creating BGP")
			if _, err, errMsg := r.createBGP(bgpMeta); err != nil {
				logger.Error(fmt.Errorf("{createBGP} %s", err), "")
				return u.patchBGPStatus(bgpCR, "Failure", errMsg.Error())
			}
			logger.Info("BGP Created")
		}
	}
	return u.patchBGPStatus(bgpCR, provisionState, "Success")
}

func (r *BGPMetaReconciler) createBGP(bgpMeta *k8sv1alpha1.BGPMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", bgpMeta.Namespace, bgpMeta.Spec.BGPName),
		"bgpName", bgpMeta.Spec.BGPCRGeneration,
	).V(int(zapcore.WarnLevel))

	bgpAdd, err := BGPMetaToNetris(bgpMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	reply, err := Cred.AddEBGP(bgpAdd)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf(resp.Message), fmt.Errorf(resp.Message)
	}

	idStruct := api.APIEBGPAddReply{}
	err = api.CustomDecode(resp.Data, &idStruct)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("BGP Created", "id", idStruct.ID)

	bgpMeta.Spec.ID = idStruct.ID

	err = r.Patch(context.Background(), bgpMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func (r *BGPMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.BGPMeta{}).
		Complete(r)
}

func updateBGP(vnet *api.APIEBGPUpdate) (ctrl.Result, error, error) {
	reply, err := Cred.UpdateEBGP(vnet)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateBGP} %s", err), err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateBGP} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}
