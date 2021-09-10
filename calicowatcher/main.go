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
	"github.com/netrisai/netris-operator/configloader"
	"github.com/netrisai/netris-operator/netrisstorage"
	"github.com/netrisai/netriswebapi/v1/types/site"
	"github.com/netrisai/netriswebapi/v2/types/vnet"
	"github.com/r3labs/diff/v2"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	requeueInterval = time.Duration(10 * time.Second)
	logger          logr.Logger
	debugLogger     logr.InfoLogger
	cntxt           = context.Background()
	contextTimeout  = requeueInterval
)

type Watcher struct {
	Options    Options
	NStorage   *netrisstorage.Storage
	MGR        manager.Manager
	Calico     *calico.Calico
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

	nodesMap   map[string]*nodeIP
	vnetName   string
	switchName string

	nodes        *v1.NodeList
	site         *site.Site
	vnetGW       string
	vnetGWIP     string
	blockSize    int
	clusterCIDR  string
	serviceCIDRs []string
	asnStart     int
	asnEnd       int
}

type Options struct {
	RequeueInterval int
	LogLevel        string
}

func NewWatcher(nStorage *netrisstorage.Storage, mgr manager.Manager, options Options) (*Watcher, error) {
	if nStorage == nil {
		return nil, fmt.Errorf("Please provide NStorage")
	}

	watcher := &Watcher{
		NStorage: nStorage,
		MGR:      mgr,
		Options:  options,
		Calico:   calico.New(calico.Options{ContextTimeout: options.RequeueInterval}),
	}
	return watcher, nil
}

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
	if len(configloader.Root.CalicoASNRange) > 0 {
		a, b, err := w.validateASNRange(configloader.Root.CalicoASNRange)
		if err != nil {
			logger.Error(err, "")
			return
		}
		w.data.asnStart = a
		w.data.asnEnd = b
	} else {
		w.data.asnStart = 4200070000
		w.data.asnEnd = 4200079999
	}
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

	if w.Options.RequeueInterval > 0 {
		requeueInterval = time.Duration(time.Duration(w.Options.RequeueInterval) * time.Second)
		contextTimeout = requeueInterval
	}

	ticker := time.NewTicker(requeueInterval)
	w.start()
	for {
		<-ticker.C
		w.start()
	}
}

