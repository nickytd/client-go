// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 14: Informer for a CRD (dynamic informer)
//
// Demonstrates:
//   - dynamicinformer.NewDynamicSharedInformerFactory
//   - Watching a custom resource without generated types
//   - Handling events on unstructured objects
package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var widgetGVR = schema.GroupVersionResource{
	Group:    "example.com",
	Version:  "v1alpha1",
	Resource: "widgets",
}

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		slog.Error("error creating dynamic client", "err", err)
		os.Exit(1)
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynClient, 30*time.Second)
	informer := factory.ForResource(widgetGVR).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			ns, name := widgetID(obj)
			slog.Info("widget added", "namespace", ns, "name", name)
		},
		UpdateFunc: func(_, newObj any) {
			ns, name := widgetID(newObj)
			slog.Info("widget updated", "namespace", ns, "name", name)
		},
		DeleteFunc: func(obj any) {
			ns, name := widgetID(obj)
			slog.Info("widget deleted", "namespace", ns, "name", name)
		},
	})

	stop := make(chan struct{})
	defer close(stop)
	factory.Start(stop)

	if !cache.WaitForCacheSync(stop, informer.HasSynced) {
		slog.Error("timed out waiting for cache sync (is the Widget CRD installed?)")
		os.Exit(1)
	}

	slog.Info("watching Widgets — create/delete some to see events")
	time.Sleep(60 * time.Second)
}

// widgetID extracts the namespace and name from an informer event object.
// Dynamic informers deliver *unstructured.Unstructured; a delete event may
// instead carry a cache.DeletedFinalStateUnknown tombstone wrapping the last
// known object, so we unwrap that first.
func widgetID(obj any) (namespace, name string) {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return "", ""
	}
	return u.GetNamespace(), u.GetName()
}

// loadConfig builds a *rest.Config from the KUBECONFIG env var, falling back to
// ~/.kube/config.
func loadConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
