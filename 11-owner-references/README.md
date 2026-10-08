# Case 11 — Owner References & Garbage Collection

## What this case covers

Owner references let Kubernetes automatically delete dependent (child) objects when their owner is deleted — no controller cleanup code needed. This case creates a ConfigMap owned by another ConfigMap and observes cascading deletion.

## Key concepts

| Concept | Description |
|---|---|
| `metav1.OwnerReference` | Link from a child object to its owner |
| `Controller: true` | Marks this reference as the managing controller (only one per object) |
| `BlockOwnerDeletion: true` | Prevents owner deletion until this object is gone (requires finalizer on owner) |
| `metav1.NewControllerRef` | Helper to build a controller `OwnerReference` from an object |
| Garbage collection | kube-controller-manager watches owner references and deletes orphans |

## How to run

```bash
go run ./...
```

Expected output:

```
Created owner ConfigMap: owner-config (UID: abc-123)
Created child ConfigMap: child-config (ownerRef → owner-config)
Deleting owner-config...
Child child-config garbage collected automatically.
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Create owner + child, delete owner, verify child is gone |

## Concepts in depth

### OwnerReference structure

```go
ownerRef := metav1.OwnerReference{
    APIVersion:         "v1",
    Kind:               "ConfigMap",
    Name:               owner.Name,
    UID:                owner.UID,           // UID is required — prevents stale refs after recreation
    Controller:         ptr(true),           // this is the managing controller
    BlockOwnerDeletion: ptr(true),           // block owner delete until this child is gone
}
child.OwnerReferences = []metav1.OwnerReference{ownerRef}
```

### Using metav1.NewControllerRef

```go
// Shorthand for the above:
ownerRef := metav1.NewControllerRef(owner, schema.GroupVersionKind{
    Group: "", Version: "v1", Kind: "ConfigMap",
})
```

### Cross-namespace restriction

Owner references can only point to objects **in the same namespace** (for namespace-scoped resources). A namespace-scoped resource cannot own a cluster-scoped resource (e.g. a Pod cannot own a Node).

Cluster-scoped resources (Nodes, Namespaces, ClusterRoles) can own other cluster-scoped resources.

### Deletion propagation policies

| Policy | Behaviour |
|---|---|
| `Foreground` | Owner lingers in "terminating" state until all children are gone |
| `Background` (default) | Owner deleted immediately; GC runs in the background |
| `Orphan` | Children's owner references are cleared; children are NOT deleted |

```go
policy := metav1.DeletePropagationForeground
clientset.CoreV1().ConfigMaps("default").Delete(ctx, "owner", metav1.DeleteOptions{
    PropagationPolicy: &policy,
})
```

### Practical use: controllers

When a controller (e.g. Deployment controller) creates a ReplicaSet, it sets itself as the owner. If the Deployment is deleted, all its ReplicaSets — and their Pods — are automatically cleaned up. You don't write cleanup code; you set owner references.

## References

- [Kubernetes — Owners and Dependents](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents)
- [metav1.OwnerReference godoc](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#OwnerReference)
- [Garbage collection — Kubernetes docs](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/)
