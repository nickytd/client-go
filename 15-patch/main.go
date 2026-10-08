// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 15: Patch strategies
//
// Demonstrates:
//   - JSON merge patch (types.MergePatchType)
//   - Strategic merge patch (types.StrategicMergePatchType)
//   - Server-side apply (types.ApplyPatchType) with field manager
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/kr/pretty"
)

func main() {
	clientset := mustClientset()

	ctx := context.Background()

	cm := &corev1.ConfigMap{
		Name: "patch-demo", Namespace: "default",
		Data: map[string]string{"key1": "original"},
	}
	created, _ := clientset.CoreV1().ConfigMaps("default").Create(ctx, cm, metav1.CreateOptions{})
	printState("after create", created)

	mergePatch, _ := json.Marshal(map[string]any{
		"data": map[string]string{"key2": "added-by-merge-patch"},
	})
	merged, _ := clientset.CoreV1().ConfigMaps("default").Patch(ctx, "patch-demo",
		types.MergePatchType, mergePatch, metav1.PatchOptions{})
	printState("after merge patch", merged)

	strategicPatch, _ := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"labels": map[string]string{"patched": "strategic"},
		},
	})
	strategic, _ := clientset.CoreV1().ConfigMaps("default").Patch(ctx, "patch-demo",
		types.StrategicMergePatchType, strategicPatch, metav1.PatchOptions{})
	printState("after strategic merge patch", strategic)

	// Apply bodies must be self-describing typed objects, so apiVersion/kind
	// are required; name/namespace come from the request path and are omitted.
	ssaPatch := `{"apiVersion":"v1","kind":"ConfigMap","data":{"key3":"ssa-value"}}`
	applied, _ := clientset.CoreV1().ConfigMaps("default").Patch(ctx, "patch-demo",
		types.ApplyPatchType, []byte(ssaPatch),
		metav1.PatchOptions{FieldManager: "client-go-lab", Force: new(true)})
	printState("after server-side apply", applied)
	printManagedFields(applied)

	clientset.CoreV1().ConfigMaps("default").Delete(ctx, "patch-demo", metav1.DeleteOptions{})
}

// printState renders the ConfigMap's labels and data so each patch's effect on
// the stored object is readable on the console. kr/pretty formats the Go value
// across multiple indented lines, unlike slog's single-line key=value escaping.
func printState(stage string, cm *corev1.ConfigMap) {
	if cm == nil {
		slog.Warn(stage, "err", "nil ConfigMap (patch failed)")
		return
	}
	view := map[string]any{
		"labels": cm.Labels,
		"data":   cm.Data,
	}
	slog.Info(stage)
	// The multi-line pretty output is written directly (not as a slog field) so
	// its indented structure stays intact — slog would collapse it onto one line.
	fmt.Printf("%# v\n", pretty.Formatter(view))
}

// printManagedFields shows the object's ManagedFields — server-side apply
// records who owns which fields here. The labels/data view hides this
// bookkeeping, so we decode each manager's owned field set (stored as raw
// JSON) and dump it with kr/pretty.
func printManagedFields(cm *corev1.ConfigMap) {
	if cm == nil {
		return
	}
	fmt.Print("\n=== managedFields (SSA ownership) ===\n")
	for _, mf := range cm.ManagedFields {
		var owned map[string]any
		if mf.FieldsV1 != nil {
			if err := json.Unmarshal(mf.FieldsV1.GetRawBytes(), &owned); err != nil {
				slog.Warn("decoding managed fields", "manager", mf.Manager, "err", err)
			}
		}
		fmt.Printf("manager=%s operation=%s owns:\n", mf.Manager, mf.Operation)
		fmt.Printf("%# v\n", pretty.Formatter(owned))
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
