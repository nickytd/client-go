// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 10 (factory variant): Controller driven by a SharedInformerFactory
//
// Demonstrates:
//   - SharedInformerFactory driving typed informers for two resource types
//   - A single workqueue keyed by a typed struct (kind + namespace + name)
//   - One reconcile loop dispatching on kind, reading from typed listers
//   - Idempotent reconcile: check desired vs actual state from cache
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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
)

// resourceKey identifies a single object to reconcile. A typed key lets one
// queue carry multiple resource kinds without stringly-typed parsing: the
// reconcile loop switches on kind and reads from the matching lister.
// namespace is empty for cluster-scoped resources such as Nodes.
type resourceKey struct {
	kind      string
	namespace string
	name      string
}

const (
	kindConfigMap = "configmap"
	kindNode      = "node"
)

type Controller struct {
	factory         informers.SharedInformerFactory
	configMapLister corelisters.ConfigMapLister
	nodeLister      corelisters.NodeLister
	queue           workqueue.TypedRateLimitingInterface[resourceKey]
}

func NewController(clientset kubernetes.Interface) *Controller {
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

	configMaps := factory.Core().V1().ConfigMaps()
	nodes := factory.Core().V1().Nodes()

	c := &Controller{
		factory:         factory,
		configMapLister: configMaps.Lister(),
		nodeLister:      nodes.Lister(),
		queue:           workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[resourceKey]()),
	}

	// AddEventHandler returns a registration handle and an error; we don't
	// deregister handlers in this example, so both are intentionally ignored.
	_, _ = configMaps.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.enqueue(kindConfigMap, obj) },
		UpdateFunc: func(_, obj any) { c.enqueue(kindConfigMap, obj) },
		DeleteFunc: func(obj any) { c.enqueue(kindConfigMap, obj) },
	})
	_, _ = nodes.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.enqueue(kindNode, obj) },
		UpdateFunc: func(_, obj any) { c.enqueue(kindNode, obj) },
		DeleteFunc: func(obj any) { c.enqueue(kindNode, obj) },
	})
	return c
}

func (c *Controller) enqueue(kind string, obj any) {
	namespace, name, err := splitKey(obj)
	if err != nil {
		slog.Error("could not derive key", "kind", kind, "err", err)
		return
	}
	c.queue.Add(resourceKey{kind: kind, namespace: namespace, name: name})
}

// splitKey extracts namespace/name from an object, tolerating the
// DeletedFinalStateUnknown tombstone the informer delivers on missed deletes.
func splitKey(obj any) (namespace, name string, err error) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return "", "", err
	}
	namespace, name, err = cache.SplitMetaNamespaceKey(key)
	return namespace, name, err
}

func (c *Controller) Run(ctx context.Context) {
	defer c.queue.ShutDown()

	c.factory.Start(ctx.Done())
	for typ, ok := range c.factory.WaitForCacheSync(ctx.Done()) {
		if !ok {
			slog.Error("cache sync failed", "type", typ)
			os.Exit(1)
		}
	}
	slog.Info("controller started, watching configmaps and nodes", "namespace", "default")

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

func (c *Controller) reconcile(_ context.Context, key resourceKey) error {
	switch key.kind {
	case kindConfigMap:
		return c.reconcileConfigMap(key)
	case kindNode:
		return c.reconcileNode(key)
	default:
		slog.Warn("unknown kind, dropping", "kind", key.kind)
		return nil
	}
}

func (c *Controller) reconcileConfigMap(key resourceKey) error {
	cm, err := c.configMapLister.ConfigMaps(key.namespace).Get(key.name)
	if apierrors.IsNotFound(err) {
		slog.Info("configmap deleted", "namespace", key.namespace, "name", key.name)
		return nil
	}
	if err != nil {
		return err
	}
	// Read from cache; never log the data values, only metadata.
	slog.Info("reconcile configmap",
		"namespace", cm.Namespace, "name", cm.Name, "dataKeys", len(cm.Data))
	return nil
}

func (c *Controller) reconcileNode(key resourceKey) error {
	node, err := c.nodeLister.Get(key.name)
	if apierrors.IsNotFound(err) {
		slog.Info("node deleted", "name", key.name)
		return nil
	}
	if err != nil {
		return err
	}
	slog.Info("reconcile node",
		"name", node.Name,
		"ready", nodeReady(node),
		"schedulable", !node.Spec.Unschedulable)
	return nil
}

func nodeReady(node *corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
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
