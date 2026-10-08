// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 11: Owner references & garbage collection
//
// Demonstrates:
//   - Setting metav1.OwnerReference on a child resource
//   - Cascading deletion: delete owner → children auto-deleted
//   - BlockOwnerDeletion and Controller flags
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

	parent, err := clientset.CoreV1().ConfigMaps("default").Create(ctx, &corev1.ConfigMap{
		Name: "parent-cm",
	}, metav1.CreateOptions{})
	if err != nil {
		slog.Error("error creating parent ConfigMap", "err", err)
		os.Exit(1)
	}
	slog.Info("created parent", "name", parent.Name, "uid", parent.UID)

	isController := true
	blockDeletion := true

	child, err := clientset.CoreV1().ConfigMaps("default").Create(ctx, &corev1.ConfigMap{
		Name: "child-cm",
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion:         "v1",
				Kind:               "ConfigMap",
				Name:               parent.Name,
				UID:                parent.UID,
				Controller:         &isController,
				BlockOwnerDeletion: &blockDeletion,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		slog.Error("error creating child ConfigMap", "err", err)
		os.Exit(1)
	}
	slog.Info("created child", "name", child.Name, "owner", child.OwnerReferences[0].Name)

	propagation := metav1.DeletePropagationForeground
	_ = clientset.CoreV1().ConfigMaps("default").Delete(ctx, parent.Name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	slog.Info("deleted parent with Foreground propagation — child will be GC'd by Kubernetes")
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
