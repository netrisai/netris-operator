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

package calicowatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/netrisai/netris-operator/api/v1alpha1"
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/calicowatcher/calico"
	"github.com/netrisai/netris-operator/netrisstorage"
	api "github.com/netrisai/netrisapi"
	"github.com/r3labs/diff/v2"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Watcher struct {
	Options    Options
	NStorage   *netrisstorage.Storage
	MGR        manager.Manager
	restClient *rest.Config
	client     client.Client
	clientset  *kubernetes.Clientset
}

type Options struct {
	LogLevel string
}

func NewWatcher(nStorage *netrisstorage.Storage, mgr manager.Manager, options Options) (*Watcher, error) {
	if nStorage == nil {
		return nil, fmt.Errorf("Please provide NStorage")
	}
	watcher := &Watcher{
		NStorage: nStorage,
		MGR:      mgr,
		Options:  options,
	}
	return watcher, nil
}

var (
	logger      logr.Logger
	debugLogger logr.InfoLogger
)

func (w *Watcher) getRestConfig() *rest.Config {
	return ctrl.GetConfigOrDie()
}

func (w *Watcher) start() {
	w.restClient = w.getRestConfig()
	w.client = w.MGR.GetClient()
	clientset, err := kubernetes.NewForConfig(w.restClient)
	if err != nil {
		logger.Error(err, "")
		return
	}
	w.clientset = clientset
	// recorder, w, _ := eventRecorder(clientset)
	// defer w.Stop()
	err = w.mainProcessing()
	if err != nil {
		logger.Error(err, "")
	}
}

func (w *Watcher) Start() {
	if w.Options.LogLevel == "debug" {
		logger = zap.New(zap.Level(zapcore.DebugLevel), zap.UseDevMode(false))
	} else {
		logger = zap.New(zap.UseDevMode(false), zap.StacktraceLevel(zapcore.DPanicLevel))
	}

	logger = ctrl.Log.WithName("CalicoWatcher")
	debugLogger = logger.V(int(zapcore.WarnLevel))

	ticker := time.NewTicker(10 * time.Second)
	w.start()
	for {
		<-ticker.C
		w.start()
	}
}

