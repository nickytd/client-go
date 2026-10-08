// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 19: Integration testing with envtest
//
// Demonstrates:
//   - sigs.k8s.io/controller-runtime/pkg/envtest — real API server in tests
//   - Starting/stopping the test environment
//   - Running tests against the real API server without a cluster
//
// NOTE: requires controller-runtime and envtest binaries.
// Install binaries: setup-envtest use 1.29 --bin-dir /usr/local/kubebuilder/bin
package main

import (
	"context"
	"log/slog"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func main() {
	binDir := os.Getenv("KUBEBUILDER_ASSETS")
	if binDir == "" {
		slog.Error("KUBEBUILDER_ASSETS not set",
			"hint", "setup-envtest use 1.37 --bin-dir .local/share/kubebuilder-envtest")
		os.Exit(1)
	}

	testEnv := &envtest.Environment{
		BinaryAssetsDirectory: binDir,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		slog.Error("error starting envtest", "err", err)
		os.Exit(1)
	}
	defer testEnv.Stop()

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		slog.Error("error creating clientset", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()

	cm, err := clientset.CoreV1().ConfigMaps("default").Create(ctx, &corev1.ConfigMap{
		Name: "envtest-cm", Namespace: "default",
		Data: map[string]string{"hello": "envtest"},
	}, metav1.CreateOptions{})
	if err != nil {
		slog.Error("error creating ConfigMap", "err", err)
		os.Exit(1)
	}

	slog.Info("created ConfigMap in envtest", "name", cm.Name, "data", cm.Data)
}