func (w *Watcher) process() error {
	debugLogger.Info("Getting IP information", "deleteMode", w.data.deleteMode)
	if err := w.getIPInfo(); err != nil {
		return err
	}

	debugLogger.Info("Getting Nodes", "deleteMode", w.data.deleteMode)
	if err := w.getNodes(); err != nil {
		return err
	}

	debugLogger.Info("Filling Nodes AS numbers", "deleteMode", w.data.deleteMode)
	if err := w.fillNodesASNs(); err != nil {
		return err
	}

	debugLogger.Info("Nodes Processing", "deleteMode", w.data.deleteMode)
	if err := w.nodesProcessing(); err != nil {
		return err
	}

	debugLogger.Info("Generating BGPs", "deleteMode", w.data.deleteMode)
	if err := w.generateBGPs(); err != nil {
		return err
	}

	debugLogger.Info("Getting BGP list from k8s", "deleteMode", w.data.deleteMode)
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
	debugLogger.Info("BGPs for create", "List", string(js), "deleteMode", w.data.deleteMode)
	js, _ = json.Marshal(bgpsForDelete)
	debugLogger.Info("BGPs for delete", "List", string(js), "deleteMode", w.data.deleteMode)
	js, _ = json.Marshal(bgpsForUpdate)
	debugLogger.Info("BGPs for update", "List", string(js), "deleteMode", w.data.deleteMode)

	var errors []error
	errors = append(errors, w.deleteBGPs(bgpsForDelete)...)
	errors = append(errors, w.updateBGPs(bgpsForUpdate)...)
	errors = append(errors, w.createBGPs(bgpsForCreate)...)
	if len(errors) > 0 {
		fmt.Println(errors)
	}

	debugLogger.Info("Getting netris-controller peer", "deleteMode", w.data.deleteMode)
	netrisPeer, err := w.Calico.GetBGPPeer("netris-controller", w.restClient)
	if err != nil {
		return err
	}

	debugLogger.Info("Generating netris-controller peer", "deleteMode", w.data.deleteMode)
	peer := w.Calico.GenerateBGPPeer("netris-controller", "", w.data.vnetGWIP, w.data.site.ASN)

	if netrisPeer == nil {
		debugLogger.Info("Creating netris-controller peer", "deleteMode", w.data.deleteMode)
		if err := w.Calico.CreateBGPPeer(peer, w.restClient); err != nil {
			return err
		}
		logger.Info("netris-controller peer created", "deleteMode", w.data.deleteMode)
	} else {
		changelog, _ := diff.Diff(netrisPeer.Spec, peer.Spec)
		if len(changelog) > 0 {
			debugLogger.Info("Updating netris-controller peer", "deleteMode", w.data.deleteMode)
			netrisPeer.Spec = peer.Spec
			if err := w.Calico.UpdateBGPPeer(netrisPeer, w.restClient); err != nil {
				return err
			}
			logger.Info("netris-controller peer updated", "deleteMode", w.data.deleteMode)
		}
	}

	bgpActive := true
	for _, bgp := range w.data.bgpList {
		if !((bgp.Status.BGPStatus == "Active" || bgp.Status.BGPStatus == "Established") && bgp.Status.BGPPrefixes > 0) {
			bgpActive = false
			break
		}
	}

	if bgpActive {
		if *w.data.bgpConfs[0].Spec.NodeToNodeMeshEnabled {
			debugLogger.Info("All BGPs are established", "deleteMode", w.data.deleteMode)
			debugLogger.Info("Disabling NodeToNodeMesh in BGP Configuration", "deleteMode", w.data.deleteMode)
			if err := w.updateBGPConfMesh(false); err != nil {
				return err
			}
			logger.Info("NodeToNodeMesh disabled in BGP Configuration", "deleteMode", w.data.deleteMode)
		}
	} else {
		if !*w.data.bgpConfs[0].Spec.NodeToNodeMeshEnabled {
			debugLogger.Info("BGPs are not established", "deleteMode", w.data.deleteMode)
			debugLogger.Info("Enabling NodeToNodeMesh in BGP Configuration", "deleteMode", w.data.deleteMode)
			if err := w.updateBGPConfMesh(true); err != nil {
				return err
			}
			logger.Info("NodeToNodeMesh enabled in BGP Configuration", "deleteMode", w.data.deleteMode)
		}
	}

	return nil
}

func (w *Watcher) deleteNodesProcessing() error {
	debugLogger.Info("Getting Nodes", "deleteMode", w.data.deleteMode)
	if err := w.getNodes(); err != nil {
		return err
	}

	debugLogger.Info("Deleting Nodes ASN annotation", "deleteMode", w.data.deleteMode)
	if err := w.deleteNodesASNs(); err != nil {
		return err
	}
	return nil
}