func (w *Watcher) mainProcessing() error {
	bgpConfs, err := calico.GetBGPConfiguration(w.restClient)
	if err != nil {
		logger.Error(err, "")
	}
	if !w.checkBGPConfigurations(bgpConfs) {
		return fmt.Errorf("Netris annotation is not present")
	}

	ipPools, err := calico.GetIPPool(w.restClient)
	if err != nil {
		logger.Error(err, "")
	}

	if len(ipPools) == 0 && ipPools[0] != nil {
		return fmt.Errorf("IPPool is missing")
	}

	var (
		blockSize    = ipPools[0].Spec.BlockSize
		clusterCIDR  = ipPools[0].Spec.CIDR
		serviceCIDRs = bgpConfs[0].Spec.ServiceClusterIPs
	)

	nodes, err := w.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(nodes.Items) == 0 {
		return fmt.Errorf("Nodes are missing")
	}

	siteName := ""
	siteID := 0
	subnet := ""
	vnet := &api.APIVNet{}

	asnStart := 4200070000
	asnEnd := 4200079000

	nodesMap := make(map[string]*nodeIP)
	asnMap := make(map[string]bool)

	for _, node := range nodes.Items {
		anns := node.GetAnnotations()
		if _, ok := anns["projectcalico.org/ASNumber"]; ok {
			asnMap[anns["projectcalico.org/ASNumber"]] = true
		}
	}

	for _, node := range nodes.Items {
		anns := node.GetAnnotations()
		if _, ok := anns["projectcalico.org/ASNumber"]; !ok {
			for i := asnStart; i < asnEnd; i++ {
				asn := strconv.Itoa(i)
				if !asnMap[asn] {
					anns["projectcalico.org/ASNumber"] = asn
					node.SetAnnotations(anns)
					_, err := w.clientset.CoreV1().Nodes().Update(context.Background(), node.DeepCopy(), metav1.UpdateOptions{})
					if err != nil {
						return err
					}
					asnMap[asn] = true
				}
			}
		} else {
			asnMap[anns["projectcalico.org/ASNumber"]] = true
		}
	}

	for _, node := range nodes.Items {
		anns := node.GetAnnotations()

		if _, ok := anns["projectcalico.org/IPv4Address"]; !ok {
			continue
		}
		if _, ok := anns["projectcalico.org/IPv4IPIPTunnelAddr"]; !ok {
			continue
		}

		asn := ""

		if _, ok := anns["projectcalico.org/ASNumber"]; !ok {
			return fmt.Errorf("Couldn't get as number for node %s", node.Name)
		} else {
			asn = anns["projectcalico.org/ASNumber"]
		}

		tmpNode := &nodeIP{
			IP:   anns["projectcalico.org/IPv4Address"],
			IPIP: anns["projectcalico.org/IPv4IPIPTunnelAddr"],
			ASN:  asn,
		}

		ip := strings.Split(anns["projectcalico.org/IPv4Address"], "/")[0]
		if net.ParseIP(ip) == nil {
			fmt.Println("Invalid IP:", anns["projectcalico.org/IPv4Address"])
			continue
		}

		if siteName == "" {
			id, gateway, err := w.findSiteByIP(ip)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if site, ok := w.NStorage.SitesStorage.FindByID(id); ok {
				siteName = site.Name
				siteID = site.ID
			}
			subnet = gateway
			if vn, ok := w.NStorage.VNetStorage.FindByGateway(gateway); ok {
				vnet = vn
			}
		}

		nodesMap[node.Name] = tmpNode
	}

	if siteName == "" {
		return fmt.Errorf("Couldn't find site")
	}

	if vnet == nil {
		return fmt.Errorf("Couldn't find vnet")
	}

	switchName := ""
	if spine := w.NStorage.HWsStorage.FindSpineBySite(siteID); spine != nil {
		switchName = spine.SwitchName
	} else {
		return fmt.Errorf("Couldn't find spine swtich for site %s", siteName)
	}

	vnetGW := ""
	for _, gw := range vnet.Gateways {
		gateway := fmt.Sprintf("%s/%d", gw.Gateway, gw.GwLength)
		_, gwNet, _ := net.ParseCIDR(gateway)
		if gwNet.String() == subnet {
			vnetGW = gateway
		}
	}

	generatedBGPs := []*k8sv1alpha1.BGP{}
	nameReg, _ := regexp.Compile("[^a-z0-9.]+")
	for name, node := range nodesMap {
		asn, err := strconv.Atoi(node.ASN)
		if err != nil {
			return err
		}
		PrefixListInboundList := []string{fmt.Sprintf("permit %s le %d", clusterCIDR, blockSize)}
		for _, cidr := range serviceCIDRs {
			PrefixListInboundList = append(PrefixListInboundList, fmt.Sprintf("permit %s le %d", cidr.CIDR, 32))
		}

		name := fmt.Sprintf("%s-%s", name, node.IP)

		bgp := &k8sv1alpha1.BGP{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nameReg.ReplaceAllString(name, "-"),
				Namespace: "default",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "BGP",
				APIVersion: "k8s.netris.ai/v1alpha1",
			},
			Spec: k8sv1alpha1.BGPSpec{
				Site:       siteName,
				NeighborAS: asn,
				TerminateOnSwitch: k8sv1alpha1.BGPTerminateOnSwitch{
					Enabled:    true,
					SwitchName: switchName,
				},
				Transport: v1alpha1.BGPTransport{
					Type: "vnet",
					Name: vnet.Name,
				},
				LocalIP:           vnetGW,
				RemoteIP:          node.IP,
				PrefixListInbound: PrefixListInboundList,
				PrefixListOutbound: []string{
					"permit 0.0.0.0/0",
					fmt.Sprintf("deny %s/%d", node.IPIP, blockSize),
					fmt.Sprintf("permit %s le %d", clusterCIDR, blockSize),
				},
			},
		}
		anns := make(map[string]string)
		anns["k8s.netris.ai/calicowatcher"] = "true"
		bgp.SetAnnotations(anns)
		generatedBGPs = append(generatedBGPs, bgp)
	}

	bgps, err := w.getBGPs()
	if err != nil {
		return err
	}

	bgpList := []*k8sv1alpha1.BGP{}
	for _, bgp := range bgps.Items {
		if ann, ok := bgp.GetAnnotations()["k8s.netris.ai/calicowatcher"]; ok && ann == "true" {
			bgpList = append(bgpList, bgp.DeepCopy())
		}
	}

	bgpsForCreate, bgpsForDelete, bgpsForUpdate := compareBGPs(bgpList, generatedBGPs)

	js, _ := json.Marshal(bgpsForCreate)
	debugLogger.Info("BGPs for create", "List", string(js))
	js, _ = json.Marshal(bgpsForDelete)
	debugLogger.Info("BGPs for update", "List", string(js))
	js, _ = json.Marshal(bgpsForUpdate)
	debugLogger.Info("BGPs for delete", "List", string(js))

	var errors []error
	errors = append(errors, w.deleteBGPs(bgpsForDelete)...)
	errors = append(errors, w.updateBGPs(bgpsForUpdate)...)
	errors = append(errors, w.createBGPs(bgpsForCreate)...)
	if len(errors) > 0 {
		fmt.Println(errors)
	}
	return nil
}

