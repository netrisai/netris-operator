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

package calico

import (
	"context"
	"time"
)

var (
	cntxt          = context.Background()
	contextTimeout = time.Duration(10 * time.Second)
)

type Calico struct {
	options Options
}

type Options struct {
	ContextTimeout int
}

func New(options Options) *Calico {
	if options.ContextTimeout > 0 {
		contextTimeout = time.Duration(time.Duration(options.ContextTimeout) * time.Second)
	}
	return &Calico{
		options: options,
	}
}
