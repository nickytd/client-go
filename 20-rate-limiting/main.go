// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 20: Client-side rate limiting
//
// Demonstrates:
//   - rest.Config QPS and Burst fields
//   - flowcontrol.NewTokenBucketRateLimiter
//   - Measuring actual throughput vs configured limits
//   - Tuning for high-throughput controllers vs polite clients
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/homedir"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}

	// Conservative client: 5 req/s, burst 10
	config.QPS = 5
	config.Burst = 10

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("error creating clientset", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Demonstrate a standalone manual rate limiter (independent of the
	// client's QPS/Burst): a token bucket allowing 20 req/s with a burst of 40.
	manual := flowcontrol.NewTokenBucketRateLimiter(20, 40)
	slog.Info("manual token-bucket limiter", "qps", 20, "burst", 40, "accepts_now", manual.TryAccept())

	var count atomic.Int64
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(start).Seconds()
			slog.Info("rate limit test complete",
				"requests", count.Load(),
				"elapsed_s", elapsed,
				"req_per_s", float64(count.Load())/elapsed,
			)
			return
		default:
			_, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}
			count.Add(1)
		}
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
