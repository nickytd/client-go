# Case 10 (factory variant) — Controller with a SharedInformerFactory

> **Companion to [`10-basic-controller`](../10-basic-controller).** That case builds a single
> hand-rolled `SharedIndexInformer` (one resource, configmaps). This variant swaps in a
> `SharedInformerFactory` and reconciles **two** resource types — compare the two side by side.

## What this case covers

This case assembles the pieces from cases 06–09 into a complete controller driven by a **`SharedInformerFactory`**. The factory manages typed informers for **two** resource types — namespaced **ConfigMaps** and cluster-scoped **Nodes** — and both feed a **single workqueue**. One reconcile loop pulls items off the queue and dispatches on kind, reading from each resource's typed lister.

## Key concepts

| Concept | Description |
|---|---|
| `informers.NewSharedInformerFactory` | One factory owns and syncs many typed informers; no hand-rolled `ListWatch` |
| Typed listers | `factory.Core().V1().ConfigMaps().Lister()` / `.Nodes().Lister()` read from the shared cache |
| Typed queue key | A single `resourceKey{kind, namespace, name}` lets one queue carry multiple kinds without string parsing |
| Dispatch on kind | `reconcile` switches on `key.kind` and calls the matching per-resource reconciler |
| Cluster-scoped vs namespaced | Nodes have an empty namespace; ConfigMaps are looked up as `Lister().ConfigMaps(ns).Get(name)` |
| Tombstone handling | `cache.DeletionHandlingMetaNamespaceKeyFunc` tolerates `DeletedFinalStateUnknown` on missed deletes |
| Idempotent reconcile | Read current state from cache; a not-found means the object was deleted |
| Graceful shutdown | `signal.NotifyContext` + `factory.Start(ctx.Done())` + `queue.ShutDown()` |

## How to run

```bash
go run ./...
# Ctrl+C for clean shutdown
# In another terminal, create/modify/delete ConfigMaps:
kubectl create configmap test --from-literal=env=prod
kubectl delete configmap test
```

Expected output (nodes reconcile on startup, ConfigMaps as you change them):

```
level=INFO msg="controller started, watching configmaps and nodes" namespace=default
level=INFO msg="reconcile node" name=minikube ready=true schedulable=true
level=INFO msg="reconcile configmap" namespace=default name=kube-root-ca.crt dataKeys=1
level=INFO msg="reconcile configmap" namespace=default name=test dataKeys=1
level=INFO msg="configmap deleted" namespace=default name=test
level=INFO msg="shutting down"
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Full controller: factory + two informers + single typed queue + dispatching reconcile loop + graceful shutdown |

## Concepts in depth

### Controller structure

```
SharedInformerFactory
 ├─ ConfigMaps().Informer()  ─┐  OnAdd/Update/Delete
 └─ Nodes().Informer()       ─┘        │ enqueue(kind, obj)
                                       ▼
             workqueue.TypedRateLimitingInterface[resourceKey]
                                       │ Get
                                       ▼
                      reconcile(ctx, key)  ── switch key.kind
                        ├─ configmap → configMapLister.ConfigMaps(ns).Get(name)
                        └─ node      → nodeLister.Get(name)
                                       │ Done + Forget / AddRateLimited
                                       └───────────────────────────────┘
```

### Why a factory instead of a hand-rolled informer?

The previous version built a single `SharedIndexInformer` from a `NewListWatchFromClient`. That works for one resource, but each additional kind means another `ListWatch`, another informer, and another `Run`/`HasSynced` to coordinate. The `SharedInformerFactory`:

- Builds **typed** informers (`Core().V1().Nodes()`) with their listers for free.
- **Shares** a single watch connection and cache per resource type across all consumers.
- Starts and syncs every registered informer with one `Start` / `WaitForCacheSync`.

### One queue, a typed key

Because both informers feed the same queue, the key must say *which* kind it refers to. A small struct keeps the dispatch type-safe:

```go
type resourceKey struct {
    kind      string // "configmap" | "node"
    namespace string // empty for cluster-scoped resources (Nodes)
    name      string
}
```

The reconcile loop then switches on `key.kind` and reads from the matching lister — no stringly-typed prefix parsing, and the compiler keeps the kinds honest.

### Reconcile reads from the cache

```go
cm, err := c.configMapLister.ConfigMaps(key.namespace).Get(key.name)
if apierrors.IsNotFound(err) {
    // Object was deleted — clean up any dependent resources.
    return nil
}
```

Listers read from the informer's local cache, not the API server. The reconcile can be called thousands of times per second; hitting etcd each time would add 10–100 ms of latency per call and overload the server. The cache is a local copy kept current by the watch stream, so reads are O(1) in-process.

> **Note:** these reconcilers are read-only — they log observed state and return. Later cases (11+) mutate state and set owner references.

### Graceful shutdown sequence

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()

c.factory.Start(ctx.Done())             // launch every registered informer
c.factory.WaitForCacheSync(ctx.Done())  // block until all caches are warm

go func() {
    <-ctx.Done()
    c.queue.ShutDown()                  // unblocks the Get() in processNext
}()

for c.processNext(ctx) { }              // worker loop exits when Get() returns quit
```

### Expanding to production

This single-threaded reconciler handles one item at a time. Real controllers run multiple worker goroutines:

```go
for range workers {
    go func() { for c.processNext(ctx) {} }()
}
```

Items for different objects run concurrently; items for the same object are serialized by the queue's deduplication.

## References

- [Writing Controllers — Kubernetes community](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md)
- [client-go SharedInformerFactory godoc](https://pkg.go.dev/k8s.io/client-go/informers#NewSharedInformerFactory)
- [client-go workqueue example](https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go)
- [cache.DeletionHandlingMetaNamespaceKeyFunc godoc](https://pkg.go.dev/k8s.io/client-go/tools/cache#DeletionHandlingMetaNamespaceKeyFunc)
