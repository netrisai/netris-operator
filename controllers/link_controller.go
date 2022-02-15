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

	"github.com/netrisai/netriswebapi/v2/types/link"
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

// LinkReconciler reconciles a Link object
type LinkReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=links,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=links/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=links/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Link object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *LinkReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	link := &k8sv1alpha1.Link{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	linkCtx, linkCancel := context.WithTimeout(cntxt, contextTimeout)
	defer linkCancel()
	if err := r.Get(linkCtx, req.NamespacedName, link); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	linkMetaNamespaced := req.NamespacedName
	linkMetaNamespaced.Name = string(link.GetUID())
	linkMeta := &k8sv1alpha1.LinkMeta{}
	metaFound := true

	linkMetaCtx, linkMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer linkMetaCancel()
	if err := r.Get(linkMetaCtx, linkMetaNamespaced, linkMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			linkMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if link.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteLink(link, linkMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteLink} %s", err), "")
			return u.patchLinkStatus(link, "Failure", err.Error())
		}
		logger.Info("Link deleted")
		return ctrl.Result{}, nil
	}

	if linkMustUpdateAnnotations(link) {
		debugLogger.Info("Setting default annotations")
		linkUpdateDefaultAnnotations(link)
		linkPatchCtx, linkPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer linkPatchCancel()
		err := r.Patch(linkPatchCtx, link.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Link default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if linkCompareFieldsForNewMeta(link, linkMeta) {
			debugLogger.Info("Generating New Meta")
			linkID := linkMeta.Spec.ID
			newVnetMeta, err := r.LinkToLinkMeta(link)
			if err != nil {
				logger.Error(fmt.Errorf("{LinkToLinkMeta} %s", err), "")
				return u.patchLinkStatus(link, "Failure", err.Error())
			}
			linkMeta.Spec = newVnetMeta.DeepCopy().Spec
			linkMeta.Spec.ID = linkID
			linkMeta.Spec.LinkCRGeneration = link.GetGeneration()

			linkMetaUpdateCtx, linkMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer linkMetaUpdateCancel()
			err = r.Update(linkMetaUpdateCtx, linkMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{linkMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if link.GetFinalizers() == nil {
			link.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			linkPatchCtx, linkPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer linkPatchCancel()
			err := r.Patch(linkPatchCtx, link.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Link Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		linkMeta, err := r.LinkToLinkMeta(link)
		if err != nil {
			logger.Error(fmt.Errorf("{LinkToLinkMeta} %s", err), "")
			return u.patchLinkStatus(link, "Failure", err.Error())
		}

		linkMeta.Spec.LinkCRGeneration = link.GetGeneration()

		linkMetaCreateCtx, linkMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer linkMetaCreateCancel()
		if err := r.Create(linkMetaCreateCtx, linkMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{linkMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *LinkReconciler) deleteLink(linkCR *k8sv1alpha1.Link, linkMeta *k8sv1alpha1.LinkMeta) (ctrl.Result, error) {
	if linkMeta != nil && linkMeta.Spec.ID != "" && !linkMeta.Spec.Reclaim {
		linkDelete := &link.Link{
			Local:  link.LinkIDName{ID: linkMeta.Spec.Local},
			Remote: link.LinkIDName{ID: linkMeta.Spec.Remote},
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
	}
	return r.deleteCRs(linkCR, linkMeta)
}

func (r *LinkReconciler) deleteCRs(link *k8sv1alpha1.Link, linkMeta *k8sv1alpha1.LinkMeta) (ctrl.Result, error) {
	if linkMeta != nil {
		_, err := r.deleteLinkMetaCR(linkMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteLinkCR(link)
}

func (r *LinkReconciler) deleteLinkCR(link *k8sv1alpha1.Link) (ctrl.Result, error) {
	link.ObjectMeta.SetFinalizers(nil)
	link.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, link.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteLinkCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *LinkReconciler) deleteLinkMetaCR(linkMeta *k8sv1alpha1.LinkMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, linkMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteLinkMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LinkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Link{}).
		Complete(r)
}
