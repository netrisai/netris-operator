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
	"strconv"
	"strings"

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
	"github.com/netrisai/netriswebapi/v2/types/link"
)

// LinkMetaReconciler reconciles a LinkMeta object
type LinkMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=linkmeta,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=linkmeta/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=linkmeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LinkMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *LinkMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	linkMeta := &k8sv1alpha1.LinkMeta{}
	linkCR := &k8sv1alpha1.Link{}
	linkMetaCtx, linkMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer linkMetaCancel()
	if err := r.Get(linkMetaCtx, req.NamespacedName, linkMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, linkMeta.Spec.LinkName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "OK"

	linkNN := req.NamespacedName
	linkNN.Name = linkMeta.Spec.LinkName
	linkNNCtx, linkNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer linkNNCancel()
	if err := r.Get(linkNNCtx, linkNN, linkCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if linkMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if linkMeta.Spec.ID == "" {
		debugLogger.Info("ID Not found in meta")
		if linkMeta.Spec.Imported {
			logger.Info("Importing link")
			debugLogger.Info("Imported yaml mode. Finding Link by name")
			if link, ok := r.NStorage.LinksStorage.Find(linkMeta.Spec.Local, linkMeta.Spec.Remote); ok {
				debugLogger.Info("Imported yaml mode. Link found")
				linkMeta.Spec.ID = fmt.Sprintf("%d-%d", link.Local.ID, link.Remote.ID)
				linkMetaPatchCtx, linkMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer linkMetaPatchCancel()
				err := r.Patch(linkMetaPatchCtx, linkMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch linkmeta.Spec.ID} %s", err), "")
					return u.patchLinkStatus(linkCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Link imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Link not found for import")
			debugLogger.Info("Imported yaml mode. Link not found")
		}

		logger.Info("Creating Link")
		if _, err, errMsg := r.createLink(linkMeta); err != nil {
			logger.Error(fmt.Errorf("{createLink} %s", err), "")
			return u.patchLinkStatus(linkCR, "Failure", errMsg.Error())
		}
		logger.Info("Link Created")
	} else {
		oldID := strings.Split(linkMeta.Spec.ID, "-")
		oldLocal, _ := strconv.Atoi(oldID[0])
		oldRemote, _ := strconv.Atoi(oldID[1])
		if _, ok := r.NStorage.LinksStorage.Find(oldLocal, oldRemote); ok {
			local := 0
			remote := 0
			if o, ok := r.NStorage.PortsStorage.FindByName(string(linkCR.Spec.Ports[0])); ok {
				local = o.ID
			} else {
				logger.Error(fmt.Errorf("Couldn't find port %s", linkCR.Spec.Ports[0]), "")
				return u.patchLinkStatus(linkCR, "Failure", fmt.Sprintf("Couldn't find port %s", linkCR.Spec.Ports[0]))
			}
			if d, ok := r.NStorage.PortsStorage.FindByName(string(linkCR.Spec.Ports[1])); ok {
				remote = d.ID
			} else {
				logger.Error(fmt.Errorf("Couldn't find port %s", linkCR.Spec.Ports[0]), "")
				return u.patchLinkStatus(linkCR, "Failure", fmt.Sprintf("Couldn't find port %s", linkCR.Spec.Ports[0]))
			}

			if (local == oldLocal && remote == oldRemote) || (local == oldRemote && remote == oldLocal) {
				debugLogger.Info("Nothing Changed")
			} else {
				linkDelete := &link.Link{
					Local:  link.LinkIDName{ID: oldLocal},
					Remote: link.LinkIDName{ID: oldRemote},
				}
				reply, err := r.Cred.Link().Delete(linkDelete)
				if err != nil {
					return ctrl.Result{}, fmt.Errorf("{deleteLink} %s", err)
				}
				resp, err := http.ParseAPIResponse(reply.Data)
				if err != nil {
					return ctrl.Result{}, err
				}

				if !resp.IsSuccess {
					return ctrl.Result{}, fmt.Errorf("{deleteLink} %s", fmt.Errorf(resp.Message))
				}

				linkMeta.Spec.ID = ""
				lCtx, cancel := context.WithTimeout(cntxt, contextTimeout)
				defer cancel()
				err = r.Patch(lCtx, linkMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
				if err != nil {
					logger.Error(fmt.Errorf("{patchLinkID} %s", err), "")
					return u.patchLinkStatus(linkCR, "Failure", err.Error())
				}
			}
		} else {
			debugLogger.Info("Link not found in Netris")
			debugLogger.Info("Going to create Link")
			logger.Info("Creating Link")
			if _, err, errMsg := r.createLink(linkMeta); err != nil {
				logger.Error(fmt.Errorf("{createLink} %s", err), "")
				return u.patchLinkStatus(linkCR, "Failure", errMsg.Error())
			}
			logger.Info("Link Created")
		}
	}

	return u.patchLinkStatus(linkCR, provisionState, "Success")
}

func (r *LinkMetaReconciler) createLink(linkMeta *k8sv1alpha1.LinkMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", linkMeta.Namespace, linkMeta.Spec.LinkName),
		"linkName", linkMeta.Spec.LinkCRGeneration,
	).V(int(zapcore.WarnLevel))

	linkAdd, err := LinkMetaToNetris(linkMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(linkAdd)
	debugLogger.Info("linkToAdd", "payload", string(js))

	reply, err := r.Cred.Link().Add(linkAdd)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if reply.StatusCode != 200 {
		return ctrl.Result{}, fmt.Errorf(string(reply.Data)), fmt.Errorf(string(reply.Data))
	}

	linkMeta.Spec.ID = fmt.Sprintf("%d-%d", linkMeta.Spec.Local, linkMeta.Spec.Remote)

	debugLogger.Info("Link Created", "id", linkMeta.Spec.ID)

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, linkMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", linkMeta.Spec.ID)
	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LinkMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.LinkMeta{}).
		Complete(r)
}