func (w *Watcher) createBGPs(BGPs []*k8sv1alpha1.BGP) []error {
	var errors []error
	for _, bgp := range BGPs {
		if err := w.createBGP(bgp); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (w *Watcher) createBGP(bgp *k8sv1alpha1.BGP) error {
	return w.client.Create(context.Background(), bgp.DeepCopyObject(), &client.CreateOptions{})
}

func (w *Watcher) updateBGPs(BGPs []*k8sv1alpha1.BGP) []error {
	var errors []error
	for _, bgp := range BGPs {
		if err := w.updateBGP(bgp); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (w *Watcher) updateBGP(bgp *k8sv1alpha1.BGP) error {
	return w.client.Update(context.Background(), bgp.DeepCopyObject(), &client.UpdateOptions{})
}

func (w *Watcher) deleteBGPs(BGPs []*k8sv1alpha1.BGP) []error {
	var errors []error
	for _, bgp := range BGPs {
		if err := w.deleteBGP(bgp); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (w *Watcher) deleteBGP(bgp *k8sv1alpha1.BGP) error {
	return w.client.Delete(context.Background(), bgp.DeepCopyObject(), &client.DeleteAllOfOptions{})
}

func compareBGPs(BGPs []*k8sv1alpha1.BGP, generatedBGPs []*k8sv1alpha1.BGP) ([]*k8sv1alpha1.BGP, []*k8sv1alpha1.BGP, []*k8sv1alpha1.BGP) {
	genBGPsMap := make(map[string]*k8sv1alpha1.BGP)
	BGPsMap := make(map[string]*k8sv1alpha1.BGP)

	bgpsForCreate := []*k8sv1alpha1.BGP{}
	bgpsForDelete := []*k8sv1alpha1.BGP{}
	bgpsForUpdate := []*k8sv1alpha1.BGP{}

	for _, bgp := range generatedBGPs {
		genBGPsMap[bgp.Name] = bgp
	}

	for _, bgp := range BGPs {
		BGPsMap[bgp.Name] = bgp
	}

	for _, genBGP := range generatedBGPs {
		if bgp, ok := BGPsMap[genBGP.Name]; !ok {
			bgpsForCreate = append(bgpsForCreate, genBGP)
			// Create
		} else {
			changelog, _ := diff.Diff(bgp.Spec, genBGP.Spec)
			if len(changelog) > 0 {
				bgp.Spec = genBGP.Spec
				bgpsForUpdate = append(bgpsForUpdate, bgp)
			}
			// Update
		}
	}

	for _, bgp := range BGPs {
		if _, ok := genBGPsMap[bgp.Name]; !ok {
			bgpsForDelete = append(bgpsForDelete, bgp)
			// Delete
		}
	}

	return bgpsForCreate, bgpsForDelete, bgpsForUpdate
}

func (w *Watcher) getBGPs() (*k8sv1alpha1.BGPList, error) {
	bgps := &k8sv1alpha1.BGPList{}
	err := w.client.List(context.Background(), bgps, &client.ListOptions{})
	if err != nil {
		return nil, err
	}
	return bgps, nil
}

type nodeIP struct {
	IP   string
	IPIP string
	ASN  string
}

func (w *Watcher) checkBGPConfigurations(configurations []*calico.BGPConfiguration) bool {
	for _, conf := range configurations {
		for name, val := range conf.Metadata.GetAnnotations() {
			if name == "manage.k8s.netris.ai/calico" && val == "true" {
				return true
			}
		}
	}
	return false
}
