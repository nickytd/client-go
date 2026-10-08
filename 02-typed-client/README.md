# Case 02 — Typed Clientset

## What this case covers

The typed clientset is the most common way to interact with core Kubernetes resources. This case walks through listing Pods and Deployments, then performs a full create/get/delete cycle on a ConfigMap.

## Key concepts

| Concept | Description |
|---|---|
| `kubernetes.NewForConfig` | Creates the typed clientset from a `rest.Config` |
| `clientset.CoreV1()` | Access to core API group (Pods, Services, ConfigMaps, Namespaces…) |
| `clientset.AppsV1()` | Access to apps group (Deployments, StatefulSets, DaemonSets…) |
| `metav1.ListOptions` | Filtering by label selector, field selector, resource version |
| `metav1.GetOptions` | Options for a single-resource GET |
| `metav1.DeleteOptions` | Propagation policy, grace period, preconditions |

## How to run

```bash
# Requires a reachable cluster with at least one Pod running
go run ./...
```

Expected output:

```
Pods in all namespaces:
  default/my-cm
  kube-system/coredns-...
Deployments in default:
  ...
Created ConfigMap: demo-configmap
Got ConfigMap data: hello=world
Deleted ConfigMap: demo-configmap
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Typed clientset — list Pods/Deployments, CRUD a ConfigMap |

## Concepts in depth

### Resource client hierarchy

```
clientset
 └── CoreV1()              → PodInterface, ConfigMapInterface, …
 └── AppsV1()             → DeploymentInterface, StatefulSetInterface, …
 └── NetworkingV1()       → IngressInterface, …
 └── RbacV1()             → RoleInterface, ClusterRoleInterface, …
```

Each interface is namespaced: `clientset.CoreV1().Pods("default")` or cluster-scoped: `clientset.CoreV1().Nodes()`.

### The standard CRUD pattern

```go
// Create
cm, err := clientset.CoreV1().ConfigMaps("default").Create(ctx, &corev1.ConfigMap{...}, metav1.CreateOptions{})

// Get
cm, err := clientset.CoreV1().ConfigMaps("default").Get(ctx, "name", metav1.GetOptions{})

// Update (full replace — watch out for resource version conflicts)
cm, err := clientset.CoreV1().ConfigMaps("default").Update(ctx, cm, metav1.UpdateOptions{})

// Delete
err := clientset.CoreV1().ConfigMaps("default").Delete(ctx, "name", metav1.DeleteOptions{})
```

### Resource version and optimistic concurrency

Every Kubernetes object carries a `metadata.resourceVersion`. Passing a stale version to `Update` returns a `409 Conflict`. The correct pattern is a retry loop: re-`Get`, mutate, re-`Update`.

## References

- [kubernetes/client-go — typed client](https://github.com/kubernetes/client-go/tree/master/kubernetes)
- [CoreV1Interface godoc](https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#CoreV1Interface)
- [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
