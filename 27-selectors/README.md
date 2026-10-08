# Case 27 — Field & Label Selectors

## What this covers

This case demonstrates **server-side filtering** using label and field selectors.
Label selectors match user-defined labels on objects; field selectors match a restricted set of built-in fields (e.g., `metadata.name`, `status.phase`).
Filtering on the server reduces data transfer and client memory compared to listing everything and filtering in Go.

## Key concepts

| Concept | Example |
|---------|---------|
| `labels.Set` | `labels.Set{"tier": "frontend"}` |
| `labels.SelectorFromSet` | Build a label selector from a map |
| `fields.OneTermEqualSelector` | Build a single-term field selector |
| `ListOptions.LabelSelector` | Pass selector to `List()` for server-side filtering |
| `ListOptions.FieldSelector` | Pass field selector for built-in field matching |
| Server-side vs client-side | Server filters first; client gets only matching objects |
| Indexable fields | ConfigMaps support `metadata.name`; limited by resource type |

## How to run

```bash
cd 27-selectors
go run ./...
```

Requires a cluster and a valid kubeconfig (defaults to `~/.kube/config` or `$KUBECONFIG`).

## Expected output

```
label "tier=frontend" -> [sel-a sel-b]
field "metadata.name=sel-c" -> [sel-c]
```

The first line shows two ConfigMaps with `tier=frontend` label.
The second line shows the single ConfigMap matching `metadata.name=sel-c`.

## Key files

- `main.go` — Creates three ConfigMaps, demonstrates label and field selector List operations
- `labels` package from `k8s.io/apimachinery/pkg/labels` for label selector construction
- `fields` package from `k8s.io/apimachinery/pkg/fields` for field selector construction
