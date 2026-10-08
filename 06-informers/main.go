// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 06: SharedInformer
//
// Demonstrates:
//   - cache.NewSharedInformer — single-resource informer
//   - AddEventHandler with ResourceEventHandlerFuncs
//   - WaitForCacheSync — waiting for the local cache to be warm
//   - Reading from the informer's in-memory store
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

func main() {
	clientset := mustClientset()

	lw := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"configmaps",
		"", // all namespaces
		fields.Everything(),
	)

	informer := cache.NewSharedInformer(lw, &corev1.ConfigMap{}, 30*time.Second)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			cm := obj.(*corev1.ConfigMap)
			slog.Info("configmap added", "namespace", cm.Namespace, "name", cm.Name)
		},
		UpdateFunc: func(old, new any) {
			cm := new.(*corev1.ConfigMap)
			slog.Info("configmap updated", "namespace", cm.Namespace, "name", cm.Name)
		},
		DeleteFunc: func(obj any) {
			cm := obj.(*corev1.ConfigMap)
			slog.Info("configmap deleted", "namespace", cm.Namespace, "name", cm.Name)
		},
	})

	stop := make(chan struct{})
	defer close(stop)
	go informer.Run(stop)

	if !cache.WaitForCacheSync(stop, informer.HasSynced) {
		slog.Error("timed out waiting for cache sync")
		os.Exit(1)
	}
	slog.Info("cache synced", "configmap", len(informer.GetStore().List()))

	time.Sleep(60 * time.Second)
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
