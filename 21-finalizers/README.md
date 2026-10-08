# Case 21 — Finalizers

See also [`11-owner-references`](../11-owner-references).

## What this case covers

Finalizers are a cleanup mechanism used to delay object deletion while a controller performs cleanup work. This case demonstrates the complete finalizer lifecycle: adding a finalizer to prevent immediate deletion, observing that `DeletionTimestamp` is set instead of removing the object, performing cleanup work, and then removing the finalizer to allow garbage collection to complete.

## Key concepts

| Concept | Description |
|---|---|
| Finalizers slice | List of strings on `ObjectMeta.Finalizers` that block deletion |
| `DeletionTimestamp` | Set when Delete is called on an object with finalizers; object remains until all finalizers are removed |
| Cleanup-then-remove pattern | Perform cleanup work, then use `slices.DeleteFunc` to remove your finalizer from the slice |
| GC completes on last finalizer | Once the last finalizer is removed, the API server immediately deletes the object |

## How to run

Requires a reachable cluster in `~/.kube/config`:

```bash
go run ./...
```

Expected output (four lines showing the finalizer lifecycle):

```
created with finalizers=[client-go-lab/cleanup]
still present after Delete; deletionTimestamp=<timestamp> finalizers=[client-go-lab/cleanup]
get after finalizer removed: err=configmaps "finalizer-demo" not found
```

The final `Get` returns a NotFound error because the object is reaped once the finalizer is removed.

## Key files

| File | Purpose |
|---|---|
| `main.go` | Demonstrates finalizer lifecycle: create, delete (object blocked), cleanup, remove finalizer (object reaped) |
| `go.mod` | Module definition with k8s.io/client-go, k8s.io/api, k8s.io/apimachinery dependencies |

## References

- [Kubernetes finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/)
- [case 11 — owner references and garbage collection](../11-owner-references/README.md)
