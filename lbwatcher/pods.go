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
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getPods(clientset *kubernetes.Clientset, namespace string) (*v1.PodList, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return pods, fmt.Errorf("{getPods} %s", err)
	}
	return pods, nil
}

func filterPodsBySelector(pods *v1.PodList, selectorKey, selectorValue string) []v1.Pod {
	filteredPods := []v1.Pod{}
	for _, pod := range pods.Items {
		for labelKey, labelValue := range pod.Labels {
			if labelKey == selectorKey && labelValue == selectorValue {
				filteredPods = append(filteredPods, pod)
			}
		}
	}
	return filteredPods
}
