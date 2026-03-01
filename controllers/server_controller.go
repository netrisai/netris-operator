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

// ServerReconciler reconciles a Server object
type ServerReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=servers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=servers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=servers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Server object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *ServerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	server := &k8sv1alpha1.Server{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	serverCtx, serverCancel := context.WithTimeout(cntxt, contextTimeout)
	defer serverCancel()
	if err := r.Get(serverCtx, req.NamespacedName, server); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	serverMetaNamespaced := req.NamespacedName
	serverMetaNamespaced.Name = string(server.GetUID())
	serverMeta := &k8sv1alpha1.ServerMeta{}
	metaFound := true

	serverMetaCtx, serverMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer serverMetaCancel()
	if err := r.Get(serverMetaCtx, serverMetaNamespaced, serverMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			serverMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if server.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteServer(server, serverMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteServer} %s", err), "")
			return u.patchServerStatus(server, "Failure", err.Error())
		}
		logger.Info("Server deleted")
		return ctrl.Result{}, nil
	}

	if serverMustUpdateAnnotations(server) {
		debugLogger.Info("Setting default annotations")
		serverUpdateDefaultAnnotations(server)
		serverPatchCtx, serverPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer serverPatchCancel()
		err := r.Patch(serverPatchCtx, server.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Server default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if serverCompareFieldsForNewMeta(server, serverMeta) {
			debugLogger.Info("Generating New Meta")
			serverID := serverMeta.Spec.ID
			newServerMeta, err := r.ServerToServerMeta(server)
			if err != nil {
				logger.Error(fmt.Errorf("{ServerToServerMeta} %s", err), "")
				return u.patchServerStatus(server, "Failure", err.Error())
			}
			serverMeta.Spec = newServerMeta.DeepCopy().Spec
			serverMeta.Spec.ID = serverID
			serverMeta.Spec.ServerCRGeneration = server.GetGeneration()

			serverMetaUpdateCtx, serverMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer serverMetaUpdateCancel()
			err = r.Update(serverMetaUpdateCtx, serverMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{serverMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if server.GetFinalizers() == nil {
			server.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			serverPatchCtx, serverPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer serverPatchCancel()
			err := r.Patch(serverPatchCtx, server.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Server Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		serverMeta, err := r.ServerToServerMeta(server)
		if err != nil {
			logger.Error(fmt.Errorf("{ServerToServerMeta} %s", err), "")
			return u.patchServerStatus(server, "Failure", err.Error())
		}

		serverMeta.Spec.ServerCRGeneration = server.GetGeneration()

		serverMetaCreateCtx, serverMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer serverMetaCreateCancel()
		if err := r.Create(serverMetaCreateCtx, serverMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{serverMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *ServerReconciler) deleteServer(server *k8sv1alpha1.Server, serverMeta *k8sv1alpha1.ServerMeta) (ctrl.Result, error) {
	if serverMeta != nil && serverMeta.Spec.ID > 0 && !serverMeta.Spec.Reclaim {
		reply, err := r.Cred.Inventory().Delete("server", serverMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteServer} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess && resp.Meta.StatusCode != 404 {
			return ctrl.Result{}, fmt.Errorf("{deleteServer} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(server, serverMeta)
}

func (r *ServerReconciler) deleteCRs(server *k8sv1alpha1.Server, serverMeta *k8sv1alpha1.ServerMeta) (ctrl.Result, error) {
	if serverMeta != nil {
		_, err := r.deleteServerMetaCR(serverMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteServerCR(server)
}

func (r *ServerReconciler) deleteServerCR(server *k8sv1alpha1.Server) (ctrl.Result, error) {
	server.ObjectMeta.SetFinalizers(nil)
	server.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, server.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteServerCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *ServerReconciler) deleteServerMetaCR(serverMeta *k8sv1alpha1.ServerMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, serverMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteServerMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Server{}).
		Complete(r)
}

