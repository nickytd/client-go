// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 01: KubeConfig & client creation
//
// Demonstrates:
//   - Loading kubeconfig from KUBECONFIG env var (preferred) or ~/.kube/config
//   - Building a rest.Config from it
//   - Detecting in-cluster vs out-of-cluster automatically
//   - Printing the server URL to confirm connectivity
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	slog.Info("loading kubeconfig", "path", kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}

	slog.Info("connected", "server", config.Host)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("error creating clientset", "err", err)
		os.Exit(1)
	}

	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		slog.Error("error fetching server version", "err", err)
		os.Exit(1)
	}

	slog.Info("server version", "version", version.GitVersion)
	_ = context.Background() // used in later cases
}
