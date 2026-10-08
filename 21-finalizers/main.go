// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 21: Finalizers
//
// Demonstrates the finalizer lifecycle on a ConfigMap:
//   - add a finalizer, then issue Delete
//   - the object is NOT removed; DeletionTimestamp is set instead
//   - perform cleanup, then remove the finalizer to let GC complete
//
// See also case 11 (owner references & garbage collection).
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const finalizerName = "client-go-lab/cleanup"

func main() {
	clientset := mustClientset()
	ctx := context.Background()
	cms := clientset.CoreV1().ConfigMaps("default")

	// 1. Create a ConfigMap carrying our finalizer.
	cm := &corev1.ConfigMap{
		Name:       "finalizer-demo",
		Finalizers: []string{finalizerName},
		Data:       map[string]string{"key": "value"},
	}
	created, err := cms.Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		slog.Error("error creating configmap", "err", err)
		os.Exit(1)
	}
	slog.Info("created", "finalizers", created.Finalizers)

	// 2. Delete — because a finalizer is present, the API server only marks
	//    the object for deletion (sets DeletionTimestamp) rather than removing it.
	if err := cms.Delete(ctx, "finalizer-demo", metav1.DeleteOptions{}); err != nil {
		slog.Error("error deleting configmap", "err", err)
		os.Exit(1)
	}
	marked, err := cms.Get(ctx, "finalizer-demo", metav1.GetOptions{})
	if err != nil {
		slog.Error("error getting configmap after delete", "err", err)
		os.Exit(1)
	}
	slog.Info("still present after Delete",
		"deletionTimestamp", marked.DeletionTimestamp, "finalizers", marked.Finalizers)

	// 3. Do cleanup work here (none needed for the demo), then remove the
	//    finalizer. Once the last finalizer is gone the object is reaped.
	marked.Finalizers = slices.DeleteFunc(marked.Finalizers, func(f string) bool {
		return f == finalizerName
	})
	if _, err := cms.Update(ctx, marked, metav1.UpdateOptions{}); err != nil {
		slog.Error("error removing finalizer", "err", err)
		os.Exit(1)
	}

	// 4. Confirm it is gone.
	_, err = cms.Get(ctx, "finalizer-demo", metav1.GetOptions{})
	slog.Info("get after finalizer removed", "err", err)
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
