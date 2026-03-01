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
	"fmt"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netriswebapi/v2/types/inventory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// normalizeTags converts nil tags to empty slice to ensure consistent comparison
func normalizeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

// ServerToServerMeta converts the Server resource to ServerMeta type and used for add the Server for Netris API.
func (r *ServerReconciler) ServerToServerMeta(server *k8sv1alpha1.Server) (*k8sv1alpha1.ServerMeta, error) {
	var (
		imported = false
		reclaim  = false
	)

	if i, ok := server.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := server.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}

	siteID := 0
	if site, ok := r.NStorage.SitesStorage.FindByName(server.Spec.Site); ok {
		siteID = site.ID
	} else {
		return nil, fmt.Errorf("invalid site '%s'", server.Spec.Site)
	}

	tenantID := 0
	if tenant, ok := r.NStorage.TenantsStorage.FindByName(server.Spec.Tenant); ok {
		tenantID = tenant.ID
	} else {
		return nil, fmt.Errorf("invalid tenant '%s'", server.Spec.Tenant)
	}

	profileID := 0
	profiles, err := r.Cred.InventoryProfile().Get()
	if err != nil {
		return nil, err
	}

	for _, p := range profiles {
		if p.Name == server.Spec.Profile {
			profileID = p.ID
		}
	}

	if profileID == 0 && server.Spec.Profile != "" {
		return nil, fmt.Errorf("invalid profile '%s'", server.Spec.Profile)
	}

	serverMeta := &k8sv1alpha1.ServerMeta{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(server.GetUID()),
			Namespace: server.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{},
		Spec: k8sv1alpha1.ServerMetaSpec{
			Imported:     imported,
			Reclaim:      reclaim,
			ServerName:   server.Name,
			Description:  server.Spec.Description,
			TenantID:     tenantID,
			SiteID:       siteID,
			ProfileID:    profileID,
		MainIP:       server.Spec.MainIP,
		MgmtIP:       server.Spec.MgmtIP,
		UUID:         server.Spec.UUID,
		ASN:          server.Spec.ASN,
		PortCount:    server.Spec.PortCount,
		CustomData:   server.Spec.CustomData,
		Tags:         normalizeTags(server.Spec.Tags),
		SRVRole:      server.Spec.SRVRole,
		},
	}

	return serverMeta, nil
}

func serverCompareFieldsForNewMeta(server *k8sv1alpha1.Server, serverMeta *k8sv1alpha1.ServerMeta) bool {
	imported := false
	reclaim := false
	if i, ok := server.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = true
	}
	if i, ok := server.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = true
	}
	return server.GetGeneration() != serverMeta.Spec.ServerCRGeneration || imported != serverMeta.Spec.Imported || reclaim != serverMeta.Spec.Reclaim
}

func serverMustUpdateAnnotations(server *k8sv1alpha1.Server) bool {
	update := false
	if i, ok := server.GetAnnotations()["resource.k8s.netris.ai/import"]; !(ok && (i == "true" || i == "false")) {
		update = true
	}
	if i, ok := server.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; !(ok && (i == "retain" || i == "delete")) {
		update = true
	}
	return update
}

func serverUpdateDefaultAnnotations(server *k8sv1alpha1.Server) {
	imported := "false"
	reclaim := "delete"
	if i, ok := server.GetAnnotations()["resource.k8s.netris.ai/import"]; ok && i == "true" {
		imported = "true"
	}
	if i, ok := server.GetAnnotations()["resource.k8s.netris.ai/reclaimPolicy"]; ok && i == "retain" {
		reclaim = "retain"
	}
	annotations := server.GetAnnotations()
	annotations["resource.k8s.netris.ai/import"] = imported
	annotations["resource.k8s.netris.ai/reclaimPolicy"] = reclaim
	server.SetAnnotations(annotations)
}

