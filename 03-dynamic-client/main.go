// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 03: Dynamic client
//
// Demonstrates:
//   - dynamic.NewForConfig
//   - Working with schema.GroupVersionResource (GVR)
//   - Listing any resource as unstructured.UnstructuredList
//   - Extracting fields from unstructured data
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

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

	podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

	list, err := dynClient.Resource(podGVR).Namespace("kube-system").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		slog.Error("error listing pods", "err", err)
		os.Exit(1)
	}

	for _, item := range list.Items {
		name, _, _ := unstructuredString(item.Object, "metadata", "name")
		phase, _, _ := unstructuredString(item.Object, "status", "phase")
		slog.Info("pod", "name", name, "phase", phase)
	}
}

func unstructuredString(obj map[string]any, keys ...string) (string, bool, error) {
	cur := obj
	for i, k := range keys {
		val, ok := cur[k]
		if !ok {
			return "", false, nil
		}
		if i == len(keys)-1 {
			s, ok := val.(string)
			return s, ok, nil
		}
		cur, ok = val.(map[string]any)
		if !ok {
			return "", false, nil
		}
	}
	return "", false, nil
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
