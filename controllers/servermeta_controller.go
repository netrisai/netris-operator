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
	"github.com/netrisai/netriswebapi/v2/types/inventory"
)

// ServerMetaReconciler reconciles a ServerMeta object
type ServerMetaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cred     *api.Clientset
	NStorage *netrisstorage.Storage
}

//+kubebuilder:rbac:groups=k8s.netris.ai,resources=servermeta,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=servermeta/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.netris.ai,resources=servermeta/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServerMeta object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *ServerMetaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	debugLogger := r.Log.WithValues("name", req.NamespacedName).V(int(zapcore.WarnLevel))

	serverMeta := &k8sv1alpha1.ServerMeta{}
	serverCR := &k8sv1alpha1.Server{}
	serverMetaCtx, serverMetaCancel := context.WithTimeout(cntxt, contextTimeout)
	defer serverMetaCancel()
	if err := r.Get(serverMetaCtx, req.NamespacedName, serverMeta); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger := r.Log.WithValues("name", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, serverMeta.Spec.ServerName))
	debugLogger = logger.V(int(zapcore.WarnLevel))

	u := uniReconciler{
		Client:      r.Client,
		Logger:      logger,
		DebugLogger: debugLogger,
		Cred:        r.Cred,
		NStorage:    r.NStorage,
	}

	provisionState := "OK"

	serverNN := req.NamespacedName
	serverNN.Name = serverMeta.Spec.ServerName
	serverNNCtx, serverNNCancel := context.WithTimeout(cntxt, contextTimeout)
	defer serverNNCancel()
	if err := r.Get(serverNNCtx, serverNN, serverCR); err != nil {
		if errors.IsNotFound(err) {
			debugLogger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if serverMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if serverMeta.Spec.ID == 0 {
		debugLogger.Info("ID Not found in meta")
		if serverMeta.Spec.Imported {
			logger.Info("Importing server")
			debugLogger.Info("Imported yaml mode. Finding Server by name")
			if server, ok := r.NStorage.HWsStorage.FindServerByName(serverMeta.Spec.ServerName); ok {
				debugLogger.Info("Imported yaml mode. Server found")
				serverMeta.Spec.ID = server.ID
				serverMeta.Spec.MainIP = server.MainIP.Address
				serverMeta.Spec.MgmtIP = server.MgmtIP.Address

				serverMetaPatchCtx, serverMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer serverMetaPatchCancel()
				err := r.Patch(serverMetaPatchCtx, serverMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{patch servermeta.Spec.ID} %s", err), "")
					return u.patchServerStatus(serverCR, "Failure", err.Error())
				}
				debugLogger.Info("Imported yaml mode. ID patched")
				logger.Info("Server imported")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
			logger.Info("Server not found for import")
			debugLogger.Info("Imported yaml mode. Server not found")
		}

		logger.Info("Creating Server")
		if _, err, errMsg := r.createServer(serverMeta); err != nil {
			logger.Error(fmt.Errorf("{createServer} %s", err), "")
			return u.patchServerStatus(serverCR, "Failure", errMsg.Error())
		}
		logger.Info("Server Created")
	} else {
		if apiServer, ok := r.NStorage.HWsStorage.FindServerByID(serverMeta.Spec.ID); ok {
			debugLogger.Info("Comparing ServerMeta with Netris Server")

			needsPatch := false
			if serverMeta.Spec.MainIP == "" && apiServer.MainIP.Address != "" {
				serverMeta.Spec.MainIP = apiServer.MainIP.Address
				needsPatch = true
			}
			if serverMeta.Spec.MgmtIP == "" && apiServer.MgmtIP.Address != "" {
				serverMeta.Spec.MgmtIP = apiServer.MgmtIP.Address
				needsPatch = true
			}
			if serverMeta.Spec.ASN == 0 && apiServer.Asn > 0 {
				serverMeta.Spec.ASN = apiServer.Asn
				needsPatch = true
			}
			if serverMeta.Spec.ProfileID == 0 && apiServer.Profile.ID > 0 {
				serverMeta.Spec.ProfileID = apiServer.Profile.ID
				needsPatch = true
			}
			if serverMeta.Spec.UUID == "" && apiServer.UUID != "" {
				serverMeta.Spec.UUID = apiServer.UUID
				needsPatch = true
			}
			if serverMeta.Spec.PortCount == 0 && apiServer.PortCount > 0 {
				serverMeta.Spec.PortCount = apiServer.PortCount
				needsPatch = true
			}
			if serverMeta.Spec.CustomData == "" && apiServer.CustomData != "" {
				serverMeta.Spec.CustomData = apiServer.CustomData
				needsPatch = true
			}
			if serverMeta.Spec.SRVRole == "" && apiServer.SRVRole != "" {
				serverMeta.Spec.SRVRole = apiServer.SRVRole
				needsPatch = true
			}

			if needsPatch {
				serverMetaPatchCtx, serverMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer serverMetaPatchCancel()
				err := r.Patch(serverMetaPatchCtx, serverMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{Patch serverMeta populated fields} %s", err), "")
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
				debugLogger.Info("Populated fields patched to serverMeta")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}

			// If API has ProfileID=0, clear it in meta to prevent constant updates
			// But we still need to compare other fields (like tags), so we clear ProfileID first
			// and then continue with comparison
			needsProfileIDClear := false
			if apiServer.Profile.ID == 0 && serverMeta.Spec.ProfileID != 0 {
				debugLogger.Info("API has ProfileID=0 (not supported), will clear ProfileID in meta and Server CR after comparison")
				serverMeta.Spec.ProfileID = 0
				needsProfileIDClear = true
			}

			if ok := compareServerMetaAPIServer(serverMeta, apiServer, u); ok {
				// Comparison passed, but we may still need to clear ProfileID
				if needsProfileIDClear {
					serverMetaPatchCtx, serverMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
					defer serverMetaPatchCancel()
					err := r.Patch(serverMetaPatchCtx, serverMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
					if err != nil {
						logger.Error(fmt.Errorf("{Patch serverMeta ProfileID} %s", err), "")
						return ctrl.Result{RequeueAfter: requeueInterval}, nil
					}
					if serverCR.Spec.Profile != "" {
						serverCR.Spec.Profile = ""
						serverCRPatchCtx, serverCRPatchCancel := context.WithTimeout(cntxt, contextTimeout)
						defer serverCRPatchCancel()
						err := r.Patch(serverCRPatchCtx, serverCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
						if err != nil {
							logger.Error(fmt.Errorf("{Patch Server CR Profile} %s", err), "")
							return ctrl.Result{RequeueAfter: requeueInterval}, nil
						}
					}
					debugLogger.Info("ProfileID cleared in meta and Server CR")
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
				debugLogger.Info("Nothing Changed")
			} else {
				debugLogger.Info("Comparison failed - differences detected (see previous logs for specific field changes)")
				debugLogger.Info("Current API state",
					"apiName", apiServer.Name,
					"apiDescription", apiServer.Description,
					"apiMainIP", apiServer.MainIP.Address,
					"apiMgmtIP", apiServer.MgmtIP.Address,
					"apiASN", apiServer.Asn,
					"apiPortCount", apiServer.PortCount,
					"apiProfileID", apiServer.Profile.ID,
					"apiUUID", apiServer.UUID,
					"apiSRVRole", apiServer.SRVRole,
					"apiCustomData", apiServer.CustomData,
					"apiTags", apiServer.Tags,
				)
				debugLogger.Info("Desired state from serverMeta",
					"metaName", serverMeta.Spec.ServerName,
					"metaDescription", serverMeta.Spec.Description,
					"metaMainIP", serverMeta.Spec.MainIP,
					"metaMgmtIP", serverMeta.Spec.MgmtIP,
					"metaASN", serverMeta.Spec.ASN,
					"metaPortCount", serverMeta.Spec.PortCount,
					"metaProfileID", serverMeta.Spec.ProfileID,
					"metaUUID", serverMeta.Spec.UUID,
					"metaSRVRole", serverMeta.Spec.SRVRole,
					"metaCustomData", serverMeta.Spec.CustomData,
					"metaTags", serverMeta.Spec.Tags,
				)
				logger.Info("Updating Server")
				serverUpdate, err := ServerMetaToNetrisUpdate(serverMeta)
				if err != nil {
					logger.Error(fmt.Errorf("{ServerMetaToNetrisUpdate} %s", err), "")
					return u.patchServerStatus(serverCR, "Failure", err.Error())
				}

				js, _ := json.Marshal(serverUpdate)
				debugLogger.Info("Update payload being sent to Netris API", "payload", string(js))

				_, err, errMsg := updateServer(serverMeta.Spec.ID, serverUpdate, r.Cred)
				if err != nil {
					logger.Error(fmt.Errorf("{updateServer} %s", err), "")
					return u.patchServerStatus(serverCR, "Failure", errMsg.Error())
				}
				logger.Info("Server Updated")
				
				// After update, refresh the storage cache and check if ProfileID was actually updated
				// If API still has ProfileID=0 but we sent ProfileID=1, the API might not support setting it
				// In that case, we should accept API's value and update serverMeta
				if err := r.NStorage.HWsStorage.Download(); err != nil {
					debugLogger.Info("Failed to refresh HWsStorage cache", "error", err)
				} else {
					if updatedServer, ok := r.NStorage.HWsStorage.FindServerByID(serverMeta.Spec.ID); ok {
						if serverMeta.Spec.ProfileID != 0 && updatedServer.Profile.ID == 0 {
							// We tried to set ProfileID but API still has 0 - accept API's value
							debugLogger.Info("ProfileID update not accepted by API, accepting API value (0)", 
								"attemptedValue", serverMeta.Spec.ProfileID, "apiValue", updatedServer.Profile.ID)
							serverMeta.Spec.ProfileID = 0
							serverMetaPatchCtx, serverMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
							defer serverMetaPatchCancel()
							err := r.Patch(serverMetaPatchCtx, serverMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
							if err != nil {
								logger.Error(fmt.Errorf("{Patch serverMeta ProfileID} %s", err), "")
								return ctrl.Result{RequeueAfter: requeueInterval}, nil
							}
							debugLogger.Info("ProfileID cleared in serverMeta to match API")
							
							// Also clear Profile in Server CR to prevent it from regenerating serverMeta with ProfileID=1
							if serverCR.Spec.Profile != "" {
								debugLogger.Info("Clearing Profile in Server CR to match API")
								serverCR.Spec.Profile = ""
								serverCRPatchCtx, serverCRPatchCancel := context.WithTimeout(cntxt, contextTimeout)
								defer serverCRPatchCancel()
								err := r.Patch(serverCRPatchCtx, serverCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
								if err != nil {
									logger.Error(fmt.Errorf("{Patch Server CR Profile} %s", err), "")
									return ctrl.Result{RequeueAfter: requeueInterval}, nil
								}
								debugLogger.Info("Profile cleared in Server CR")
							}
							return ctrl.Result{RequeueAfter: requeueInterval}, nil
						}
					}
				}
			}
			
			// Clear ProfileID if needed (after comparison and update)
			if needsProfileIDClear {
				serverMetaPatchCtx, serverMetaPatchCancel := context.WithTimeout(cntxt, contextTimeout)
				defer serverMetaPatchCancel()
				err := r.Patch(serverMetaPatchCtx, serverMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{})
				if err != nil {
					logger.Error(fmt.Errorf("{Patch serverMeta ProfileID} %s", err), "")
					return ctrl.Result{RequeueAfter: requeueInterval}, nil
				}
				if serverCR.Spec.Profile != "" {
					serverCR.Spec.Profile = ""
					serverCRPatchCtx, serverCRPatchCancel := context.WithTimeout(cntxt, contextTimeout)
					defer serverCRPatchCancel()
					err := r.Patch(serverCRPatchCtx, serverCR.DeepCopyObject(), client.Merge, &client.PatchOptions{})
					if err != nil {
						logger.Error(fmt.Errorf("{Patch Server CR Profile} %s", err), "")
						return ctrl.Result{RequeueAfter: requeueInterval}, nil
					}
				}
				debugLogger.Info("ProfileID cleared in meta and Server CR after update")
				return ctrl.Result{RequeueAfter: requeueInterval}, nil
			}
		} else {
			debugLogger.Info("Server not found in Netris")
			debugLogger.Info("Going to create Server")
			logger.Info("Creating Server")
			if _, err, errMsg := r.createServer(serverMeta); err != nil {
				logger.Error(fmt.Errorf("{createServer} %s", err), "")
				return u.patchServerStatus(serverCR, "Failure", errMsg.Error())
			}
			logger.Info("Server Created")
		}
	}

	// Get the API server object to populate profile name if needed
	var apiServerForUpdate *inventory.HW
	if serverMeta.Spec.ID > 0 {
		if apiSrv, ok := r.NStorage.HWsStorage.FindServerByID(serverMeta.Spec.ID); ok {
			apiServerForUpdate = apiSrv
		}
	}
	if _, err := u.updateServerIfNeccesarry(serverCR, *serverMeta, apiServerForUpdate); err != nil {
		logger.Error(fmt.Errorf("{updateServerIfNeccesarry} %s", err), "")
		return u.patchServerStatus(serverCR, "Failure", err.Error())
	}

	return u.patchServerStatus(serverCR, provisionState, "Success")
}

func (r *ServerMetaReconciler) createServer(serverMeta *k8sv1alpha1.ServerMeta) (ctrl.Result, error, error) {
	debugLogger := r.Log.WithValues(
		"name", fmt.Sprintf("%s/%s", serverMeta.Namespace, serverMeta.Spec.ServerName),
		"serverName", serverMeta.Spec.ServerCRGeneration,
	).V(int(zapcore.WarnLevel))

	serverAdd, err := ServerMetaToNetris(serverMeta)
	if err != nil {
		return ctrl.Result{}, err, err
	}

	js, _ := json.Marshal(serverAdd)
	debugLogger.Info("serverToAdd", "payload", string(js))

	reply, err := r.Cred.Inventory().AddServer(serverAdd)
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

	debugLogger.Info("Server Created", "id", idStruct.ID)

	serverMeta.Spec.ID = idStruct.ID

	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	err = r.Patch(ctx, serverMeta.DeepCopyObject(), client.Merge, &client.PatchOptions{}) // requeue
	if err != nil {
		return ctrl.Result{}, err, err
	}

	debugLogger.Info("ID patched to meta", "id", idStruct.ID)
	return ctrl.Result{}, nil, nil
}

func updateServer(id int, server *inventory.HWServer, cred *api.Clientset) (ctrl.Result, error, error) {
	reply, err := cred.Inventory().UpdateServer(id, server)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("{updateServer} %s", err), err
	}
	resp, err := http.ParseAPIResponse(reply.Data)
	if err != nil {
		return ctrl.Result{}, err, err
	}
	if !resp.IsSuccess {
		return ctrl.Result{}, fmt.Errorf("{updateServer} %s", fmt.Errorf(resp.Message)), fmt.Errorf(resp.Message)
	}

	return ctrl.Result{}, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerMetaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.ServerMeta{}).
		Complete(r)
}

func (u *uniReconciler) updateServerIfNeccesarry(serverCR *k8sv1alpha1.Server, serverMeta k8sv1alpha1.ServerMeta, apiServer *inventory.HW) (ctrl.Result, error) {
	shouldUpdateCR := false
	if serverCR.Spec.MainIP == "" && serverCR.Spec.MainIP != serverMeta.Spec.MainIP {
		serverCR.Spec.MainIP = serverMeta.Spec.MainIP
		shouldUpdateCR = true
	}
	if serverCR.Spec.MgmtIP == "" && serverCR.Spec.MgmtIP != serverMeta.Spec.MgmtIP {
		serverCR.Spec.MgmtIP = serverMeta.Spec.MgmtIP
		shouldUpdateCR = true
	}
	// Populate profile name from API if API has a profile and Server CR doesn't have one set
	if apiServer != nil && apiServer.Profile.ID > 0 && apiServer.Profile.Name != "" {
		if serverCR.Spec.Profile == "" || serverCR.Spec.Profile != apiServer.Profile.Name {
			u.DebugLogger.Info("Populating Profile name from API", "profileName", apiServer.Profile.Name, "currentProfile", serverCR.Spec.Profile)
			serverCR.Spec.Profile = apiServer.Profile.Name
			shouldUpdateCR = true
		}
	}
	if shouldUpdateCR {
		u.DebugLogger.Info("Updating Server CR")
		if _, err := u.patchServer(serverCR); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