// ServerMetaToNetris converts the k8s Server resource to Netris type and used for add the Server for Netris API.
func ServerMetaToNetris(serverMeta *k8sv1alpha1.ServerMeta) (*inventory.HWServer, error) {
	mainIP := serverMeta.Spec.MainIP
	if serverMeta.Spec.MainIP == "" {
		mainIP = "auto"
	}

	mgmtIP := serverMeta.Spec.MgmtIP
	if serverMeta.Spec.MgmtIP == "" {
		mgmtIP = "auto"
	}

	var asn interface{} = serverMeta.Spec.ASN
	if serverMeta.Spec.ASN == 0 {
		asn = "auto"
	}

	tags := normalizeTags(serverMeta.Spec.Tags)

	serverAdd := &inventory.HWServer{
		Name:        serverMeta.Spec.ServerName,
		Description: serverMeta.Spec.Description,
		Tenant:      inventory.IDName{ID: serverMeta.Spec.TenantID},
		Site:        inventory.IDName{ID: serverMeta.Spec.SiteID},
		Profile:     inventory.IDName{ID: serverMeta.Spec.ProfileID},
		MainAddress: mainIP,
		MgmtAddress: mgmtIP,
		UUID:        serverMeta.Spec.UUID,
		Asn:         asn,
		PortCount:   serverMeta.Spec.PortCount,
		CustomData:  serverMeta.Spec.CustomData,
		Tags:        tags,
		SRVRole:     serverMeta.Spec.SRVRole,
		Links:       []inventory.HWLink{},
	}

	return serverAdd, nil
}

// ServerMetaToNetrisUpdate converts the k8s Server resource to Netris type and used for update the Server for Netris API.
func ServerMetaToNetrisUpdate(serverMeta *k8sv1alpha1.ServerMeta) (*inventory.HWServer, error) {
	mainIP := serverMeta.Spec.MainIP
	if serverMeta.Spec.MainIP == "" {
		mainIP = "auto"
	}

	mgmtIP := serverMeta.Spec.MgmtIP
	if serverMeta.Spec.MgmtIP == "" {
		mgmtIP = "auto"
	}

	var asn interface{} = serverMeta.Spec.ASN
	if serverMeta.Spec.ASN == 0 {
		asn = "auto"
	}

	tags := normalizeTags(serverMeta.Spec.Tags)

	serverUpdate := &inventory.HWServer{
		Name:        serverMeta.Spec.ServerName,
		Description: serverMeta.Spec.Description,
		Tenant:      inventory.IDName{ID: serverMeta.Spec.TenantID},
		Site:        inventory.IDName{ID: serverMeta.Spec.SiteID},
		Profile:     inventory.IDName{ID: serverMeta.Spec.ProfileID},
		MainAddress: mainIP,
		MgmtAddress: mgmtIP,
		UUID:        serverMeta.Spec.UUID,
		Asn:         asn,
		PortCount:   serverMeta.Spec.PortCount,
		CustomData:  serverMeta.Spec.CustomData,
		Tags:        tags,
		SRVRole:     serverMeta.Spec.SRVRole,
		Links:       []inventory.HWLink{},
	}

	return serverUpdate, nil
}

