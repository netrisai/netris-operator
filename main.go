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

package main

import (
	"flag"
	"log"
	"os"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	api "github.com/netrisai/netriswebapi/v2"

	k8sv1alpha1 "github.com/netrisai/netris-operator/api/v1alpha1"
	"github.com/netrisai/netris-operator/calicowatcher"
	"github.com/netrisai/netris-operator/configloader"
	"github.com/netrisai/netris-operator/controllers"
	"github.com/netrisai/netris-operator/lbwatcher"
	"github.com/netrisai/netris-operator/netrisstorage"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	// cred stores the Netris API usepoint.
	cred *api.Clientset

	// nStorage is the instance of the Netris API in-memory storage.
	nStorage *netrisstorage.Storage
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(k8sv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	if configloader.Root.LogDevMode {
		ctrl.SetLogger(zap.New(zap.Level(zapcore.DebugLevel), zap.UseDevMode(false)))
	} else {
		ctrl.SetLogger(zap.New(zap.UseDevMode(false), zap.StacktraceLevel(zapcore.DPanicLevel)))
	}

	var err error
	cred, err = api.Client(configloader.Root.Controller.Host, configloader.Root.Controller.Login, configloader.Root.Controller.Password, configloader.Root.RequeueInterval)
	if err != nil {
		log.Panicf("newHTTPCredentials error %v", err)
	}
	cred.Client.InsecureVerify(configloader.Root.Controller.Insecure)
	err = cred.Client.LoginUser()
	if err != nil {
		log.Printf("LoginUser error %v", err)
		os.Exit(1)
	}
	go cred.Client.CheckAuthWithInterval()

	nStorage = netrisstorage.NewStorage(cred)
	err = nStorage.Download()
	if err != nil {
		log.Printf("Storage.Download() error %v", err)
	}
	go nStorage.DownloadWithInterval()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Namespace:          "",
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "abac3abe.netris.ai",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.VNetReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("VNet"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VNet")
		os.Exit(1)
	}
	if err = (&controllers.VNetMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("VNetMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VNetMeta")
		os.Exit(1)
	}

	if err = (&controllers.BGPReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("BGP"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BGP")
		os.Exit(1)
	}
	if err = (&controllers.BGPMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("BGPMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BGPMeta")
		os.Exit(1)
	}
	if err = (&controllers.L4LBReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("L4LB"),
		Scheme:     mgr.GetScheme(),
		Cred:       cred,
		NStorage:   nStorage,
		L4LBTenant: configloader.Root.L4lbTenant,
		VPCID:      configloader.Root.VPCID,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "L4LB")
		os.Exit(1)
	}
	if err = (&controllers.L4LBMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("L4LBMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
		VPCID:    configloader.Root.VPCID,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "L4LBMeta")
		os.Exit(1)
	}
	if err = (&controllers.SiteReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("Site"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Site")
		os.Exit(1)
	}
	if err = (&controllers.SiteMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("SiteMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SiteMeta")
		os.Exit(1)
	}
	if err = (&controllers.AllocationReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("Allocation"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Allocation")
		os.Exit(1)
	}
	if err = (&controllers.AllocationMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("AllocationMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AllocationMeta")
		os.Exit(1)
	}
	if err = (&controllers.SubnetReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("Subnet"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Subnet")
		os.Exit(1)
	}
	if err = (&controllers.SubnetMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("SubnetMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SubnetMeta")
		os.Exit(1)
	}
	if err = (&controllers.SoftgateReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("Softgate"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Softgate")
		os.Exit(1)
	}
	if err = (&controllers.SoftgateMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("SoftgateMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SoftgateMeta")
		os.Exit(1)
	}
	if err = (&controllers.SwitchReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("Switch"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Switch")
		os.Exit(1)
	}
	if err = (&controllers.SwitchMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("SwitchMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SwitchMeta")
		os.Exit(1)
	}
	if err = (&controllers.ControllerReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("Controller"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Controller")
		os.Exit(1)
	}
	if err = (&controllers.ControllerMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("ControllerMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ControllerMeta")
		os.Exit(1)
	}
	if err = (&controllers.LinkReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("Link"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Link")
		os.Exit(1)
	}
	if err = (&controllers.LinkMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("LinkMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LinkMeta")
		os.Exit(1)
	}
	if err = (&controllers.NatReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("Nat"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Nat")
		os.Exit(1)
	}
	if err = (&controllers.NatMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("NatMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NatMeta")
		os.Exit(1)
	}
	if err = (&controllers.InventoryProfileReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("InventoryProfile"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "InventoryProfile")
		os.Exit(1)
	}
	if err = (&controllers.InventoryProfileMetaReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("InventoryProfileMeta"),
		Scheme:   mgr.GetScheme(),
		Cred:     cred,
		NStorage: nStorage,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "InventoryProfileMeta")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	watcherLogLevel := "info"
	if configloader.Root.LogDevMode {
		watcherLogLevel = "debug"
	}

	lbWatcher, err := lbwatcher.NewWatcher(nStorage, mgr, lbwatcher.Options{LogLevel: watcherLogLevel, RequeueInterval: configloader.Root.RequeueInterval})
	if err != nil {
		setupLog.Error(err, "problem running lbwatcher")
		os.Exit(1)
	}
	go lbWatcher.Start()

	cWatcher, err := calicowatcher.NewWatcher(nStorage, mgr, calicowatcher.Options{LogLevel: watcherLogLevel, RequeueInterval: configloader.Root.RequeueInterval})
	if err != nil {
		setupLog.Error(err, "problem running calicowatcher")
		os.Exit(1)
	}
	go cWatcher.Start()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
