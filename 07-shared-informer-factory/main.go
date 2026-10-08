// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 07: SharedInformerFactory
//
// Demonstrates:
//   - informers.NewSharedInformerFactory — factory managing many informers
//   - Typed informers: factory.Core().V1().Pods()
//   - Lister pattern: podLister.Pods(ns).Get(name)
//   - factory.WaitForCacheSync
package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	clientset := mustClientset()

	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

	podInformer := factory.Core().V1().Pods()
	nodeInformer := factory.Core().V1().Nodes()

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) { /* handle add */ },
	})
	_ = nodeInformer.Informer()

	stop := make(chan struct{})
	defer close(stop)
	factory.Start(stop)

	synced := factory.WaitForCacheSync(stop)
	for t, ok := range synced {
		if !ok {
			slog.Error("cache never synced", "type", t)
			os.Exit(1)
		}
	}

	pods, err := podInformer.Lister().Pods("kube-system").List(labels.Everything())
	if err != nil {
		slog.Error("error listing pods from cache", "err", err)
		os.Exit(1)
	}
	slog.Info("kube-system pods in cache", "count", len(pods))

	nodes, err := nodeInformer.Lister().List(labels.Everything())
	if err != nil {
		slog.Error("error listing nodes from cache", "err", err)
		os.Exit(1)
	}
	slog.Info("nodes in cache", "count", len(nodes))
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

// mustClientset builds a typed clientset from the local kubeconfig, exiting on
// error.
func mustClientset() *kubernetes.Clientset {
	config, err := loadConfig()
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("error creating clientset", "err", err)
		os.Exit(1)
	}
	return clientset
}
