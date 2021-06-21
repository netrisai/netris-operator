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

package lbwatcher

import (
	"github.com/netrisai/netris-operator/netrisstorage"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Watcher struct {
	Options  Options
	NStorage *netrisstorage.Storage
	MGR      manager.Manager
}

type selector struct {
	Key   string
	Value string
}

type lbIP struct {
	Name      string
	IP        string
	Port      int
	NodePort  int
	Protocol  string
	Automatic bool
}

type Options struct {
	LogLevel string
}