func (w *Watcher) deleteNodesASNs() error {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	for _, node := range w.data.nodes.Items {
		anns := node.GetAnnotations()
		if asn, ok := anns["projectcalico.org/ASNumber"]; ok {
			as, _ := strconv.Atoi(asn)
			if as >= w.data.asnStart && as <= w.data.asnEnd {
				delete(anns, "projectcalico.org/ASNumber")
				node.SetAnnotations(anns)
				payload := []patchStringValue{{
					Op:    "remove",
					Path:  "/metadata/annotations/projectcalico.org~1ASNumber",
					Value: asn,
				}}
				payloadBytes, _ := json.Marshal(payload)
				_, err := w.clientset.CoreV1().Nodes().Patch(ctx, node.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (w *Watcher) deleteProcess() error {
	if !*w.data.bgpConfs[0].Spec.NodeToNodeMeshEnabled {
		if err := w.updateBGPConfMesh(true); err != nil {
			return err
		}
		logger.Info("NodeToNodeMesh enabled in BGP Configuration", "deleteMode", w.data.deleteMode)
	}

	if err := w.deleteNodesProcessing(); err != nil {
		return err
	}

	w.data.generatedBGPs = []*k8sv1alpha1.BGP{}

	debugLogger.Info("Geting BGPs from k8s", "deleteMode", w.data.deleteMode)
	bgps, err := w.getBGPs()
	if err != nil {
		return err
	}

	for _, bgp := range bgps.Items {
		if ann, ok := bgp.GetAnnotations()["k8s.netris.ai/calicowatcher"]; ok && ann == "true" {
			w.data.bgpList = append(w.data.bgpList, bgp.DeepCopy())
		}
	}

	_, bgpsForDelete, _ := w.compareBGPs()

	js, _ := json.Marshal(bgpsForDelete)
	debugLogger.Info("BGPs for delete", "List", string(js), "deleteMode", w.data.deleteMode)

	var errors []error
	errors = append(errors, w.deleteBGPs(bgpsForDelete)...)
	if len(errors) > 0 {
		fmt.Println(errors)
	}

	debugLogger.Info("Geting netris-controller peer", "deleteMode", w.data.deleteMode)
	netrisPeer, err := w.Calico.GetBGPPeer("netris-controller", w.restClient)
	if err != nil {
		return err
	}

	if netrisPeer != nil {
		debugLogger.Info("Deleting netris-controller peer", "deleteMode", w.data.deleteMode)
		if err := w.Calico.DeleteBGPPeer(netrisPeer, w.restClient); err != nil {
			return err
		}
		logger.Info("netris-controller pee deleted", "deleteMode", w.data.deleteMode)
	}

	return nil
}

func (w *Watcher) mainProcessing() error {
	var err error
	if w.data.bgpConfs, err = w.Calico.GetBGPConfiguration(w.restClient); err != nil {
		return err
	}
	if !w.checkBGPConfigurations() {
		w.data.deleteMode = true
	}

	if w.data.deleteMode {
		debugLogger.Info("manage.k8s.netris.ai/calico is missing in BGP Configuration", "deleteMode", w.data.deleteMode)
		debugLogger.Info("Clearing Netris staff", "deleteMode", w.data.deleteMode)
		return w.deleteProcess()
	} else {
		debugLogger.Info("manage.k8s.netris.ai/calico is present in BGP Configuration", "deleteMode", w.data.deleteMode)
		debugLogger.Info("Creating Netris staff", "deleteMode", w.data.deleteMode)
		return w.process()
	}
}

func (w *Watcher) updateBGPConfMesh(enabled bool) error {
	if len(w.data.bgpConfs) > 0 {
		bgpConf := w.data.bgpConfs[0]
		bgpConf.Spec.NodeToNodeMeshEnabled = &enabled
		return w.Calico.UpdateBGPConfiguration(bgpConf, w.restClient)
	}
	return fmt.Errorf("BGPConfiguration is missing in calico")
}

func (w *Watcher) generateBGPs() error {
	generatedBGPs := []*k8sv1alpha1.BGP{}

	nameReg, _ := regexp.Compile("[^a-z0-9.]+")
	for name, node := range w.data.nodesMap {
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
				Site:       w.data.site.Name,
				NeighborAS: asn,
				TerminateOnSwitch: k8sv1alpha1.BGPTerminateOnSwitch{
					Enabled:    true,
					SwitchName: w.data.switchName,
				},
				Transport: v1alpha1.BGPTransport{
					Type: "vnet",
					Name: w.data.vnetName,
				},
				LocalIP:           w.data.vnetGW,
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
		anns["resource.k8s.netris.ai/import"] = "true"
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
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	return w.client.Create(ctx, bgp.DeepCopyObject(), &client.CreateOptions{})
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
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	return w.client.Update(ctx, bgp.DeepCopyObject(), &client.UpdateOptions{})
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
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	return w.client.Delete(ctx, bgp.DeepCopyObject(), &client.DeleteAllOfOptions{})
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
		} else {
			changelog, _ := diff.Diff(bgp.Spec, genBGP.Spec)
			if len(changelog) > 0 {
				bgp.Spec = genBGP.Spec
				bgpsForUpdate = append(bgpsForUpdate, bgp)
			}
		}
	}

	for _, bgp := range w.data.bgpList {
		if _, ok := genBGPsMap[bgp.Name]; !ok {
			bgpsForDelete = append(bgpsForDelete, bgp)
		}
	}

	return bgpsForCreate, bgpsForDelete, bgpsForUpdate
}

func (w *Watcher) getBGPs() (*k8sv1alpha1.BGPList, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	bgps := &k8sv1alpha1.BGPList{}
	err := w.client.List(ctx, bgps, &client.ListOptions{})
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

func (w *Watcher) getNodes() error {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	nodes, err := w.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(nodes.Items) == 0 {
		return fmt.Errorf("Nodes are missing")
	}
	w.data.nodes = nodes
	return nil
}

func (w *Watcher) getIPPools() ([]*calico.IPPool, error) {
	ipPools, err := w.Calico.GetIPPool(w.restClient)
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

func (w *Watcher) fillNodesASNs() error {
	asnMap := make(map[string]bool)

	for _, node := range w.data.nodes.Items {
		anns := node.GetAnnotations()
		if _, ok := anns["projectcalico.org/ASNumber"]; ok {
			asnMap[anns["projectcalico.org/ASNumber"]] = true
		}
	}

	for _, node := range w.data.nodes.Items {
		anns := node.GetAnnotations()
		if _, ok := anns["projectcalico.org/ASNumber"]; !ok {
			for i := w.data.asnStart; i < w.data.asnEnd; i++ {
				asn := strconv.Itoa(i)
				if !asnMap[asn] {
					anns["projectcalico.org/ASNumber"] = asn
					node.SetAnnotations(anns)
					payload := []patchStringValue{{
						Op:    "replace",
						Path:  "/metadata/annotations/projectcalico.org~1ASNumber",
						Value: asn,
					}}
					payloadBytes, _ := json.Marshal(payload)
					ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
					_, err := w.clientset.CoreV1().Nodes().Patch(ctx, node.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
					if err != nil {
						cancel()
						return err
					}
					asnMap[asn] = true
					cancel()
					break
				}
			}
		} else {
			asnMap[anns["projectcalico.org/ASNumber"]] = true
		}
	}
	return nil
}

func (w *Watcher) nodesProcessing() error {
	var (
		siteName   string
		site       *site.Site
		vnetName   string
		vnetGW     string
		switchName string
		vnetGWIP   string
	)

	siteID := 0
	subnet := ""
	vnet := &vnet.VNet{}

	nodesMap := make(map[string]*nodeIP)

	for _, node := range w.data.nodes.Items {
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
		return fmt.Errorf("Couldn't find site")
	}

	if vnet == nil {
		return fmt.Errorf("Couldn't find vnet")
	}

	if spine := w.NStorage.HWsStorage.FindSpineBySite(siteID); spine != nil {
		switchName = spine.Name
	} else {
		return fmt.Errorf("Couldn't find spine swtich for site %s", siteName)
	}

	vnetName = vnet.Name
	for _, gw := range vnet.Gateways {
		gateway := strings.Split(gw.Prefix, "/")[0]
		_, gwNet, _ := net.ParseCIDR(gateway)
		if gwNet.String() == subnet {
			vnetGW = gw.Prefix
			vnetGWIP = gateway
		}
	}
	w.data.nodesMap = nodesMap
	w.data.site = site
	w.data.vnetName = vnetName
	w.data.vnetGW = vnetGW
	w.data.vnetGWIP = vnetGWIP
	w.data.switchName = switchName

	return nil
}

func (w *Watcher) validateASNRange(asns string) (int, int, error) {
	s := strings.Split(asns, "-")
	a := 0
	b := 0
	var err error
	if len(s) == 2 {
		a, err = strconv.Atoi(s[0])
		if err != nil {
			return a, b, err
		}
		if !(a > 0 && a <= 4294967294) {
			return a, b, fmt.Errorf("invalid ASN  range")
		}
		b, err = strconv.Atoi(s[1])
		if err != nil {
			return a, b, err
		}
		if !(b > 0 && b <= 4294967294) {
			return a, b, fmt.Errorf("invalid ASN  range")
		}

		if !(a < b) {
			return a, b, fmt.Errorf("invalid ASN  range")
		}
	} else {
		return a, b, fmt.Errorf("invalid ASN  range")
	}

	return a, b, nil
}

//  patchStringValue specifies a patch operation for a string.
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}
