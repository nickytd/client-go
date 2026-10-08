# client-go Learning Lab

A hands-on workspace for deeply learning [`kubernetes/client-go`](https://github.com/kubernetes/client-go).
Each top-level folder is an independent Go module with a focused, runnable example that builds on the previous one.

---

## Quick start

```bash
# Clone and enter the repo
cd client-go/

# Run any case directly from its folder
cd 01-kubeconfig
go run ./...
```

---

## Learning path

Work through the cases in order — each one introduces a concept that later cases build on.

### Contents

- **Foundation** — [01 kubeconfig](01-kubeconfig/) · [02 typed-client](02-typed-client/) · [03 dynamic-client](03-dynamic-client/) · [04 discovery](04-discovery/)
- **Reading & Watching** — [05 list-watch](05-list-watch/) · [06 informers](06-informers/) · [07 shared-informer-factory](07-shared-informer-factory/) · [08 indexers](08-indexers/)
- **Work Queues & Controllers** — [09 workqueue](09-workqueue/) · [10 basic-controller](10-basic-controller/) · [10 factory-controller *(variant)*](10-shared-informer-factory-controller/) · [11 owner-references](11-owner-references/)
- **Custom Resources** — [12 crd-dynamic](12-crd-dynamic/) · [13 typed-crd](13-typed-crd/) · [13 typed-crd-with-version *(variant)*](13-typed-crd-with-version/) · [14 crd-informer](14-crd-informer/)
- **Advanced Patterns** — [15 patch](15-patch/) · [16 subresources](16-subresources/) · [17 leader-election](17-leader-election/) · [18 fake-client](18-fake-client/) · [19 envtest](19-envtest/) · [20 rate-limiting](20-rate-limiting/)
- **Controller Correctness** — [21 finalizers](21-finalizers/) · [22 retry-on-conflict](22-retry-on-conflict/) · [23 crd-status](23-crd-status/) · [24 events](24-events/)
- **Apply & Scale** — [25 apply-configurations](25-apply-configurations/) · [26 pagination](26-pagination/) · [27 selectors](27-selectors/) · [28 watch-bookmarks](28-watch-bookmarks/)
- **Testing & Observability** — [29 fake-reactors](29-fake-reactors/) · [30 metrics](30-metrics/)

### Foundation

| # | Folder | Concept | What you learn |
|---|--------|---------|----------------|
| 01 | [`01-kubeconfig`](01-kubeconfig/) | KubeConfig & client creation | `clientcmd`, in-cluster vs out-of-cluster config, `rest.Config` |
| 02 | [`02-typed-client`](02-typed-client/) | Typed clientset | `kubernetes.NewForConfig`, `CoreV1()`, `AppsV1()` — CRUD on built-in resources |
| 03 | [`03-dynamic-client`](03-dynamic-client/) | Dynamic client | `dynamic.NewForConfig`, `unstructured.Unstructured`, arbitrary GVR |
| 04 | [`04-discovery`](04-discovery/) | Discovery client | List API groups/resources, check server version, map GVK↔GVR |

### Reading & Watching Resources

| # | Folder | Concept | What you learn |
|---|--------|---------|----------------|
| 05 | [`05-list-watch`](05-list-watch/) | List & Watch | `ListWatch`, resource versions, watch events (`ADDED`, `MODIFIED`, `DELETED`) |
| 06 | [`06-informers`](06-informers/) | SharedInformer | `cache.NewSharedInformer`, `AddEventHandler`, `HasSynced` |
| 07 | [`07-shared-informer-factory`](07-shared-informer-factory/) | SharedInformerFactory | `informers.NewSharedInformerFactory`, typed informers, lister pattern |
| 08 | [`08-indexers`](08-indexers/) | Indexers | Custom index functions, `cache.MetaNamespaceIndexFunc`, `ByIndex` queries |

### Work Queues & Controllers

| # | Folder | Concept | What you learn |
|---|--------|---------|----------------|
| 09 | [`09-workqueue`](09-workqueue/) | Rate-limiting work queue | `workqueue.NewRateLimitingQueue`, exponential backoff, requeue on error |
| 10 | [`10-basic-controller`](10-basic-controller/) | Controller pattern | Informer + workqueue → reconcile loop, idempotent handlers, graceful shutdown |
| 10 | [`10-shared-informer-factory-controller`](10-shared-informer-factory-controller/) | Controller via factory *(variant)* | `SharedInformerFactory`, two resource types → one typed queue, dispatch on kind |
| 11 | [`11-owner-references`](11-owner-references/) | Owner references & GC | `OwnerReference`, cascading deletion, `PropagationPolicy` |

### Custom Resources

| # | Folder | Concept | What you learn |
|---|--------|---------|----------------|
| 12 | [`12-crd-dynamic`](12-crd-dynamic/) | CRD via dynamic client | CRUD on custom resources without code-gen using `unstructured` |
| 13 | [`13-typed-crd`](13-typed-crd/) | Typed CRD client | Hand-written `runtime.Object` types, typed REST client, custom scheme |
| 13 | [`13-typed-crd-with-version`](13-typed-crd-with-version/) | Multi-version CRD *(variant)* | Two versions in one scheme, `AddConversionFunc`, `scheme.Convert`, served vs storage |
| 14 | [`14-crd-informer`](14-crd-informer/) | Informer for CRD | `dynamicinformer.NewDynamicSharedInformerFactory`, watching custom resources |

### Advanced Patterns

| # | Folder | Concept | What you learn |
|---|--------|---------|----------------|
| 15 | [`15-patch`](15-patch/) | Patch strategies | JSON merge patch, strategic merge patch, server-side apply (SSA) |
| 16 | [`16-subresources`](16-subresources/) | Subresources | Stream pod logs, exec into pods (`remotecommand`), `/status` subresource |
| 17 | [`17-leader-election`](17-leader-election/) | Leader election | `leaderelection` package, `LeaseLock`, multi-replica safety |
| 18 | [`18-fake-client`](18-fake-client/) | Testing with fake clients | `fake.NewSimpleClientset`, pre-seeded objects, asserting recorded actions |
| 19 | [`19-envtest`](19-envtest/) | Integration testing | `controller-runtime/pkg/envtest`, real API server in tests, no cluster needed |
| 20 | [`20-rate-limiting`](20-rate-limiting/) | Client-side rate limiting | `rest.Config` QPS/Burst, `flowcontrol.NewTokenBucketRateLimiter`, throughput measurement |

### Controller Correctness

| # | Folder | Concept | What you learn |
|---|--------|---------|----------------|
| 21 | [`21-finalizers`](21-finalizers/) | Finalizers | Finalizer lifecycle, `DeletionTimestamp`, cleanup before GC |
| 22 | [`22-retry-on-conflict`](22-retry-on-conflict/) | Optimistic concurrency | `retry.RetryOnConflict`, resourceVersion conflicts |
| 23 | [`23-crd-status`](23-crd-status/) | CRD status subresource | `UpdateStatus`, spec vs status, `observedGeneration` |
| 24 | [`24-events`](24-events/) | Event recording | `record.Broadcaster`, `EventRecorder.Eventf`, `kubectl describe` |

### Apply & Scale

| # | Folder | Concept | What you learn |
|---|--------|---------|----------------|
| 25 | [`25-apply-configurations`](25-apply-configurations/) | Typed server-side apply | `applyconfigurations` builders, field-manager ownership |
| 26 | [`26-pagination`](26-pagination/) | List pagination | `ListOptions.Limit/Continue`, chunked lists |
| 27 | [`27-selectors`](27-selectors/) | Field & label selectors | `labels.Selector`, `fields.Selector`, server-side filtering |
| 28 | [`28-watch-bookmarks`](28-watch-bookmarks/) | Watch bookmarks | `AllowWatchBookmarks`, resumable watches, `resourceVersion` |

### Testing & Observability

| # | Folder | Concept | What you learn |
|---|--------|---------|----------------|
| 29 | [`29-fake-reactors`](29-fake-reactors/) | Fake reactors & dynamic fake | `PrependReactor`, `dynamicfake.NewSimpleDynamicClient` |
| 30 | [`30-metrics`](30-metrics/) | Client/workqueue metrics | `workqueue.MetricsProvider`, Prometheus `/metrics` endpoint |

---

## Layout

```
client-go/
├── Makefile                         # build / test / vet / tidy all modules
├── README.md
├── CLAUDE.md
│
├── 01-kubeconfig/
│   ├── go.mod
│   └── main.go
├── 02-typed-client/
│   ├── go.mod
│   └── main.go
├── 03-dynamic-client/
│   ├── go.mod
│   └── main.go
├── 04-discovery/
│   ├── go.mod
│   └── main.go
├── 05-list-watch/
│   ├── go.mod
│   └── main.go
├── 06-informers/
│   ├── go.mod
│   └── main.go
├── 07-shared-informer-factory/
│   ├── go.mod
│   └── main.go
├── 08-indexers/
│   ├── go.mod
│   └── main.go
├── 09-workqueue/
│   ├── go.mod
│   └── main.go
├── 10-basic-controller/
│   ├── go.mod
│   └── main.go
├── 10-shared-informer-factory-controller/
│   ├── go.mod
│   └── main.go
├── 11-owner-references/
│   ├── go.mod
│   └── main.go
├── 12-crd-dynamic/
│   ├── go.mod
│   └── main.go
├── 13-typed-crd/
│   ├── go.mod
│   ├── main.go
│   └── widget_crd.yaml
├── 13-typed-crd-with-version/
│   ├── api/
│   ├── conversion.go
│   ├── go.mod
│   ├── main.go
│   └── widget_crd.yaml
├── 14-crd-informer/
│   ├── go.mod
│   └── main.go
├── 15-patch/
│   ├── go.mod
│   └── main.go
├── 16-subresources/
│   ├── go.mod
│   └── main.go
├── 17-leader-election/
│   ├── go.mod
│   └── main.go
├── 18-fake-client/
│   ├── go.mod
│   └── main.go
├── 19-envtest/
│   ├── go.mod
│   └── main.go
├── 20-rate-limiting/
│   ├── go.mod
│   └── main.go
├── 21-finalizers/
│   ├── go.mod
│   └── main.go
├── 22-retry-on-conflict/
│   ├── go.mod
│   └── main.go
├── 23-crd-status/
│   ├── go.mod
│   ├── main.go
│   └── widget_crd.yaml
├── 24-events/
│   ├── go.mod
│   └── main.go
├── 25-apply-configurations/
│   ├── go.mod
│   └── main.go
├── 26-pagination/
│   ├── go.mod
│   └── main.go
├── 27-selectors/
│   ├── go.mod
│   └── main.go
├── 28-watch-bookmarks/
│   ├── go.mod
│   └── main.go
├── 29-fake-reactors/
│   ├── go.mod
│   ├── main.go
│   └── reactors_test.go
└── 30-metrics/
    ├── go.mod
    └── main.go
```

---

## Key dependencies

| Package | Used in |
|---------|---------|
| `k8s.io/client-go` | All cases |
| `k8s.io/api` | All cases |
| `k8s.io/apimachinery` | All cases |
| `sigs.k8s.io/controller-runtime` | Case 19 (envtest) |

---

## References

- [client-go official examples](https://github.com/kubernetes/client-go/tree/master/examples)
- [client-go godoc](https://pkg.go.dev/k8s.io/client-go)
- [Programming Kubernetes (O'Reilly)](https://www.oreilly.com/library/view/programming-kubernetes/9781492047094/)
- [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [controller-runtime docs](https://pkg.go.dev/sigs.k8s.io/controller-runtime)

---

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
