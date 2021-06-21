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
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/netrisai/netris-operator/api/v1alpha1"
	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/calicowatcher/calico"
	"github.com/netrisai/netris-operator/netrisstorage"
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
	Options  Options
	NStorage *netrisstorage.Storage
	MGR      manager.Manager
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

var logger logr.Logger // debugLogger logr.InfoLogger

func (w *Watcher) getRestConfig() *rest.Config {
	return ctrl.GetConfigOrDie()
}

func (w *Watcher) start() {
	restClient := w.getRestConfig()
	cl := w.MGR.GetClient()
	// recorder, w, _ := eventRecorder(clientset)
	// defer w.Stop()
	err := w.mainProcessing(cl, restClient)
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
	// debugLogger = logger.V(int(zapcore.WarnLevel))

	ticker := time.NewTicker(10 * time.Second)
	w.start()
	for {
		<-ticker.C
		w.start()
	}
}

func (w *Watcher) mainProcessing(cl client.Client, restClient *rest.Config) error {
	bgpConfs, err := calico.GetBGPConfiguration(restClient)
	if err != nil {
		logger.Error(err, "")
	}
	if !w.checkBGPConfigurations(bgpConfs) {
		return fmt.Errorf("Netris annotation is not present")
	}

	ipPools, err := calico.GetIPPool(restClient)
	if err != nil {
		logger.Error(err, "")
	}

	if len(ipPools) == 0 && ipPools[0] != nil {
		return fmt.Errorf("IPPool is missing")
	}

	if len(bgpConfs[0].Spec.ServiceClusterIPs) == 0 {
		return fmt.Errorf("serviceCIDRs are missing")
	}

	if len(bgpConfs[0].Spec.ServiceClusterIPs) == 0 {
		return fmt.Errorf("ServiceClusterIPs are missing")
	}

	var (
		blockSize   = ipPools[0].Spec.BlockSize
		clusterCIDR = ipPools[0].Spec.CIDR
		serviceCIDR = bgpConfs[0].Spec.ServiceClusterIPs[0].CIDR
	)

	clientset, err := kubernetes.NewForConfig(restClient)
	if err != nil {
		return err
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	nodesMap := make(map[string]*nodeIP)
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
			// Get free ASN from calico dedicated range(read in loop already used ASNs from nodesInfo map and take next usable) ?????????????
			// Get free range from ENV variables and fill empty spaces
			// Example 4200070000 - 4200079000
			// ASNs should be unique
			// asn = ""
			_, err := clientset.CoreV1().Nodes().Update(context.Background(), node.DeepCopy(), metav1.UpdateOptions{})
			if err != nil {
				return err
			}
			continue
		} else {
			asn = anns["projectcalico.org/ASNumber"]
		}

		tmpNode := &nodeIP{
			IP:   anns["projectcalico.org/IPv4Address"],
			IPIP: anns["projectcalico.org/IPv4IPIPTunnelAddr"],
			ASN:  asn,
		}
		nodesMap[node.Name] = tmpNode
	}

	bgpList := []*k8sv1alpha1.BGP{}

	for name, node := range nodesMap {
		asn, err := strconv.Atoi(node.ASN)
		if err != nil {
			return err
		}
		bgp := &k8sv1alpha1.BGP{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", name, node.IP),
				Namespace: "Default",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "BGP",
				APIVersion: "k8s.netris.ai/v1alpha1",
			},
			Spec: k8sv1alpha1.BGPSpec{
				Site:       "", // Site ???????????????????????????? Get from Node IP Subnet
				NeighborAS: asn,
				TerminateOnSwitch: k8sv1alpha1.BGPTerminateOnSwitch{
					Enabled:    true,
					SwitchName: "", // Swich Name ????????????????????????????
				},
				Transport: v1alpha1.BGPTransport{
					Type: "vnet",
					Name: "", // Vnet Name ???????????????????????????? Get from Node IP subnet
				},
				LocalIP:  "", // Vnet Gateway ????????????????????????????
				RemoteIP: node.IP,
				PrefixListInbound: []string{
					fmt.Sprintf("permit %s le %d", clusterCIDR, blockSize),
					fmt.Sprintf("permit %s le %d", serviceCIDR, 32),
				},
				PrefixListOutbound: []string{
					"permit 0.0.0.0/0",
					fmt.Sprintf("deny %s/%d", node.IPIP, blockSize),
					fmt.Sprintf("permit %s le %d", clusterCIDR, blockSize),
				},
			},
		}
		bgpList = append(bgpList, bgp)
	}

	bgps, err := getBGPs(cl)
	if err != nil {
		return err
	}

	// peersProcessing(cl, restClient)
	return nil
}

// func getSiteByIP(ip string) (string, error) {
// 	controllers.NStorage.SubnetsStorage
// 	return "", nil
// }

func getBGPs(cl client.Client) (*k8sv1alpha1.BGPList, error) {
	bgps := &k8sv1alpha1.BGPList{}
	err := cl.List(context.Background(), bgps, &client.ListOptions{})
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

// func peersProcessing(cl client.Client, restClient *rest.Config) {
// 	ipPools, err := calico.GetIPPool(restClient)
// 	if err != nil {
// 		logger.Error(err, "")
// 	}
// 	js, _ := json.Marshal(ipPools)
// 	fmt.Println(string(js))
// }

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
