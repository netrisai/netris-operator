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

// EBGPMetaReconciler reconciles a EBGPMeta object
type EBGPMetaReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=ebgpmeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=ebgpmeta/status,verbs=get;update;patch

func (r *EBGPMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	ebgpMeta := &k8sv1alpha1.EBGPMeta{}
	ebgpCR := &k8sv1alpha1.EBGP{}
	if err := r.Get(context.Background(), req.NamespacedName, ebgpMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, ebgpMeta.Spec.EBGPName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
	}

	provisionState := "Provisioning"

	ebgpNN := req.NamespacedName
	ebgpNN.Name = ebgpMeta.Spec.EBGPName
	if err := r.Get(context.Background(), ebgpNN, ebgpCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if ebgpMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if ebgpMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if ebgpMeta.Spec.Imported {
			logger.Info("Importing ebgp")
			debugLogger.Info("Imported yaml mode. Finding EBGP by name")
			if ebgp, ok := NStorage.EBGPStorage.FindByName(ebgpMeta.Spec.EBGPName); ok {
				debugLogger.Info("Imported yaml mode. EBGP found")
				ebgpMeta.Spec.ID = ebgp.ID
				ebgpCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(ebgp.ModifiedDate/1000), 0))
				err := r.Patch(context.Background(), ebgpMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch ebgpmeta.Spec.ID} %s", err), "")
					return u.patchEBGPStatus(ebgpCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("EBGP imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("EBGP not found for import")
			debugLogger.Info("Imported yaml mode. EBGP not found")
		}

		logger.Info("Creating EBGP")
		if _, err, errMsg := r.createEBGP(ebgpMeta); err != nil {
			logger.Error(fmt.Errorf("{createEBGP} %s", err), "")
			return u.patchEBGPStatus(ebgpCR, "Failure", errMsg.Error())
		}
		logger.Info("EBGP Created")
	} else {
		if apiEBGP, ok := NStorage.EBGPStorage.FindByID(ebgpMeta.Spec.ID); ok {
			ebgpCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(apiEBGP.ModifiedDate/1000), 0))
			debugLogger.Info("Comparing EBGPMeta with Netris EBGP")
			if ok := compareEBGPMetaAPIEBGP(ebgpMeta, apiEBGP); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Something changed")
				debugLogger.Info("Go to update EBGP in Netris")
				logger.Info("Updating EBGP")
				ebgpUpdate, err := EBGPMetaToNetrisUpdate(ebgpMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{EBGPMetaToNetrisUpdate} %s", err), "")
					return u.patchEBGPStatus(ebgpCR, "Failure", err.Error())
				}
				_, err, errMsg := updateEBGP(ebgpUpdate)
				if err != nil {
					logger.Error(fmt.Errorf("{updateEBGP} %s", err), "")
					return u.patchEBGPStatus(ebgpCR, "Failure", errMsg.Error())
				}
				logger.Info("EBGP Updated")
			}
		} else {
			debugLogger.Info("EBGP not found in Netris")
			debugLogger.Info("Going to create EBGP")
			logger.Info("Creating EBGP")
			if _, err, errMsg := r.createEBGP(ebgpMeta); err != nil {
				logger.Error(fmt.Errorf("{createEBGP} %s", err), "")
				return u.patchEBGPStatus(ebgpCR, "Failure", errMsg.Error())
			}
			logger.Info("EBGP Created")
		}
	}
	return u.patchEBGPStatus(ebgpCR, provisionState, "Success")
}

func (r *EBGPMetaReconciler) createEBGP(ebgpMeta *k8sv1alpha1.EBGPMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", ebgpMeta.Namespace, ebgpMeta.Spec.EBGPName),
		"ebgpName", ebgpMeta.Spec.EBGPCRGeneration,
	).V(int(zapcore.WarnLevel))

	ebgpAdd, err := EBGPMetaToNetris(ebgpMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	reply, err := Cred.AddEBGP(ebgpAdd)
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

	debugLogger.Info("EBGP Created", "id", idStruct.ID)

	ebgpMeta.Spec.ID = idStruct.ID

	err = r.Patch(context.Background(), ebgpMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func (r *EBGPMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.EBGPMeta{}).
		Complete(r)
}

func updateEBGP(vnet *api.APIEBGPUpdate) (ctrl.Result, error, error) {
	reply, err := Cred.UpdateEBGP(vnet)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateEBGP} %s", err), err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateEBGP} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}
