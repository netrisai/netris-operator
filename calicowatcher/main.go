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
	v1 "k8s.io/api/core/v1"
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
	data       data
}

type data struct {
	deleteMode    bool
	generatedBGPs []*k8sv1alpha1.BGP
	bgpList       []*k8sv1alpha1.BGP
	bgpConfs      []*calico.BGPConfiguration
	site          *api.APISite
	vnetGW        string
	blockSize     int
	clusterCIDR   string
	serviceCIDRs  []string
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
	w.data = data{}
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
	var err error
	if w.data.bgpConfs, err = calico.GetBGPConfiguration(w.restClient); err != nil {
		return err
	}
	if !w.checkBGPConfigurations() {
		w.data.deleteMode = true
	}

	if err := w.getIPInfo(); err != nil {
		return err
	}

	if !w.data.deleteMode {
		if err = w.generateBGPs(); err != nil {
			return err
		}
	}

	bgps, err := w.getBGPs()
	if err != nil {
		return err
	}

	for _, bgp := range bgps.Items {
		if ann, ok := bgp.GetAnnotations()["k8s.netris.ai/calicowatcher"]; ok && ann == "true" {
			w.data.bgpList = append(w.data.bgpList, bgp.DeepCopy())
		}
	}

	bgpsForCreate, bgpsForDelete, bgpsForUpdate := w.compareBGPs()

	js, _ := json.Marshal(bgpsForCreate)
	debugLogger.Info("BGPs for create", "List", string(js))
	js, _ = json.Marshal(bgpsForDelete)
	debugLogger.Info("BGPs for delete", "List", string(js))
	js, _ = json.Marshal(bgpsForUpdate)
	debugLogger.Info("BGPs for update", "List", string(js))

	var errors []error
	errors = append(errors, w.deleteBGPs(bgpsForDelete)...)
	errors = append(errors, w.updateBGPs(bgpsForUpdate)...)
	errors = append(errors, w.createBGPs(bgpsForCreate)...)
	if len(errors) > 0 {
		fmt.Println(errors)
	}

	netrisPeer, err := calico.GetBGPPeer("netris-controller", w.restClient)
	if err != nil {
		return err
	}

	peer := calico.GenerateBGPPeer("netris-controller", "", w.data.vnetGW, w.data.site.ASN)

	if netrisPeer == nil {
		if !w.data.deleteMode {
			if err := calico.CreateBGPPeer(peer, w.restClient); err != nil {
				return err
			}
		}
	} else {
		if w.data.deleteMode {
			if err := calico.DeleteBGPPeer(netrisPeer, w.restClient); err != nil {
				return err
			}
		} else {
			changelog, _ := diff.Diff(netrisPeer.Spec, peer.Spec)
			if len(changelog) > 0 {
				netrisPeer.Spec = peer.Spec
				if err := calico.UpdateBGPPeer(netrisPeer, w.restClient); err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func (w *Watcher) generateBGPs() error {
	generatedBGPs := []*k8sv1alpha1.BGP{}
	nodes, err := w.getNodes()
	if err != nil {
		return err
	}

	err = w.fillNodesASNs(nodes.Items)
	if err != nil {
		return err
	}

	nodesMap, site, vnetName, vnetGW, switchName, err := w.nodesProcessing(nodes.Items)
	if err != nil {
		return err
	}

	w.data.site = site
	w.data.vnetGW = vnetGW

	nameReg, _ := regexp.Compile("[^a-z0-9.]+")
	for name, node := range nodesMap {
		asn, err := strconv.Atoi(node.ASN)
		if err != nil {
			return err
		}
		PrefixListInboundList := []string{fmt.Sprintf("permit %s le %d", w.data.clusterCIDR, w.data.blockSize)}
		for _, cidr := range w.data.serviceCIDRs {
			PrefixListInboundList = append(PrefixListInboundList, fmt.Sprintf("permit %s le %d", cidr, 32))
		}

		name := fmt.Sprintf("%s-%s", name, strings.Split(node.IP, "/")[0])

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
				Site:       site.Name,
				NeighborAS: asn,
				TerminateOnSwitch: k8sv1alpha1.BGPTerminateOnSwitch{
					Enabled:    true,
					SwitchName: switchName,
				},
				Transport: v1alpha1.BGPTransport{
					Type: "vnet",
					Name: vnetName,
				},
				LocalIP:           vnetGW,
				RemoteIP:          node.IP,
				PrefixListInbound: PrefixListInboundList,
				PrefixListOutbound: []string{
					"permit 0.0.0.0/0",
					fmt.Sprintf("deny %s/%d", node.IPIP, w.data.blockSize),
					fmt.Sprintf("permit %s le %d", w.data.clusterCIDR, w.data.blockSize),
				},
			},
		}
		anns := make(map[string]string)
		anns["k8s.netris.ai/calicowatcher"] = "true"
		bgp.SetAnnotations(anns)
		generatedBGPs = append(generatedBGPs, bgp)
	}
	w.data.generatedBGPs = generatedBGPs
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

func (w *Watcher) compareBGPs() ([]*k8sv1alpha1.BGP, []*k8sv1alpha1.BGP, []*k8sv1alpha1.BGP) {
	genBGPsMap := make(map[string]*k8sv1alpha1.BGP)
	BGPsMap := make(map[string]*k8sv1alpha1.BGP)

	bgpsForCreate := []*k8sv1alpha1.BGP{}
	bgpsForDelete := []*k8sv1alpha1.BGP{}
	bgpsForUpdate := []*k8sv1alpha1.BGP{}

	for _, bgp := range w.data.generatedBGPs {
		genBGPsMap[bgp.Name] = bgp
	}

	for _, bgp := range w.data.bgpList {
		BGPsMap[bgp.Name] = bgp
	}

	for _, genBGP := range w.data.generatedBGPs {
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

	for _, bgp := range w.data.bgpList {
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

func (w *Watcher) checkBGPConfigurations() bool {
	for _, conf := range w.data.bgpConfs {
		for name, val := range conf.Metadata.GetAnnotations() {
			if name == "manage.k8s.netris.ai/calico" && val == "true" {
				return true
			}
		}
	}
	return false
}

func (w *Watcher) getNodes() (*v1.NodeList, error) {
	nodes, err := w.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(nodes.Items) == 0 {
		return nil, fmt.Errorf("Nodes are missing")
	}
	return nodes, nil
}

func (w *Watcher) getIPPools() ([]*calico.IPPool, error) {
	ipPools, err := calico.GetIPPool(w.restClient)
	if err != nil {
		return nil, err
	}

	if len(ipPools) == 0 && ipPools[0] != nil {
		return nil, fmt.Errorf("IPPool is missing")
	}
	return ipPools, nil
}

func (w *Watcher) getIPInfo() error {
	var (
		blockSize    int
		clusterCIDR  string
		serviceCIDRs []string
	)

	ipPools, err := w.getIPPools()
	if err != nil {
		return err
	}

	blockSize = ipPools[0].Spec.BlockSize
	clusterCIDR = ipPools[0].Spec.CIDR
	for _, c := range w.data.bgpConfs[0].Spec.ServiceClusterIPs {
		serviceCIDRs = append(serviceCIDRs, c.CIDR)
	}
	w.data.blockSize = blockSize
	w.data.clusterCIDR = clusterCIDR
	w.data.serviceCIDRs = serviceCIDRs
	return nil
}

func (w *Watcher) fillNodesASNs(nodes []v1.Node) error {
	asnStart := 4200070000
	asnEnd := 4200079000
	asnMap := make(map[string]bool)

	for _, node := range nodes {
		anns := node.GetAnnotations()
		if _, ok := anns["projectcalico.org/ASNumber"]; ok {
			asnMap[anns["projectcalico.org/ASNumber"]] = true
		}
	}

	for _, node := range nodes {
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
	return nil
}

func (w *Watcher) nodesProcessing(nodes []v1.Node) (map[string]*nodeIP, *api.APISite, string, string, string, error) {
	var (
		siteName   string
		site       *api.APISite
		vnetName   string
		vnetGW     string
		switchName string
	)

	siteID := 0
	subnet := ""
	vnet := &api.APIVNet{}

	nodesMap := make(map[string]*nodeIP)

	for _, node := range nodes {
		anns := node.GetAnnotations()

		if _, ok := anns["projectcalico.org/IPv4Address"]; !ok {
			continue
		}
		if _, ok := anns["projectcalico.org/IPv4IPIPTunnelAddr"]; !ok {
			continue
		}

		asn := ""

		if _, ok := anns["projectcalico.org/ASNumber"]; !ok {
			return nodesMap, site, vnetName, vnetGW, switchName, fmt.Errorf("Couldn't get as number for node %s", node.Name)
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
			var ok bool
			if site, ok = w.NStorage.SitesStorage.FindByID(id); ok {
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
		return nodesMap, site, vnetName, vnetGW, switchName, fmt.Errorf("Couldn't find site")
	}

	if vnet == nil {
		return nodesMap, site, vnetName, vnetGW, switchName, fmt.Errorf("Couldn't find vnet")
	}

	if spine := w.NStorage.HWsStorage.FindSpineBySite(siteID); spine != nil {
		switchName = spine.SwitchName
	} else {
		return nodesMap, site, vnetName, vnetGW, switchName, fmt.Errorf("Couldn't find spine swtich for site %s", siteName)
	}

	vnetName = vnet.Name
	for _, gw := range vnet.Gateways {
		gateway := fmt.Sprintf("%s/%d", gw.Gateway, gw.GwLength)
		_, gwNet, _ := net.ParseCIDR(gateway)
		if gwNet.String() == subnet {
			vnetGW = gateway
		}
	}

	return nodesMap, site, vnetName, vnetGW, switchName, nil
}
