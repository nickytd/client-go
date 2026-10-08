# Case 15 — Patch Strategies

## What this case covers

Kubernetes supports multiple patch strategies, each with different merge semantics and conflict behaviour. This case demonstrates all three patch types on ConfigMaps, showing when to use each.

## Key concepts

| Concept | Description |
|---|---|
| `types.MergePatchType` | RFC 7396 JSON merge patch — replaces nested objects wholesale |
| `types.StrategicMergePatchType` | Kubernetes-specific — merges lists by key, not by replace |
| `types.ApplyPatchType` | Server-side apply (SSA) — field manager ownership, conflict detection |
| `types.JSONPatchType` | RFC 6902 JSON Patch — explicit op/path/value operations |
| Field manager | SSA identity string — who owns which fields |
| `metav1.ApplyOptions` | `Force: true` — take ownership of conflicting fields |

## How to run

```bash
go run ./...
```

Expected output:

```
Created ConfigMap with data: {env: prod, region: us-east-1}
After merge patch: {env: staging}         (region key removed!)
After strategic merge patch: containers merged by name
After server-side apply: field manager 'case15' owns spec.data
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | All three patch types with visible side effects per strategy |

## Concepts in depth

### JSON Merge Patch (RFC 7396)

Shallow merge — keys are replaced, not merged. **Nested objects are replaced wholesale.**

```go
patch := []byte(`{"metadata":{"labels":{"env":"staging"}}}`)
clientset.CoreV1().ConfigMaps("default").Patch(ctx, name,
    types.MergePatchType, patch, metav1.PatchOptions{})
```

Gotcha: to add a label without removing others, you must include ALL existing labels in the patch. To remove a key, set it to `null`.

### Strategic Merge Patch

Kubernetes-specific extension: lists are merged by a strategic key (e.g. `name` for container lists), not replaced.

```go
// Add a new container without removing existing ones:
patch := []byte(`{"spec":{"containers":[{"name":"sidecar","image":"nginx"}]}}`)
clientset.AppsV1().Deployments("default").Patch(ctx, name,
    types.StrategicMergePatchType, patch, metav1.PatchOptions{})
```

Only works for built-in types that have strategic merge patch annotations. CRDs use JSON merge patch.

### Server-Side Apply (SSA)

The API server tracks which field manager owns which fields. Conflicts are surfaced explicitly.

```go
// Apply a desired state:
desired := &corev1.ConfigMap{
    TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
    ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
    Data: map[string]string{"key": "value"},
}
data, _ := json.Marshal(desired)
clientset.CoreV1().ConfigMaps("default").Patch(ctx, name,
    types.ApplyPatchType, data,
    metav1.PatchOptions{FieldManager: "my-controller"})
```

SSA is the recommended approach for controllers because:
- Multiple actors can own different fields without stepping on each other
- Drift detection is built in — the API server knows what you last applied
- `Force: true` takes ownership of conflicted fields

### Comparison

| Strategy | List merge | CRD support | Conflict detection | Use case |
|---|---|---|---|---|
| Merge patch | Replace | Yes | No | Simple key updates |
| Strategic merge | By key | No | No | Built-in types |
| Server-side apply | By key | Yes | Yes | Controllers (preferred) |
| JSON Patch | N/A | Yes | No | Precise atomic ops |

## References

- [Kubernetes — Update API objects in place using kubectl patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/)
- [Server-Side Apply — Kubernetes docs](https://kubernetes.io/docs/reference/using-api/server-side-apply/)
- [RFC 7396 — JSON Merge Patch](https://www.rfc-editor.org/rfc/rfc7396)
- [types.PatchType godoc](https://pkg.go.dev/k8s.io/apimachinery/pkg/types#PatchType)
