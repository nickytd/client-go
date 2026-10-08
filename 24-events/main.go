// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 24: Event recording
//
// Controllers surface decisions as Kubernetes Events (what you see under the
// Events: section of `kubectl describe`). This wires up the modern events/v1
// recorder:
//
//	broadcaster -> sink (writes events.k8s.io/v1 Events to the API) + logger
//	recorder.Eventf(regarding, related, type, reason, action, note, args...)
//
// Delivery is asynchronous and best-effort: recorder.Eventf enqueues each event
// from a detached goroutine, and broadcaster.Shutdown only drains the in-memory
// queue — it does not guarantee an event already handed to Eventf has reached
// the API.
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

func main() {
	clientset := mustClientset()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the object we will attach events to.
	cm, err := clientset.CoreV1().ConfigMaps("default").Create(ctx, &corev1.ConfigMap{
		Name: "event-demo",
	}, metav1.CreateOptions{})
	if err != nil {
		slog.Error("error creating configmap", "err", err)
		os.Exit(1)
	}

	// Build the broadcaster: the sink writes Events (events.k8s.io/v1) back to
	// the API server, and a second watcher logs them locally so you can see
	// them without kubectl.
	broadcaster := events.NewBroadcaster(&events.EventSinkImpl{
		Interface: clientset.EventsV1(),
	})
	if err := broadcaster.StartRecordingToSinkWithContext(ctx); err != nil {
		slog.Error("error starting event sink", "err", err)
		os.Exit(1)
	}
	if _, err := broadcaster.StartLogging(klog.Background()); err != nil {
		slog.Error("error starting event logging", "err", err)
		os.Exit(1)
	}
	recorder := broadcaster.NewRecorder(scheme.Scheme, "client-go-lab")

	// Normal and Warning events, exactly as a reconciler would emit them. The
	// events/v1 signature adds a `related` object (nil here) and an `action`
	// verb describing what the controller did.
	wantReasons := sets.New("Reconciled", "Degraded")
	recorder.Eventf(cm, nil, corev1.EventTypeNormal, "Reconciled", "Reconcile", "processed ConfigMap %s", cm.Name)
	recorder.Eventf(cm, nil, corev1.EventTypeWarning, "Degraded", "Reconcile", "example warning at %s", time.Now().Format(time.Kitchen))

	// Delivery is async, so wait until the API confirms both events exist rather
	// than guessing a flush delay.
	if err := waitForEvents(ctx, clientset, cm, wantReasons); err != nil {
		slog.Error("events not confirmed in API", "err", err)
		os.Exit(1)
	}

	// Both events are durably recorded; stop the broadcaster's background
	// goroutines before exiting.
	broadcaster.Shutdown()
	slog.Info("emitted events", "object", "default/"+cm.Name,
		"inspect", "kubectl describe configmap "+cm.Name)

	_ = clientset.CoreV1().ConfigMaps("default").Delete(ctx, "event-demo", metav1.DeleteOptions{})
}

// waitForEvents polls the events.k8s.io/v1 API until every reason in want has
// been recorded against cm, or the context times out.
func waitForEvents(ctx context.Context, clientset kubernetes.Interface, cm *corev1.ConfigMap, want sets.Set[string]) error {
	return wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 10*time.Second, true,
		func(ctx context.Context) (bool, error) {
			list, err := clientset.EventsV1().Events(cm.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			seen := sets.New[string]()
			for _, ev := range list.Items {
				if ev.Regarding.Name == cm.Name && ev.Regarding.UID == cm.UID {
					seen.Insert(ev.Reason)
				}
			}
			return seen.IsSuperset(want), nil
		})
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
