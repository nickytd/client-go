// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 12: CRD via dynamic client (no code-gen)
//
// Demonstrates:
//   - Installing a CRD programmatically via the apiextensions client
//   - Waiting for the CRD's Established condition before use
//   - CRUD on custom resources using the dynamic client
//   - No generated types required — pure unstructured.Unstructured
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	crdName   = "widgets.example.com"
	namespace = "default"
)

var widgetGVR = schema.GroupVersionResource{
	Group:    "example.com",
	Version:  "v1",
	Resource: "widgets",
}

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}

	apiextClient, err := apiextclientset.NewForConfig(config)
	if err != nil {
		slog.Error("error creating apiextensions client", "err", err)
		os.Exit(1)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		slog.Error("error creating dynamic client", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if err := installCRD(ctx, apiextClient); err != nil {
		slog.Error("error installing CRD", "err", err)
		os.Exit(1)
	}
	slog.Info("CRD created", "name", crdName)

	if err := crudWidget(ctx, dynClient); err != nil {
		slog.Error("error during widget CRUD", "err", err)
		os.Exit(1)
	}
}

// installCRD registers the Widget CRD and blocks until the API server
// reports it as Established (ready to serve). Creation is idempotent: an
// already-existing CRD is treated as success.
func installCRD(ctx context.Context, client apiextclientset.Interface) error {
	crd := &apiextensionsv1.CustomResourceDefinition{
		Name: crdName,
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:     "widgets",
				Singular:   "widget",
				Kind:       "Widget",
				ShortNames: []string{"wd"},
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    "v1",
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"spec": {
								Type: "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{
									"color": {Type: "string"},
									"size":  {Type: "integer"},
								},
							},
						},
					},
				},
			}},
		},
	}

	_, err := client.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return waitForEstablished(ctx, client, crdName)
}

// waitForEstablished polls the CRD until its Established condition is true.
// CRD registration is asynchronous: the custom resource endpoint is not
// usable until the API server finishes wiring it up.
func waitForEstablished(ctx context.Context, client apiextclientset.Interface, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		crd, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for _, cond := range crd.Status.Conditions {
			if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return errors.New("timed out waiting for CRD to become established")
		case <-ticker.C:
		}
	}
}

// crudWidget exercises the full lifecycle of a custom resource through the
// dynamic client: create, get, update, delete — all via unstructured data.
func crudWidget(ctx context.Context, dynClient dynamic.Interface) error {
	widgets := dynClient.Resource(widgetGVR).Namespace(namespace)

	widget := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "example.com/v1",
			"kind":       "Widget",
			"metadata":   map[string]any{"name": "my-widget", "namespace": namespace},
			"spec":       map[string]any{"color": "blue", "size": int64(3)},
		},
	}

	created, err := widgets.Create(ctx, widget, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	slog.Info("created widget", "name", created.GetName())

	got, err := widgets.Get(ctx, "my-widget", metav1.GetOptions{})
	if err != nil {
		return err
	}
	color, _, _ := unstructured.NestedString(got.Object, "spec", "color")
	slog.Info("got widget", "name", got.GetName(), "color", color)

	// Update the color field in place and push it back.
	if err := unstructured.SetNestedField(got.Object, "red", "spec", "color"); err != nil {
		return err
	}
	updated, err := widgets.Update(ctx, got, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	color, _, _ = unstructured.NestedString(updated.Object, "spec", "color")
	slog.Info("updated widget", "name", updated.GetName(), "color", color)

	if err := widgets.Delete(ctx, "my-widget", metav1.DeleteOptions{}); err != nil {
		return err
	}
	slog.Info("deleted widget", "name", "my-widget")

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
