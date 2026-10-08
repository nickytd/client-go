# Case 05 — List & Watch

## What this case covers

`List` + `Watch` is the foundational primitive that all informers and controllers are built on. This case uses it directly to show the raw mechanics: initial list, resource version bookmarking, and streaming watch events.

## Key concepts

| Concept | Description |
|---|---|
| `cache.NewListWatchFromClient` | Builds a `ListWatch` from a REST client, resource path, and field selector |
| `ResourceVersion` | Opaque cursor; `""` means "start from current state", `"0"` means "any cached version" |
| `watch.Event` | Carries `Type` (ADDED/MODIFIED/DELETED/BOOKMARK/ERROR) and `Object` |
| `metav1.ListOptions.Watch` | When true, the API server streams events instead of returning a list |
| `watch.Interface` | Returned by `.Watch()`; read from `.ResultChan()` |

## How to run

```bash
go run ./...
# In another terminal, create/modify/delete ConfigMaps to see events
kubectl create configmap test --from-literal=key=val
kubectl delete configmap test
```

Expected output:

```
Listed 3 ConfigMaps (resourceVersion: 12345)
ADDED   default/kube-root-ca.crt
MODIFIED default/test
DELETED  default/test
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Manual list+watch loop, event handling, resource version tracking |

## Concepts in depth

### The list+watch protocol

1. **List** — GET `/api/v1/namespaces/default/configmaps` → returns all objects with a `resourceVersion`
2. **Watch** — GET `/api/v1/namespaces/default/configmaps?watch=1&resourceVersion=<rv>` → server streams events from that point forward

This ensures no events are missed between the list and the watch start.

### Resource version semantics

| Value | Meaning |
|---|---|
| `""` | Quorum read — latest state from etcd |
| `"0"` | Any cached version — may be stale |
| `"12345"` | Watch from this exact version forward |

In production, always re-watch from the last seen resource version after a disconnect.

### Watch event types

| Type | When |
|---|---|
| `ADDED` | Object exists (initial list replay or new creation) |
| `MODIFIED` | Object was updated |
| `DELETED` | Object was deleted |
| `BOOKMARK` | Periodic heartbeat carrying latest `resourceVersion` (no object change) |
| `ERROR` | Stream error — status object in `Object` field; 410 Gone means re-list |

### Why informers exist (case 06)

Manually managing the list+watch loop is error-prone: reconnect on error, re-list on 410, concurrent access to the local cache. `cache.SharedInformer` wraps all of this, giving you a local in-memory store and event callbacks without the boilerplate.

## References

- [Kubernetes API — efficient detection of changes](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes)
- [cache.NewListWatchFromClient godoc](https://pkg.go.dev/k8s.io/client-go/tools/cache#NewListWatchFromClient)
- [watch.Interface godoc](https://pkg.go.dev/k8s.io/apimachinery/pkg/watch#Interface)
