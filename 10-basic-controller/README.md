# Case 10 — Basic Controller Pattern

> **See also [`10-shared-informer-factory-controller`](../10-shared-informer-factory-controller)** for the
> same pattern built on a `SharedInformerFactory` reconciling two resource types (configmaps + nodes).

## What this case covers

This case assembles the pieces from cases 06–09 into a complete controller: an informer populates the work queue, a reconcile loop processes items idempotently, and graceful shutdown ties everything together with context cancellation.

## Key concepts

| Concept | Description |
|---|---|
| Enqueue on event | `OnAdd/OnUpdate/OnDelete` extract the key and call `queue.Add` |
| `cache.MetaNamespaceKeyFunc` | Standard key extractor: returns `"namespace/name"` |
| `cache.SplitMetaNamespaceKey` | Splits a key back into namespace and name |
| Idempotent reconcile | Always check desired vs actual state — never assume events are ordered |
| Graceful shutdown | `context.WithCancel` + `signal.NotifyContext` + `queue.ShutDown()` |

## How to run

```bash
go run ./...
# Ctrl+C for clean shutdown
# In another terminal, create/modify/delete ConfigMaps
kubectl create configmap test --from-literal=env=prod
kubectl delete configmap test
```

Expected output:

```
Controller started. Watching ConfigMaps in 'default'.
Reconcile: default/kube-root-ca.crt
Reconcile: default/test
Reconcile: default/test (deleted)
Shutting down...
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Full controller: informer + queue + reconcile loop + graceful shutdown |

## Concepts in depth

### Controller structure

```
Informer (runs in background)
   │ OnAdd/OnUpdate/OnDelete
   ▼
Work Queue (deduplicates, rate-limits)
   │ Get
   ▼
Reconcile Loop (goroutine pool)
   │ Done + Forget / AddRateLimited
   └──────────────────────────────┘
```

### The reconcile function

```go
func (c *Controller) reconcile(key string) error {
    ns, name, _ := cache.SplitMetaNamespaceKey(key)

    // Get from local cache — not the API server
    obj, exists, err := c.store.GetByKey(key)
    if err != nil {
        return err
    }
    if !exists {
        // Object was deleted — clean up any dependent resources
        return nil
    }
    cm := obj.(*corev1.ConfigMap)
    // Reconcile desired state...
    _ = cm
    return nil
}
```

Key principle: **always read from the cache, never from the API server** inside a reconcile. The cache is local and fast; the API server adds latency and load.

### Graceful shutdown sequence

```go
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
defer cancel()

go informer.Run(ctx.Done())
cache.WaitForCacheSync(ctx.Done(), informer.HasSynced)

go c.runWorker()  // calls queue.Get in a loop

<-ctx.Done()
queue.ShutDown()  // unblocks any waiting Get() calls
```

### Why read from cache, not API server?

The reconcile function can be called thousands of times per second. Hitting the API server each time would overload etcd and add 10–100ms latency per reconcile. The informer cache is a local copy kept in sync by the watch stream, making reads O(1) in-process.

### Expanding to production

This single-threaded reconciler handles one item at a time. Real controllers run multiple worker goroutines:

```go
for i := 0; i < workers; i++ {
    go c.runWorker()
}
```

Items for different objects can be processed concurrently; items for the same object are serialized by the queue's deduplication.

## References

- [Writing Controllers — Kubernetes community](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md)
- [client-go workqueue example](https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go)
- [cache.MetaNamespaceKeyFunc godoc](https://pkg.go.dev/k8s.io/client-go/tools/cache#MetaNamespaceKeyFunc)
