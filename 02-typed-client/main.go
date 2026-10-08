// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 02: Typed clientset
//
// Demonstrates:
//   - kubernetes.NewForConfig (typed clientset)
//   - CoreV1() — list Pods across all namespaces
//   - AppsV1() — list Deployments in default namespace
//   - Basic CRUD: create, get, delete a ConfigMap
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	clientset := mustClientset()

	ctx := context.Background()

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Error("error listing pods", "err", err)
		os.Exit(1)
	}
	slog.Info("pods across all namespaces", "count", len(pods.Items))

	deploys, err := clientset.AppsV1().Deployments("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Error("error listing deployments", "err", err)
		os.Exit(1)
	}
	slog.Info("deployments in default", "count", len(deploys.Items))

	cm := &corev1.ConfigMap{
		Name: "client-go-lab", Namespace: "default",
		Data: map[string]string{"hello": "client-go"},
	}
	created, err := clientset.CoreV1().ConfigMaps("default").Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		slog.Error("error creating configmap", "err", err)
		os.Exit(1)
	}
	slog.Info("created ConfigMap", "name", created.Name)

	_ = clientset.CoreV1().ConfigMaps("default").Delete(ctx, created.Name, metav1.DeleteOptions{})
	slog.Info("deleted ConfigMap", "name", created.Name)
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
