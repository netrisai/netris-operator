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
	"time"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	api "github.com/netrisai/netrisapi"
)

// L4LBMetaReconciler reconciles a L4LBMeta object
type L4LBMetaReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.netris.ai,resources=l4lbmeta,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.netris.ai,resources=l4lbmeta/status,verbs=get;update;patch

func (r *L4LBMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	l4lbMeta := &k8sv1alpha1.L4LBMeta{}
	l4lbCR := &k8sv1alpha1.L4LB{}
	if err := r.Get(context.Background(), req.NamespacedName, l4lbMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, l4lbMeta.Spec.L4LBName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
	}

	provisionState := ""

	l4lbNN := req.NamespacedName
	l4lbNN.Name = l4lbMeta.Spec.L4LBName
	if err := r.Get(context.Background(), l4lbNN, l4lbCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if l4lbMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if l4lbMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if l4lbMeta.Spec.Imported {
			logger.Info("Importing l4lb")
			debugLogger.Info("Imported yaml mode. Finding L4LB by name")
			if l4lb, ok := NStorage.L4LBStorage.findByName(l4lbMeta.Spec.L4LBName); ok {
				debugLogger.Info("Imported yaml mode. L4LB found")
				l4lbMeta.Spec.ID = l4lb.ID
				l4lbCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(l4lb.ModifiedDate/1000), 0))
				err := r.Patch(context.Background(), l4lbMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch l4lbMeta.Spec.ID} %s", err), "")
					return u.patchL4LBStatus(l4lbCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("L4LB imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("L4LB not found for import")
			debugLogger.Info("Imported yaml mode. L4LB not found")
		}

		logger.Info("Creating L4LB")
		if _, err, errMsg := r.createL4LB(l4lbMeta); err != nil {
			logger.Error(fmt.Errorf("{createL4LB} %s", err), "")
			return u.patchL4LBStatus(l4lbCR, "Failure", errMsg.Error())
		}
		logger.Info("L4LB Created")
	} else {
		apiL4LB, ok := NStorage.L4LBStorage.FindByID(l4lbMeta.Spec.ID)
		if !ok {
			debugLogger.Info("L4LB not found in Netris")
			debugLogger.Info("Going to create L4LB")
			logger.Info("Creating L4LB")
			if _, err, errMsg := r.createL4LB(l4lbMeta); err != nil {
				logger.Error(fmt.Errorf("{createL4LB} %s", err), "")
				return u.patchL4LBStatus(l4lbCR, "Failure", errMsg.Error())
			}
			logger.Info("L4LB Created")
		} else {
			l4lbCR.Status.ModifiedDate = metav1.NewTime(time.Unix(int64(apiL4LB.ModifiedDate/1000), 0))
			debugLogger.Info("Comparing L4LBMeta with Netris L4LB")
			if ok := compareL4LBMetaAPIL4LB(l4lbMeta, apiL4LB); ok {
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Something changed")
				debugLogger.Info("Go to update L4LB in Netris")
				logger.Info("Updating L4LB")
				l4lbUpdate, err := L4LBMetaToNetrisUpdate(l4lbMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{VnetMetaToNetrisUpdate} %s", err), "")
					return u.patchL4LBStatus(l4lbCR, "Failure", err.Error())
				}
				_, err, errMsg := updateL4LB(l4lbUpdate)
				if err != nil {
					logger.Error(fmt.Errorf("{updateL4LB} %s", err), "")
					return u.patchL4LBStatus(l4lbCR, "Failure", errMsg.Error())
				}
				logger.Info("L4LB Updated")
			}
			provisionState = apiL4LB.Label.Text
			fmt.Println("AAAA", apiL4LB.Label)
		}
	}
	return u.patchL4LBStatus(l4lbCR, provisionState, "Successfully reconciled")
}

func (r *L4LBMetaReconciler) createL4LB(l4lbMeta *k8sv1alpha1.L4LBMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", l4lbMeta.Namespace, l4lbMeta.Spec.L4LBName),
		"l4lbName", l4lbMeta.Spec.L4LBName,
	).V(int(zapcore.WarnLevel))

	l4lbAdd, err := L4LBMetaToNetris(l4lbMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	reply, err := Cred.AddLB4(l4lbAdd)
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

	var id int
	err = api.CustomDecode(resp.Data, &id)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("L4LB Created", "id", id)

	l4lbMeta.Spec.ID = id

	err = r.Patch(context.Background(), l4lbMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", id)
	return ctrl.Result{}, nil, nil
}

func updateL4LB(l4lb *api.APIUpdateLoadBalancer) (ctrl.Result, error, error) {
	js, err := json.Marshal(l4lb)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateL4LB} %s", err), err
	}

	reply, err := Cred.UpdateLB4(js)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateL4LB} %s", err), err
	}
	resp, err := api.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateL4LB} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

func (r *L4LBMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.L4LBMeta{}).
		Complete(r)
}
