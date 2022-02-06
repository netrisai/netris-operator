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

// ControllerReconciler reconciles a Controller object
type ControllerReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=controllers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=controllers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=controllers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Controller object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *ControllerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.NamespacedName)
	debugLogger := logger.V(int(zapcore.WarnLevel))
	controller := &k8sv1alpha1.Controller{}

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	controllerCtx, controllerCancel := context.WithTimeout(cntxt, contextTimeout)
	defer controllerCancel()
	if err := r.Get(controllerCtx, req.NamespacedName, controller); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	controllerMetaNamespaced := req.NamespacedName
	controllerMetaNamespaced.Name = string(controller.GetUID())
	controllerMeta := &k8sv1alpha1.ControllerMeta{}
	metaFound := true

	controllerMetaCtx, controllerMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer controllerMetaCancel()
	if err := r.Get(controllerMetaCtx, controllerMetaNamespaced, controllerMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			metaFound = false
			controllerMeta = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if controller.DeletionTimestamp != nil {
		logger.Info("Go to delete")
		_, err := r.deleteController(controller, controllerMeta)
		if err != nil {
			logger.Error(fmt.Errorf("{deleteController} %s", err), "")
			return u.patchControllerStatus(controller, "Failure", err.Error())
		}
		logger.Info("Controller deleted")
		return ctrl.Result{}, nil
	}

	if controllerMustUpdateAnnotations(controller) {
		debugLogger.Info("Setting default annotations")
		controllerUpdateDefaultAnnotations(controller)
		controllerPatchCtx, controllerPatchCancel := context.WithTimeout(cntxt, contextTimeout)
		defer controllerPatchCancel()
		err := r.Patch(controllerPatchCtx, controller.DeepCopyObject(), client.Merge, &client.PatchOptions{})
		if err != nil {
			logger.Error(fmt.Errorf("{Patch Controller default annotations} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, nil
	}

	if metaFound {
		debugLogger.Info("Meta found")
		if controllerCompareFieldsForNewMeta(controller, controllerMeta) {
			debugLogger.Info("Generating New Meta")
			controllerID := controllerMeta.Spec.ID
			newControllerMeta, err := r.ControllerToControllerMeta(controller)
			if err != nil {
				logger.Error(fmt.Errorf("{ControllerToControllerMeta} %s", err), "")
				return u.patchControllerStatus(controller, "Failure", err.Error())
			}
			controllerMeta.Spec = newControllerMeta.DeepCopy().Spec
			controllerMeta.Spec.ID = controllerID
			controllerMeta.Spec.ControllerCRGeneration = controller.GetGeneration()

			controllerMetaUpdateCtx, controllerMetaUpdateCancel := context.WithTimeout(cntxt, contextTimeout)
			defer controllerMetaUpdateCancel()
			err = r.Update(controllerMetaUpdateCtx, controllerMeta.DeepCopyObject(), &client.UpdateOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{controllerMeta Update} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		}
	} else {
		debugLogger.Info("Meta not found")
		if controller.GetFinalizers() == nil {
			controller.SetFinalizers([]string{"resource.k8s.netris.ai/delete"})

			controllerPatchCtx, controllerPatchCancel := context.WithTimeout(cntxt, contextTimeout)
			defer controllerPatchCancel()
			err := r.Patch(controllerPatchCtx, controller.DeepCopyObject(), client.Merge, &client.PatchOptions{})
			if err != nil {
				logger.Error(fmt.Errorf("{Patch Controller Finalizer} %s", err), "")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			return ctrl.Result{}, nil
		}

		controllerMeta, err := r.ControllerToControllerMeta(controller)
		if err != nil {
			logger.Error(fmt.Errorf("{ControllerToControllerMeta} %s", err), "")
			return u.patchControllerStatus(controller, "Failure", err.Error())
		}

		controllerMeta.Spec.ControllerCRGeneration = controller.GetGeneration()

		controllerMetaCreateCtx, controllerMetaCreateCancel := context.WithTimeout(cntxt, contextTimeout)
		defer controllerMetaCreateCancel()
		if err := r.Create(controllerMetaCreateCtx, controllerMeta.DeepCopyObject(), &client.CreateOptions{}); err != nil {
			logger.Error(fmt.Errorf("{controllerMeta Create} %s", err), "")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *ControllerReconciler) deleteController(controller *k8sv1alpha1.Controller, controllerMeta *k8sv1alpha1.ControllerMeta) (ctrl.Result, error) {
	if controllerMeta != nil && controllerMeta.Spec.ID > 0 && !controllerMeta.Spec.Reclaim {
		reply, err := r.Cred.Inventory().Delete("controller", controllerMeta.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteController} %s", err)
		}
		resp, err := http.ParseAPIResponse(reply.Data)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !resp.IsSuccess && resp.Meta.StatusCode != 400 {
			return ctrl.Result{}, fmt.Errorf("{deleteController} %s", fmt.Errorf(resp.Message))
		}
	}
	return r.deleteCRs(controller, controllerMeta)
}

func (r *ControllerReconciler) deleteCRs(controller *k8sv1alpha1.Controller, controllerMeta *k8sv1alpha1.ControllerMeta) (ctrl.Result, error) {
	if controllerMeta != nil {
		_, err := r.deleteControllerMetaCR(controllerMeta)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("{deleteCRs} %s", err)
		}
	}

	return r.deleteControllerCR(controller)
}

func (r *ControllerReconciler) deleteControllerCR(controller *k8sv1alpha1.Controller) (ctrl.Result, error) {
	controller.ObjectMeta.SetFinalizers(nil)
	controller.SetFinalizers(nil)
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Update(ctx, controller.DeepCopyObject(), &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteControllerCR} %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *ControllerReconciler) deleteControllerMetaCR(controllerMeta *k8sv1alpha1.ControllerMeta) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	if err := r.Delete(ctx, controllerMeta.DeepCopyObject(), &client.DeleteOptions{}); err != nil {
		return ctrl.Result{}, fmt.Errorf("{deleteControllerMetaCR} %s", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Controller{}).
		Complete(r)
}
