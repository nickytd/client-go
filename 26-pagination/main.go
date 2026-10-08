// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 26: List pagination
//
// Large lists are chunked. ListOptions.Limit caps items per response; the
// server returns a Continue token when more remain. Feed it back on the next
// List until Continue is empty. This bounds client memory and API load.
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const seedCount = 7

func main() {
	clientset := mustClientset()
	ctx := context.Background()
	cms := clientset.CoreV1().ConfigMaps("default")

	// Seed enough objects that a small Limit forces multiple pages.
	for i := range seedCount {
		name := "page-demo-" + strconv.Itoa(i)
		_, _ = cms.Create(ctx, &corev1.ConfigMap{
			Name:   name,
			Labels: map[string]string{"demo": "pagination"},
		}, metav1.CreateOptions{})
	}

	// Page through with Limit=3, following Continue until it is empty.
	const pageSize = 3
	var (
		continueToken string
		page, total   int
	)
	for {
		list, err := cms.List(ctx, metav1.ListOptions{
			LabelSelector: "demo=pagination",
			Limit:         pageSize,
			Continue:      continueToken,
		})
		if err != nil {
			slog.Error("error listing configmaps", "err", err)
			os.Exit(1)
		}
		page++
		total += len(list.Items)
		slog.Info("page",
			"page", page, "items", len(list.Items), "continue", truncate(list.Continue))
		continueToken = list.Continue
		if continueToken == "" {
			break
		}
	}
	slog.Info("done", "items", total, "pages", page)

	// cleanup
	for i := range seedCount {
		_ = cms.Delete(ctx, "page-demo-"+strconv.Itoa(i), metav1.DeleteOptions{})
	}
}

// truncate shortens the opaque continue token for readable logs.
func truncate(s string) string {
	if len(s) > 12 {
		return s[:12] + "..."
	}
	return s
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
