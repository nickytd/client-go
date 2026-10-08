// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 04: Discovery client
//
// Demonstrates:
//   - clientset.Discovery() — list all API groups and resources
//   - ServerVersion
//   - restmapper.GetAPIGroupResources — map GVK to GVR at runtime
package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	clientset := mustClientset()

	disc := clientset.Discovery()

	v, _ := disc.ServerVersion()
	slog.Info("server version", "version", v.GitVersion)

	groups, err := disc.ServerGroups()
	if err != nil {
		slog.Error("error fetching API groups", "err", err)
		os.Exit(1)
	}
	slog.Info("API groups", "count", len(groups.Groups))
	for _, g := range groups.Groups {
		slog.Info("group", "name", g.Name, "preferred", g.PreferredVersion.Version)
	}

	gr, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		slog.Error("error getting API group resources", "err", err)
		os.Exit(1)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(gr)

	gvk := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		slog.Error("error mapping GVK", "err", err)
		os.Exit(1)
	}
	slog.Info("Deployment GVK→GVR", "resource", mapping.Resource)

	gvk = schema.GroupVersionKind{Group: "metrics.k8s.io", Version: "v1beta1", Kind: "PodMetrics"}
	mapping, err = mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		slog.Error("error mapping GVK", "err", err)
		os.Exit(1)
	}
	slog.Info("Pod Metric GVK→GVR", "resource", mapping.Resource)

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
