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

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
)

func getServices(clientset *kubernetes.Clientset, namespace string) (*v1.ServiceList, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return services, fmt.Errorf("{getServices} %s", err)
	}
	return services, nil
}

func assignIngress(clientset *kubernetes.Clientset, ips []string, namespace string, name string) (*v1.Service, error) {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()

	var ingressList []v1.LoadBalancerIngress

	for _, ip := range ips {
		ingressList = append(ingressList, v1.LoadBalancerIngress{IP: ip})
	}

	// ingressList = append(ingressList, v1.LoadBalancerIngress{IP: "aaaaa"})

	var updatedService *v1.Service

	service, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return updatedService, err
	}

	service.Status = v1.ServiceStatus{LoadBalancer: v1.LoadBalancerStatus{Ingress: ingressList}}

	updatedService, updateErr := clientset.CoreV1().Services(namespace).UpdateStatus(context.TODO(), service, metav1.UpdateOptions{})

	return updatedService, updateErr
}

func eventRecorder(kubeClient *kubernetes.Clientset) (record.EventRecorder, watch.Interface, record.EventBroadcaster) {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logrus.New().Debugf)
	w := eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: kubeClient.CoreV1().Events(""),
		},
	)

	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		v1.EventSource{},
	)

	return recorder, w, eventBroadcaster
}

func createEvent(clientset *kubernetes.Clientset, recorder record.EventRecorder, namespace, name, reason, message string) error {
	ctx, cancel := context.WithTimeout(cntxt, contextTimeout)
	defer cancel()
	service, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("{createEvent} %s", err)
	}

	ref, err := reference.GetReference(scheme.Scheme, service)
	if err != nil {
		return fmt.Errorf("{createEvent} %s", err)
	}
	recorder.Event(ref, v1.EventTypeWarning, reason, message)

	return nil
}
