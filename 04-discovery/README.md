# Case 04 — Discovery Client

## What this case covers

The discovery client interrogates the API server's `/api`, `/apis`, and `/openapi` endpoints to learn what resources, versions, and capabilities exist at runtime. This is how `kubectl api-resources` and REST mappers work.

## Key concepts

| Concept | Description |
|---|---|
| `clientset.Discovery()` | Returns the `DiscoveryInterface` |
| `ServerVersion()` | Returns the cluster's Kubernetes version |
| `ServerGroups()` | Lists all API groups the server exposes |
| `ServerResourcesForGroupVersion` | Lists all resources in a specific group/version |
| `restmapper.GetAPIGroupResources` | Aggregates all groups+resources in one call |
| `meta.RestMapper` | Maps GVK ↔ GVR at runtime |

## How to run

```bash
go run ./...
```

Expected output:

```
Server version: v1.30.0
API groups: ["" "apps" "batch" "networking.k8s.io" ...]
GVR for Deployment: apps/v1/deployments
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Enumerate API groups, server version, map GVK → GVR |

## Concepts in depth

### API server discovery endpoints

| Endpoint | What it returns |
|---|---|
| `/api` | Core group versions (`v1`) |
| `/apis` | All named API groups |
| `/api/v1` | All resources in core v1 |
| `/apis/apps/v1` | All resources in apps/v1 |
| `/openapi/v2` | Full OpenAPI schema |

### GVK vs GVR

- **GVK** (GroupVersionKind) — used in Go types: `apps/v1, Kind=Deployment`
- **GVR** (GroupVersionResource) — used in REST paths: `apps/v1/deployments`

The REST mapper translates between them:

```go
mapping, err := mapper.RESTMapping(schema.GroupKind{Group: "apps", Kind: "Deployment"}, "v1")
// mapping.Resource → {Group:"apps", Version:"v1", Resource:"deployments"}
```

### Caching discovery results

Discovery calls are expensive (multiple HTTP requests). In production controllers:

```go
// Use a cached discovery client
cachedDisc := memory.NewMemCacheClient(clientset.Discovery())
// Or the disk-based cache (default in kubectl):
// ~/.kube/cache/discovery/
```

`restmapper.GetAPIGroupResources` populates a mapper once; subsequent lookups are in-memory.

### Practical uses

- Check whether a CRD is installed before proceeding
- Generic controllers/tools that adapt to cluster capabilities
- Admission webhooks that list available resource versions
- Helm-style templating that resolves `apiVersion` per cluster

## References

- [DiscoveryInterface godoc](https://pkg.go.dev/k8s.io/client-go/discovery#DiscoveryInterface)
- [restmapper godoc](https://pkg.go.dev/k8s.io/client-go/restmapper)
- [Kubernetes API groups docs](https://kubernetes.io/docs/reference/using-api/#api-groups)
