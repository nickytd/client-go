# Case 03 — Dynamic Client

## What this case covers

The dynamic client works with any resource — including CRDs and future API groups — without requiring generated Go types. Objects are represented as `unstructured.Unstructured` maps, giving full flexibility at the cost of compile-time type safety.

## Key concepts

| Concept | Description |
|---|---|
| `dynamic.NewForConfig` | Creates the dynamic client from a `rest.Config` |
| `schema.GroupVersionResource` | Identifies a resource type: `{Group, Version, Resource}` |
| `unstructured.Unstructured` | A generic `map[string]interface{}` representing any API object |
| `unstructured.UnstructuredList` | List of unstructured objects |
| `dynamic.Interface.Resource(gvr)` | Returns a namespace or cluster-scoped resource client |

## How to run

```bash
go run ./...
```

Expected output:

```
ConfigMaps in default:
  kube-root-ca.crt
  ...
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Dynamic client — list resources via GVR, extract fields from unstructured data |

## Concepts in depth

### GroupVersionResource (GVR)

Every Kubernetes resource is identified by its GVR:

```go
gvr := schema.GroupVersionResource{
    Group:    "apps",
    Version:  "v1",
    Resource: "deployments",  // plural, lowercase
}
```

For core resources the group is empty (`""`):

```go
podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
```

### Working with unstructured data

```go
// Get a field
name, found, err := unstructured.NestedString(obj.Object, "metadata", "name")

// Set a field
unstructured.SetNestedField(obj.Object, "value", "spec", "replicas")

// Get a slice
containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
```

### Namespace vs cluster-scoped clients

```go
// Namespace-scoped
client.Resource(gvr).Namespace("default").List(ctx, metav1.ListOptions{})

// Cluster-scoped (Nodes, Namespaces, ClusterRoles…)
client.Resource(gvr).List(ctx, metav1.ListOptions{})
```

### When to use the dynamic client

- CRDs without generated types (see case 12)
- Generic controllers that work across resource types
- kubectl-style tools that enumerate arbitrary resources
- Mutation webhooks that handle unknown types

For known core/apps resources, prefer the typed clientset (case 02) — it provides compile-time safety and avoids manual field extraction.

## References

- [dynamic package godoc](https://pkg.go.dev/k8s.io/client-go/dynamic)
- [unstructured package godoc](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured)
- [client-go examples — dynamic client](https://github.com/kubernetes/client-go/tree/master/examples/dynamic-create-update-delete-deployment)
