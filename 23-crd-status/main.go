// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 23: CRD status subresource
//
// Spec is desired state (set by users); status is observed state (set by the
// controller). When a CRD enables the /status subresource, spec and status are
// written through SEPARATE endpoints: a normal Update cannot change status, and
// UpdateStatus cannot change spec. This case writes status via the dynamic
// client's /status subresource and sets observedGeneration.
//
// See also case 16 (/status on built-in pods) and case 13 (the typed CRD).
//
// Prerequisite: install this folder's status-enabled Widget CRD before running:
//
//	kubectl apply -f widget_crd.yaml
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var gvr = schema.GroupVersionResource{
	Group:    "example.com",
	Version:  "v1alpha1",
	Resource: "widgets",
}

const (
	apiVersion = "example.com/v1alpha1"
	kind       = "Widget"
	namespace  = "default"
	name       = "status-demo"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		slog.Error("error creating dynamic client", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()
	ri := dyn.Resource(gvr).Namespace(namespace)

	// Create the custom resource (spec only).
	cr := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   map[string]any{"name": name},
		"spec":       map[string]any{"color": "blue", "size": int64(3)},
	}}
	created, err := ri.Create(ctx, cr, metav1.CreateOptions{})
	if err != nil {
		slog.Error("error creating CR (is the widget_crd.yaml from this folder installed?)", "err", err)
		os.Exit(1)
	}
	gen := created.GetGeneration()
	slog.Info("created", "namespace", namespace, "name", name, "generation", gen)

	// Set status via the /status subresource. observedGeneration records which
	// spec generation this status reflects — the standard staleness signal.
	if err := unstructured.SetNestedMap(created.Object, map[string]any{
		"observedGeneration": gen,
		"phase":              "Ready",
	}, "status"); err != nil {
		slog.Error("error setting nested status map", "err", err)
		os.Exit(1)
	}
	updated, err := ri.UpdateStatus(ctx, created, metav1.UpdateOptions{})
	if err != nil {
		slog.Error("error updating status (is subresources.status enabled on the CRD?)", "err", err)
		os.Exit(1)
	}
	status, _, _ := unstructured.NestedMap(updated.Object, "status")
	slog.Info("status written", "status", status)

	// Prove spec/status separation: a plain Update does NOT persist status.
	if err := unstructured.SetNestedField(updated.Object, "ShouldBeIgnored", "status", "phase"); err != nil {
		slog.Error("error setting nested field", "err", err)
		os.Exit(1)
	}
	afterUpdate, err := ri.Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		slog.Error("error updating spec", "err", err)
		os.Exit(1)
	}
	phase, _, _ := unstructured.NestedString(afterUpdate.Object, "status", "phase")
	slog.Info("phase after plain Update (unchanged by design)", "phase", phase)

	_ = ri.Delete(ctx, name, metav1.DeleteOptions{})
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
