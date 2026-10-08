// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 27: Field and label selectors
//
// Server-side filtering. Label selectors match user labels; field selectors
// match a restricted set of built-in fields (e.g. metadata.name,
// status.phase). Filtering on the server beats listing everything and
// filtering in Go: less data transferred, less client memory.
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	clientset := mustClientset()
	ctx := context.Background()
	cms := clientset.CoreV1().ConfigMaps("default")

	// Seed objects with distinguishing labels.
	for _, spec := range []struct{ name, tier string }{
		{"sel-a", "frontend"}, {"sel-b", "frontend"}, {"sel-c", "backend"},
	} {
		_, _ = cms.Create(ctx, &corev1.ConfigMap{
			Name:   spec.name,
			Labels: map[string]string{"tier": spec.tier},
		}, metav1.CreateOptions{})
	}

	// Label selector built with the typed helper (equivalent to "tier=frontend").
	labelSel := labels.SelectorFromSet(labels.Set{"tier": "frontend"}).String()
	byLabel, err := cms.List(ctx, metav1.ListOptions{LabelSelector: labelSel})
	if err != nil {
		slog.Error("error listing by label", "err", err)
		os.Exit(1)
	}
	slog.Info("list by label", "selector", labelSel, "matched", names(byLabel.Items))

	// Field selector: match a built-in field. metadata.name is always indexable.
	fieldSel := fields.OneTermEqualSelector("metadata.name", "sel-c").String()
	byField, err := cms.List(ctx, metav1.ListOptions{FieldSelector: fieldSel})
	if err != nil {
		slog.Error("error listing by field", "err", err)
		os.Exit(1)
	}
	slog.Info("list by field", "selector", fieldSel, "matched", names(byField.Items))

	// cleanup
	for _, n := range []string{"sel-a", "sel-b", "sel-c"} {
		_ = cms.Delete(ctx, n, metav1.DeleteOptions{})
	}
}

func names(items []corev1.ConfigMap) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
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
