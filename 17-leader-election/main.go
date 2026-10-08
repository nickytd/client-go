// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 17: Leader election
//
// Demonstrates:
//   - leaderelection.RunOrDie — distributed leader election via Lease
//   - LeaderElectionConfig: LeaseDuration, RenewDeadline, RetryPeriod
//   - OnStartedLeading / OnStoppedLeading / OnNewLeader callbacks
//   - Safe multi-replica controller pattern
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/homedir"
)

func main() {
	clientset := mustClientset()

	// Each candidate must present a UNIQUE identity, or leader election can't
	// tell replicas apart. The README runs two instances on one host, so prefer
	// REPLICA_ID and fall back to the hostname only when it isn't set.
	id := os.Getenv("REPLICA_ID")
	if id == "" {
		id, _ = os.Hostname()
	}
	ctx := context.Background()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "client-go-lab-leader",
			Namespace: "default",
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   5 * time.Second,
		RenewDeadline:   2 * time.Second,
		RetryPeriod:     1 * time.Second,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				slog.Info("I am the leader — doing work", "id", id)
				time.Sleep(30 * time.Second)
			},
			OnStoppedLeading: func() {
				slog.Info("lost leadership — stopping", "id", id)
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				if identity != id {
					slog.Info("new leader elected", "id", id, "leader", identity)
				}
			},
		},
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
