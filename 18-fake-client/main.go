// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 18: Testing with fake clients
//
// Demonstrates:
//   - fake.NewSimpleClientset — in-memory clientset for unit tests
//   - Pre-seeding objects with fake.NewSimpleClientset(objs...)
//   - Asserting actions via clientset.Actions()
//   - Testing a controller without a real cluster
package main

import (
	"context"
	"log/slog"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// ensureConfigMap is the function under test.
func ensureConfigMap(ctx context.Context, clientset *fake.Clientset, ns, name string) error {
	_, err := clientset.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
			Name: name, Namespace: ns,
		}, metav1.CreateOptions{})
	}
	return err
}

func TestEnsureConfigMap_Creates(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	if err := ensureConfigMap(ctx, clientset, "default", "my-cm"); err != nil {
		t.Fatal(err)
	}

	actions := clientset.Actions()
	if len(actions) != 2 { // get (not found) + create
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[1].GetVerb() != "create" {
		t.Fatalf("expected create, got %s", actions[1].GetVerb())
	}
	slog.Info("TestEnsureConfigMap_Creates passed")
}

func TestEnsureConfigMap_AlreadyExists(t *testing.T) {
	existing := &corev1.ConfigMap{
		Name: "my-cm", Namespace: "default",
	}
	clientset := fake.NewSimpleClientset(existing)
	ctx := context.Background()

	if err := ensureConfigMap(ctx, clientset, "default", "my-cm"); err != nil {
		t.Fatal(err)
	}

	for _, a := range clientset.Actions() {
		if a.GetVerb() == "create" {
			t.Fatal("unexpected create — object already existed")
		}
	}
	slog.Info("TestEnsureConfigMap_AlreadyExists passed")
}

// main runs the tests directly so this is runnable with `go run ./...`
func main() {
	t := &testing.T{}
	TestEnsureConfigMap_Creates(t)
	TestEnsureConfigMap_AlreadyExists(t)
	_ = k8stesting.Action(nil) // import anchor
}
