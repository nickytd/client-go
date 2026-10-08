# Case 23 — CRD Status Subresource

See also [`13-typed-crd`](../13-typed-crd), [`16-subresources`](../16-subresources)

## What this covers

Kubernetes CRDs can enable a `/status` subresource that enforces strict separation between spec and status. This case demonstrates the separation using the dynamic client against a Widget CRD:

1. **Create** a Widget (spec only — status is ignored on create).
2. **WriteStatus** via `UpdateStatus`, setting `phase` and `observedGeneration`.
3. **Prove** that a plain `Update` cannot overwrite status — the server discards any status changes made through the main endpoint.

## Key concepts

| Concept | Description |
|---|---|
| `spec` | Desired state, written by users/tooling |
| `status` | Observed state, written by controllers |
| `/status` subresource | Separate REST endpoint; `UpdateStatus` hits it; `Update` does not |
| `UpdateStatus` vs `Update` | `UpdateStatus` ignores spec changes; `Update` ignores status changes |
| `metadata.generation` | Incremented by the API server on every spec change |
| `status.observedGeneration` | Copied from `generation` by the controller; signals whether status is current |
| `subresources.status: {}` | CRD field that enables the separation |

## Prerequisite

> **Install this folder's status-enabled CRD before running:**
>
> ```bash
> kubectl apply -f widget_crd.yaml
> ```
>
> Case 13's `widget_crd.yaml` does **not** enable the status subresource — use **this** folder's manifest.

## How to run

```bash
cd 23-crd-status
kubectl apply -f widget_crd.yaml
go run ./...
```

## Expected output

```
created default/status-demo generation=1
status written: map[observedGeneration:1 phase:Ready]
phase after plain Update (unchanged by design): "Ready"
```

The final line confirms that a plain `Update` (which targets the main resource endpoint) cannot modify the status field — it stays `"Ready"` even though `"ShouldBeIgnored"` was set in the object before the call.

## Key files

| File | Purpose |
|---|---|
| `main.go` | Dynamic client: create Widget, write status via UpdateStatus, prove plain Update cannot change status |
| `widget_crd.yaml` | CRD manifest with `subresources.status: {}` enabled (required before running) |
| `go.mod` | Module declaration and dependencies |
