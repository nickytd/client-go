// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 13: Typed CRD client (hand-written types, no controller-gen)
//
// Demonstrates:
//   - Defining Go structs that implement runtime.Object
//   - Registering types with a scheme
//   - Building a typed REST client for a CRD
//   - Typed CRUD without code-gen tooling
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Widget is a minimal hand-written CRD type.
type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WidgetSpec `json:"spec"`
}

type WidgetSpec struct {
	Color string `json:"color"`
	Size  int    `json:"size"`
}

type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Widget `json:"items"`
}

// DeepCopyObject returns a shallow copy, which is enough for this demo. Real
// controller-gen output deep-copies nested fields too — e.g. WidgetList.Items
// would be cloned rather than shared, so mutating one copy can't affect another.
func (w *Widget) DeepCopyObject() runtime.Object     { out := *w; return &out }
func (w *WidgetList) DeepCopyObject() runtime.Object { out := *w; return &out }

var (
	schemeGVK = schema.GroupVersion{Group: "example.com", Version: "v1alpha1"}
	scheme    = runtime.NewScheme()
	codecs    serializer.CodecFactory
)

func init() {
	scheme.AddKnownTypes(schemeGVK, &Widget{}, &WidgetList{})
	metav1.AddToGroupVersion(scheme, schemeGVK)
	codecs = serializer.NewCodecFactory(scheme)
}

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}

	crdConfig := *config
	crdConfig.GroupVersion = &schemeGVK
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = codecs.WithoutConversion()

	client, err := rest.RESTClientFor(&crdConfig)
	if err != nil {
		slog.Error("error creating REST client", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if err := run(ctx, client); err != nil {
		slog.Error("widget operations failed (is the CRD installed?)", "err", err)
		os.Exit(1)
	}
}

// run performs a typed Create → Get → List cycle against the Widget CRD,
// then deletes the created object so the program is re-runnable.
func run(ctx context.Context, client rest.Interface) error {
	const name = "my-widget"

	widget := &Widget{
		Kind: "Widget", APIVersion: schemeGVK.String(),
		Name:      name,
		Namespace: "default",
		Spec:      WidgetSpec{Color: "blue", Size: 3},
	}

	created := &Widget{}
	err := client.Post().
		Resource("widgets").
		Namespace("default").
		Body(widget).
		Do(ctx).
		Into(created)
	if err != nil {
		return err
	}
	slog.Info("created Widget", "name", created.Name, "color", created.Spec.Color)

	got := &Widget{}
	err = client.Get().
		Resource("widgets").
		Namespace("default").
		Name(name).
		Do(ctx).
		Into(got)
	if err != nil {
		return err
	}
	slog.Info("got Widget", "name", got.Name)

	list := &WidgetList{}
	err = client.Get().
		Resource("widgets").
		Namespace("default").
		Do(ctx).
		Into(list)
	if err != nil {
		return err
	}
	slog.Info("listed Widgets", "count", len(list.Items))

	// Clean up so the next run starts from a known state.
	err = client.Delete().
		Resource("widgets").
		Namespace("default").
		Name(name).
		Do(ctx).
		Error()
	if err != nil {
		return err
	}
	slog.Info("deleted Widget", "name", name)

	return nil
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
