// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 08: Indexers & custom store queries
//
// Demonstrates:
//   - cache.Indexers — adding custom index functions
//   - cache.MetaNamespaceIndexFunc (built-in)
//   - Custom index: pods by node name
//   - store.ByIndex("byNode", nodeName)
package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const indexByNode = "byNode"

func main() {
	clientset := mustClientset()

	lw := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods", "",
		fields.Everything(),
	)

	indexers := cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		indexByNode: func(obj any) ([]string, error) {
			pod, ok := obj.(*corev1.Pod)
			if !ok || pod.Spec.NodeName == "" {
				return nil, nil
			}
			return []string{pod.Spec.NodeName}, nil
		},
	}

	store, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: lw,
		ObjectType:    &corev1.Pod{},
		Handler:       cache.ResourceEventHandlerFuncs{},
		ResyncPeriod:  30 * time.Second,
		Indexers:      indexers,
	})

	// NewInformerWithOptions returns a Store; because Indexers were supplied it is
	// backed by an Indexer, so assert to reach ByIndex/ListIndexFuncValues.
	indexer := store.(cache.Indexer)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)

	if !cache.WaitForCacheSync(stop, controller.HasSynced) {
		slog.Error("cache sync timed out")
		os.Exit(1)
	}

	nodes := indexer.ListIndexFuncValues(indexByNode)
	slog.Info("nodes with pods", "nodes", nodes)
	for _, node := range nodes {
		items, _ := indexer.ByIndex(indexByNode, node)
		slog.Info("node pods", "node", node, "count", len(items))
	}
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
