// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

// TestReactorInjectsError shows PrependReactor forcing Get to fail, so you can
// exercise error paths that are hard to trigger against a real API server.
func TestReactorInjectsError(t *testing.T) {
	cs := fake.NewSimpleClientset(&corev1.ConfigMap{
		Name: "cm", Namespace: "default",
	})

	wantErr := errors.New("boom")
	cs.PrependReactor("get", "configmaps",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			// handled=true short-circuits the default tracker.
			return true, nil, wantErr
		})

	_, err := cs.CoreV1().ConfigMaps("default").Get(context.Background(), "cm", metav1.GetOptions{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("got err=%v, want %v", err, wantErr)
	}
}

// TestReactorCountsCalls shows a reactor that observes actions while delegating
// behavior back to the default tracker (handled=false).
func TestReactorCountsCalls(t *testing.T) {
	cs := fake.NewSimpleClientset()
	var creates int
	cs.PrependReactor("create", "configmaps",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			creates++
			return false, nil, nil // fall through to the real fake behavior
		})

	_, err := cs.CoreV1().ConfigMaps("default").Create(context.Background(),
		&corev1.ConfigMap{Name: "x"}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if creates != 1 {
		t.Fatalf("reactor saw %d creates, want 1", creates)
	}

	got, err := cs.CoreV1().ConfigMaps("default").Get(context.Background(), "x", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get after fall-through create: %v", err)
	}
	if got.Name != "x" {
		t.Fatalf("stored object name = %q, want x", got.Name)
	}
}

// TestDynamicFake shows the dynamic fake client operating on an arbitrary GVR
// without generated types — the testing counterpart to the dynamic client.
func TestDynamicFake(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	dc := dynamicfake.NewSimpleDynamicClient(scheme)

	obj := &unstructuredConfigMap{}
	created, err := dc.Resource(gvr).Namespace("default").
		Create(context.Background(), obj.toUnstructured(), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("dynamic create: %v", err)
	}
	if created.GetName() != "dyn" {
		t.Fatalf("got name %q, want dyn", created.GetName())
	}

	got, err := dc.Resource(gvr).Namespace("default").
		Get(context.Background(), "dyn", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("dynamic get: %v", err)
	}
	if got.GetName() != "dyn" {
		t.Fatalf("round-trip name %q, want dyn", got.GetName())
	}

	// Assert the dynamic client is the interface we expect (compile-time check).
	var _ dynamic.Interface = dc
}

type unstructuredConfigMap struct{}

func (unstructuredConfigMap) toUnstructured() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": "dyn", "namespace": "default"},
		"data":       map[string]any{"k": "v"},
	}}
}
