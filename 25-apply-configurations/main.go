// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 25: Typed apply configurations (server-side apply)
//
// Case 15 did SSA with hand-written JSON bytes. This is the typed path:
// applyconfigurations builders produce a strongly-typed, nil-omitting apply
// object, and Apply() with a FieldManager records ownership. A second manager
// applying a conflicting field is rejected unless Force is set.
//
// See also case 15 (raw patch strategies).
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyconfigcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	clientset := mustClientset()
	ctx := context.Background()
	cms := clientset.CoreV1().ConfigMaps("default")

	// Manager "alice" owns key1. Apply creates the object if absent.
	alice := applyconfigcorev1.ConfigMap("apply-demo", "default").
		WithData(map[string]string{"key1": "owned-by-alice"})
	out, err := cms.Apply(ctx, alice, metav1.ApplyOptions{FieldManager: "alice"})
	if err != nil {
		slog.Error("error applying as alice", "err", err)
		os.Exit(1)
	}
	printOwners("after alice", out)

	// Manager "bob" adds a different key — disjoint fields, no conflict.
	bob := applyconfigcorev1.ConfigMap("apply-demo", "default").
		WithData(map[string]string{"key2": "owned-by-bob"})
	out, err = cms.Apply(ctx, bob, metav1.ApplyOptions{FieldManager: "bob"})
	if err != nil {
		slog.Error("error applying as bob", "err", err)
		os.Exit(1)
	}
	printOwners("after bob", out)

	// Bob now tries to overwrite key1, which alice owns -> conflict (expected).
	conflicting := applyconfigcorev1.ConfigMap("apply-demo", "default").
		WithData(map[string]string{"key1": "stolen-by-bob"})
	_, err = cms.Apply(ctx, conflicting, metav1.ApplyOptions{FieldManager: "bob"})
	slog.Info("bob overwrites alice's field without force", "err", err)

	// With Force, bob wins and takes ownership of key1.
	out, err = cms.Apply(ctx, conflicting, metav1.ApplyOptions{FieldManager: "bob", Force: true})
	if err != nil {
		slog.Error("error force-applying as bob", "err", err)
		os.Exit(1)
	}
	printOwners("after bob --force", out)

	_ = cms.Delete(ctx, "apply-demo", metav1.DeleteOptions{})
}

// printOwners dumps data plus which manager owns which fields.
func printOwners(stage string, cm *corev1.ConfigMap) {
	slog.Info(stage, "data", cm.Data)
	for _, mf := range cm.ManagedFields {
		var owned map[string]any
		if mf.FieldsV1 != nil {
			_ = json.Unmarshal(mf.FieldsV1.GetRawBytes(), &owned)
		}
		slog.Info("field ownership", "manager", mf.Manager, "owns", owned)
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
