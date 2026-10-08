# Case 08 — Indexers & Custom Store Queries

## What this case covers

Informers store objects in a flat key-value map. Indexers add secondary indexes — like a database index — so you can efficiently look up objects by arbitrary fields (e.g. "all Pods on node X") without scanning the entire store.

## Key concepts

| Concept | Description |
|---|---|
| `cache.Indexers` | `map[string]IndexFunc` — name → function that returns index keys for an object |
| `cache.IndexFunc` | `func(obj interface{}) ([]string, error)` — extracts index key(s) from an object |
| `cache.MetaNamespaceIndexFunc` | Built-in indexer: indexes objects by namespace |
| `cache.NewInformerWithOptions` | Creates an informer from a `cache.InformerOptions` struct, with custom indexers attached |
| `store.ByIndex(indexName, key)` | Returns all objects matching a given index key |

## How to run

```bash
go run ./...
```

Expected output:

```
Cache synced. Pods indexed by node.
Pods on node kind-worker: my-pod-abc coredns-xyz
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Custom index by node name, query via `ByIndex` |

## Concepts in depth

### How indexes work

An `IndexFunc` is called for every object added to the store. It returns one or more string keys. The store builds an inverted map: `indexName → key → set of object keys`.

```go
// Index Pods by the node they run on
byNodeIndex := func(obj interface{}) ([]string, error) {
    pod, ok := obj.(*corev1.Pod)
    if !ok {
        return nil, nil
    }
    return []string{pod.Spec.NodeName}, nil
}

indexers := cache.Indexers{
    cache.NamespaceIndex: cache.MetaNamespaceIndexFunc, // built-in
    "byNode":             byNodeIndex,                  // custom
}
```

### Querying by index

```go
// All Pods on a specific node
objs, err := indexer.ByIndex("byNode", "worker-node-1")
for _, obj := range objs {
    pod := obj.(*corev1.Pod)
    fmt.Println(pod.Name)
}
```

`ByIndex` is O(1) for the lookup — the store maintains the inverted index incrementally as objects are added/updated/deleted.

### Multi-value indexes

An `IndexFunc` can return multiple keys, placing the object in multiple index buckets:

```go
// Index ConfigMap by each label value
byLabelValue := func(obj interface{}) ([]string, error) {
    cm := obj.(*corev1.ConfigMap)
    var vals []string
    for _, v := range cm.Labels {
        vals = append(vals, v)
    }
    return vals, nil
}
```

### cache.NewInformerWithOptions signature

```go
store, controller := cache.NewInformerWithOptions(cache.InformerOptions{
    ListerWatcher: listWatch,
    ObjectType:    &corev1.Pod{},
    Handler:       eventHandlerFuncs,
    ResyncPeriod:  resyncPeriod,
    Indexers:      indexers, // cache.Indexers map — NOT cache.NewIndexer(...)
})

// NewInformerWithOptions returns a cache.Store. When Indexers are supplied it
// is backed by an Indexer, so assert to reach ByIndex/ListIndexFuncValues.
indexer := store.(cache.Indexer)
```

`Indexers` is a `cache.Indexers` map directly, not the `cache.Indexer` interface returned by `cache.NewIndexer`.

> `cache.NewIndexerInformer` did the same thing with positional arguments and
> returned a `cache.Indexer` directly, but it is deprecated in favor of
> `NewInformerWithOptions`.

### controller-runtime equivalent

`controller-runtime` exposes this as `mgr.GetFieldIndexer().IndexField(ctx, obj, field, fn)` — the same concept with a cleaner API (see case 13 in the companion repo).

## References

- [cache.Indexers godoc](https://pkg.go.dev/k8s.io/client-go/tools/cache#Indexers)
- [cache.NewInformerWithOptions godoc](https://pkg.go.dev/k8s.io/client-go/tools/cache#NewInformerWithOptions)
- [Writing Controllers — indexers section](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md)
