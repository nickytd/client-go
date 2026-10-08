// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 22: Retry on conflict
//
// Demonstrates optimistic concurrency. An Update carries the object's
// resourceVersion; if the stored version advanced since we read it, the API
// server rejects the write with a Conflict (409). retry.RetryOnConflict wraps
// a re-GET + re-mutate loop that converges under contention.
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}
	// The default client limiter (QPS=5, Burst=10) throttles this demo: 10 workers
	// each run a GET+PUT retry loop, far exceeding 5 req/s. The resulting ~1s delays
	// per request both spam "client-side throttling" logs and widen the race window.
	// Raise the limits so the concept under test — conflict/retry — is what you
	// observe, not the limiter. See case 20 (client-side rate limiting) for this knob.
	config.QPS = 50
	config.Burst = 100

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("error creating clientset", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()
	cms := clientset.CoreV1().ConfigMaps("default")

	// Seed an object with a numeric counter in its data.
	seed := &corev1.ConfigMap{
		Name: "retry-demo",
		Data: map[string]string{"counter": "0"},
	}
	_ = cms.Delete(ctx, "retry-demo", metav1.DeleteOptions{})
	if _, err := cms.Create(ctx, seed, metav1.CreateOptions{}); err != nil {
		slog.Error("error creating configmap", "err", err)
		os.Exit(1)
	}

	// Fire N concurrent incrementers. Without RetryOnConflict most would fail
	// with a 409; with it, every increment eventually lands.
	const workers = 10
	var wg sync.WaitGroup
	var conflicts int
	var mu sync.Mutex

	// Add Jitter to de-synchronize the workers.
	backoff := wait.Backoff{
		Steps:    20,
		Duration: 10 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0.5,
	}

	for i := range workers {
		wg.Go(func() {
			err := retry.RetryOnConflict(backoff, func() error {
				cur, err := cms.Get(ctx, "retry-demo", metav1.GetOptions{})
				if err != nil {
					return err
				}
				n, _ := strconv.Atoi(cur.Data["counter"])
				cur.Data["counter"] = strconv.Itoa(n + 1)
				_, err = cms.Update(ctx, cur, metav1.UpdateOptions{})
				if apierrors.IsConflict(err) {
					mu.Lock()
					conflicts++
					mu.Unlock()
				}
				return err
			})
			if err != nil {
				slog.Error("error updating configmap in worker", "worker", i, "err", err)
			}
		})
	}
	wg.Wait()

	final, err := cms.Get(ctx, "retry-demo", metav1.GetOptions{})
	if err != nil {
		slog.Error("error getting final configmap", "err", err)
		os.Exit(1)
	}
	slog.Info("done",
		"counter", final.Data["counter"], "want", workers, "conflictsRetried", conflicts)

	_ = cms.Delete(ctx, "retry-demo", metav1.DeleteOptions{})
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
