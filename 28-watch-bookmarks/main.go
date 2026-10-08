// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 28: Watch bookmarks
//
// A long watch can be disrupted; to resume you must know a resourceVersion the
// server still remembers. With AllowWatchBookmarks the server periodically
// sends a Bookmark event carrying only an up-to-date resourceVersion (no
// object payload). Persisting it lets a reconnect start from a fresh point and
// avoid "resourceVersion too old" (410 Gone).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	clientset := mustClientset()
	ctx := context.Background()
	cms := clientset.CoreV1().ConfigMaps("default")

	// Establish a starting resourceVersion from a List.
	initial, err := cms.List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Error("error listing configmaps", "err", err)
		os.Exit(1)
	}
	rv := initial.ResourceVersion
	slog.Info("watching", "resourceVersion", rv, "bookmarks", true)
	slog.Info("watching (Ctrl-C to stop)")

	w, err := cms.Watch(ctx, metav1.ListOptions{
		ResourceVersion:     rv,
		AllowWatchBookmarks: true,
	})
	if err != nil {
		slog.Error("error starting watch", "err", err)
		os.Exit(1)
	}
	defer w.Stop()

	// Generate some churn so the server has events + a reason to bookmark.
	go func() {
		for i := range 3 {
			name := fmt.Sprintf("bookmark-demo-%d", i)
			_, _ = cms.Create(ctx, &corev1.ConfigMap{
				Name: name,
			}, metav1.CreateOptions{})
			time.Sleep(500 * time.Millisecond)
			_ = cms.Delete(ctx, name, metav1.DeleteOptions{})
		}
		slog.Info("churn done; idling until a bookmark arrives (Ctrl-C to stop)")
	}()

	// Each read blocks until the server sends the next event. The watch streams
	// over a long-lived HTTP connection; Ctrl-C terminates the process.
	var lastBookmark string
	for event := range w.ResultChan() {
		obj, _ := event.Object.(*corev1.ConfigMap)
		switch event.Type {
		case watch.Bookmark:
			// Bookmark objects carry only metadata.resourceVersion.
			lastBookmark = obj.ResourceVersion
			slog.Info("bookmark (safe resume point)", "resourceVersion", lastBookmark)
		case watch.Added, watch.Modified, watch.Deleted:
			slog.Info("event", "type", event.Type, "name", obj.Name, "resourceVersion", obj.ResourceVersion)
		}
	}
	slog.Info("watch closed", "lastBookmark", lastBookmark)
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