func compareServerMetaAPIServer(serverMeta *k8sv1alpha1.ServerMeta, apiServer *inventory.HW, u uniReconciler) bool {
	if apiServer.Name != serverMeta.Spec.ServerName {
		u.DebugLogger.Info("Name changed", "netrisValue", apiServer.Name, "k8sValue", serverMeta.Spec.ServerName)
		return false
	}

	if apiServer.Description != serverMeta.Spec.Description {
		u.DebugLogger.Info("Description changed", "netrisValue", apiServer.Description, "k8sValue", serverMeta.Spec.Description)
		return false
	}

	if apiServer.Tenant.ID != serverMeta.Spec.TenantID {
		u.DebugLogger.Info("Tenant changed", "netrisValue", apiServer.Tenant.ID, "k8sValue", serverMeta.Spec.TenantID)
		return false
	}

	if apiServer.Site.ID != serverMeta.Spec.SiteID {
		u.DebugLogger.Info("Site changed", "netrisValue", apiServer.Site.ID, "k8sValue", serverMeta.Spec.SiteID)
		return false
	}

	// Compare ProfileID: only compare if API actually has a ProfileID set (not 0)
	// If API has ProfileID=0, it means the API doesn't support/accept ProfileID for this server
	// In that case, we should ignore ProfileID in meta and not try to update it
	if apiServer.Profile.ID != 0 {
		// API has ProfileID set, so compare it with meta
		if apiServer.Profile.ID != serverMeta.Spec.ProfileID {
			u.DebugLogger.Info("Profile changed", "netrisValue", apiServer.Profile.ID, "k8sValue", serverMeta.Spec.ProfileID)
			return false
		}
	} else {
		// API has ProfileID=0 - API doesn't support ProfileID for this server
		// Clear ProfileID in meta to match API and prevent constant updates
		if serverMeta.Spec.ProfileID != 0 {
			u.DebugLogger.Info("API has ProfileID=0 (not supported), clearing ProfileID in meta", "metaProfileID", serverMeta.Spec.ProfileID)
			// Note: We'll clear this in the controller after comparison
		}
	}

	if apiServer.MainIP.Address != serverMeta.Spec.MainIP {
		u.DebugLogger.Info("MainIP changed", "netrisValue", apiServer.MainIP.Address, "k8sValue", serverMeta.Spec.MainIP)
		return false
	}

	if apiServer.MgmtIP.Address != serverMeta.Spec.MgmtIP {
		u.DebugLogger.Info("MgmtIP changed", "netrisValue", apiServer.MgmtIP.Address, "k8sValue", serverMeta.Spec.MgmtIP)
		return false
	}

	// Only compare UUID if it's set in meta (populated from API if empty)
	if serverMeta.Spec.UUID != "" && apiServer.UUID != serverMeta.Spec.UUID {
		u.DebugLogger.Info("UUID changed", "netrisValue", apiServer.UUID, "k8sValue", serverMeta.Spec.UUID)
		return false
	}

	// Only compare SRVRole if it's set in meta (populated from API if empty)
	if serverMeta.Spec.SRVRole != "" && apiServer.SRVRole != serverMeta.Spec.SRVRole {
		u.DebugLogger.Info("SRVRole changed", "netrisValue", apiServer.SRVRole, "k8sValue", serverMeta.Spec.SRVRole)
		return false
	}

	// Only compare ASN if it's explicitly set (not 0)
	if apiServer.Asn != serverMeta.Spec.ASN && serverMeta.Spec.ASN != 0 {
		u.DebugLogger.Info("ASN changed", "netrisValue", apiServer.Asn, "k8sValue", serverMeta.Spec.ASN)
		return false
	}

	// Only compare PortCount if it's explicitly set (not 0)
	if apiServer.PortCount != serverMeta.Spec.PortCount && serverMeta.Spec.PortCount != 0 {
		u.DebugLogger.Info("PortCount changed", "netrisValue", apiServer.PortCount, "k8sValue", serverMeta.Spec.PortCount)
		return false
	}

	// Only compare CustomData if it's set in meta (populated from API if empty)
	if serverMeta.Spec.CustomData != "" && apiServer.CustomData != serverMeta.Spec.CustomData {
		u.DebugLogger.Info("CustomData changed", "netrisValue", apiServer.CustomData, "k8sValue", serverMeta.Spec.CustomData)
		return false
	}

	// Compare Tags - normalize nil to empty slice
	apiTags := normalizeTags(apiServer.Tags)
	metaTags := normalizeTags(serverMeta.Spec.Tags)
	
	// Compare lengths first
	if len(apiTags) != len(metaTags) {
		u.DebugLogger.Info("Tags length changed", "netrisValue", len(apiTags), "k8sValue", len(metaTags), "apiTags", apiTags, "metaTags", metaTags)
		return false
	}
	
	// If both are empty, they match
	if len(apiTags) == 0 && len(metaTags) == 0 {
		return true
	}
	
	// Compare both directions: all metaTags should be in apiTags AND all apiTags should be in metaTags
	apiTagMap := make(map[string]bool)
	for _, tag := range apiTags {
		apiTagMap[tag] = true
	}
	metaTagMap := make(map[string]bool)
	for _, tag := range metaTags {
		metaTagMap[tag] = true
	}
	
	// Check if all metaTags are in apiTags
	for _, tag := range metaTags {
		if !apiTagMap[tag] {
			u.DebugLogger.Info("Tags changed - meta tag not in API", "tag", tag, "apiTags", apiTags, "metaTags", metaTags)
			return false
		}
	}
	
	// Check if all apiTags are in metaTags
	for _, tag := range apiTags {
		if !metaTagMap[tag] {
			u.DebugLogger.Info("Tags changed - API tag not in meta", "tag", tag, "apiTags", apiTags, "metaTags", metaTags)
			return false
		}
	}

	return true
}

