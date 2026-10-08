// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 10: Basic controller pattern
//
// Demonstrates:
//   - Combining SharedInformer + workqueue into a reconcile loop
//   - Enqueue key on Add/Update/Delete events
//   - Idempotent reconcile: check desired vs actual state
//   - Graceful shutdown with context cancellation
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	clientset kubernetes.Interface
	informer  cache.SharedIndexInformer
	queue     workqueue.TypedRateLimitingInterface[string]
}

func NewController(clientset kubernetes.Interface) *Controller {
	lw := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "configmaps", "default", fields.Everything())
	informer := cache.NewSharedIndexInformer(lw, &corev1.ConfigMap{}, 30*time.Second, cache.Indexers{})
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())

	c := &Controller{clientset: clientset, informer: informer, queue: queue}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.enqueue(obj) },
		UpdateFunc: func(_, obj any) { c.enqueue(obj) },
		DeleteFunc: func(obj any) { c.enqueue(obj) },
	})
	return c
}

func (c *Controller) enqueue(obj any) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}

func (c *Controller) Run(ctx context.Context) {
	defer c.queue.ShutDown()
	go c.informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		slog.Error("cache sync timed out")
		os.Exit(1)
	}
	slog.Info("controller started, watching configmaps", "namespace", "default")

	// Shut the queue down on cancellation so the blocked Get() in
	// processNext unblocks and the worker loop can exit cleanly.
	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		c.queue.ShutDown()
	}()

	for c.processNext(ctx) {
	}
}

func (c *Controller) processNext(ctx context.Context) bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	if err := c.reconcile(ctx, key); err != nil {
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}

func (c *Controller) reconcile(_ context.Context, key string) error {
	slog.Info("reconcile", "key", key)
	return nil
}

func main() {
	clientset := mustClientset()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	NewController(clientset).Run(ctx)
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
