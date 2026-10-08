// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 05: List & Watch
//
// Demonstrates:
//   - cache.NewListWatchFromClient — low-level list+watch loop
//   - resource version tracking
//   - Handling ADDED / MODIFIED / DELETED watch events
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	clientset := mustClientset()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	watcher, err := clientset.CoreV1().ConfigMaps("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Error("error starting watch", "err", err)
		os.Exit(1)
	}
	defer watcher.Stop()

	slog.Info("watching configmaps for 30 seconds (create/delete pods to see events)")
	for event := range watcher.ResultChan() {
		cm, ok := event.Object.(*corev1.ConfigMap)
		if !ok {
			continue
		}
		switch event.Type {
		case watch.Added:
			slog.Info("configmap added", "namespace", cm.Namespace, "name", cm.Name)
		case watch.Modified:
			slog.Info("configmap modified", "namespace", cm.Namespace, "name", cm.Name)
		case watch.Deleted:
			slog.Info("configmap deleted", "namespace", cm.Namespace, "name", cm.Name)
		}
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
