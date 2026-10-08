# Case 07 — SharedInformerFactory

## What this case covers

Real controllers watch multiple resource types. `informers.NewSharedInformerFactory` manages a pool of informers — one per resource type — started and synced together. It also provides typed listers for ergonomic, read-only cache access.

## Key concepts

| Concept | Description |
|---|---|
| `informers.NewSharedInformerFactory` | Creates a factory scoped to a namespace (or all namespaces) |
| `factory.Core().V1().Pods()` | Returns a typed `PodInformer` (wraps the underlying SharedIndexInformer) |
| `factory.Apps().V1().Deployments()` | Typed `DeploymentInformer` |
| `podInformer.Lister()` | Returns a `PodNamespaceLister` — read-only cache interface |
| `podLister.Pods(ns).Get(name)` | Retrieves one Pod from the local cache by name |
| `factory.Start(stopCh)` | Starts all registered informers' goroutines |
| `factory.WaitForCacheSync(stopCh)` | Blocks until all started informers have synced |

## How to run

```bash
go run ./...
```

Expected output:

```
All caches synced.
Pods in kube-system: coredns-... etcd-... kube-apiserver-...
Deployments in default: (none or your deployments)
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Factory setup, typed informers, lister-based cache queries |

## Concepts in depth

### Factory pattern

```go
factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

// Register informers (lazy — goroutines start only on factory.Start)
podInformer := factory.Core().V1().Pods()
deployInformer := factory.Apps().V1().Deployments()

// Wire up event handlers before starting
podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{...})

// Start all informers and wait for sync
factory.Start(stopCh)
factory.WaitForCacheSync(stopCh)
```

The second argument to `NewSharedInformerFactory` is the resync period — how often the informer re-lists all objects and fires `OnUpdate` for each, even if nothing changed. This is a safety net for missed events; `0` disables resync.

### Lister vs direct store access

The lister provides a type-safe wrapper over the store:

```go
pod, err := podInformer.Lister().Pods("default").Get("my-pod")
// Returns *corev1.Pod — no type assertion needed

pods, err := podInformer.Lister().Pods("default").List(labels.Everything())
```

Under the hood, listers call `store.GetByKey` or `store.List` — same as case 06, but with generated type safety.

### Namespace scoping

```go
// All namespaces (most common)
informers.NewSharedInformerFactory(clientset, 0)

// Single namespace
informers.NewSharedInformerFactoryWithOptions(clientset, 0,
    informers.WithNamespace("default"))
```

### Relationship to controller-runtime

`controller-runtime` (cases 01–23 in the companion repo) uses its own cache abstraction built on the same primitives, but hides the factory entirely. Understanding this layer makes controller-runtime's behavior transparent.

## References

- [SharedInformerFactory godoc](https://pkg.go.dev/k8s.io/client-go/informers#SharedInformerFactory)
- [Lister godoc](https://pkg.go.dev/k8s.io/client-go/listers/core/v1#PodLister)
- [client-go architecture](https://github.com/kubernetes/client-go/blob/master/ARCHITECTURE.md)
