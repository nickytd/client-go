# Case 06 — SharedInformer

## What this case covers

`cache.SharedInformer` wraps the raw list+watch loop (case 05) and provides a thread-safe in-memory store plus event callbacks. It is the building block for every Kubernetes controller.

## Key concepts

| Concept | Description |
|---|---|
| `cache.NewSharedInformer` | Creates a single-resource informer from a `ListWatch` |
| `AddEventHandler` | Registers `OnAdd`, `OnUpdate`, `OnDelete` callbacks |
| `ResourceEventHandlerFuncs` | Convenience struct implementing `ResourceEventHandler` |
| `WaitForCacheSync` | Blocks until the initial list has been replayed into the local store |
| `store.List()` | Returns all objects currently in the local cache |
| `store.GetByKey(ns/name)` | Retrieves one object from the cache by key |

## How to run

```bash
go run ./...
# Keep it running; create/delete ConfigMaps to observe callbacks
kubectl create configmap foo --from-literal=x=1
kubectl delete configmap foo
```

Expected output:

```
Cache synced. 2 ConfigMaps in store.
ADD    default/foo
DELETE default/foo
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | SharedInformer setup, event handler, cache sync wait, store query |

## Concepts in depth

### Informer lifecycle

```
Start informer goroutine
    │
    ▼
List all objects → replay as ADDED events → mark cache as synced
    │
    ▼
Watch for changes → fire ADD/UPDATE/DELETE handlers
    │
    └── reconnect on error (transparently)
```

Callers must call `WaitForCacheSync` before trusting the store — before sync, the store is empty even if objects exist in the cluster.

### Event handler gotcha: OnUpdate fires on re-list

When the informer reconnects and re-lists, it replays all objects as `OnUpdate` (not `OnAdd`) for objects that already existed. Reconcile logic must be idempotent.

### The store interface

```go
store := informer.GetStore()
store.List()                          // all objects
store.GetByKey("default/my-cm")     // one object by namespace/name
store.ListKeys()                     // all keys
```

The store is safe for concurrent reads from multiple goroutines.

### SharedInformer vs SharedIndexInformer

`NewSharedInformer` returns a `SharedInformer` with a basic key-value store.
`NewSharedIndexInformer` (used in case 08) adds custom indexes for efficient lookup by arbitrary fields.

### Why "Shared"?

Multiple consumers (e.g. two controllers) can call `AddEventHandler` on the same informer. The underlying list+watch runs only once, with events fanned out to all handlers — saves connections and memory compared to one informer per controller.

## References

- [SharedInformer godoc](https://pkg.go.dev/k8s.io/client-go/tools/cache#SharedInformer)
- [Writing Controllers — Kubernetes docs](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md)
- [client-go informer example](https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go)
