# Case 25 — Typed Apply Configurations (SSA)

See also [`15-patch`](../15-patch).

## What this covers

This case demonstrates **typed server-side apply (SSA)** using `applyconfigurations` builders.
Unlike raw JSON patches (case 15), typed apply configurations provide strong typing, nil omission,
and integrated field ownership tracking through `ManagedFields`.

Multiple managers can apply to the same object safely as long as they target disjoint fields.
Conflicts (two managers claiming the same field) are rejected by default; `Force` mode allows
forced acquisition of ownership.

## Key concepts

| Concept | Example |
|---------|---------|
| `applyconfigurations/core/v1.ConfigMap` | `applyconfigcorev1.ConfigMap(name, namespace)` |
| `.WithData()` | Build the data map fields |
| `Apply()` + `FieldManager` | Apply configuration and record ownership |
| Disjoint field coexistence | `alice` owns `key1`, `bob` adds `key2` — no conflict |
| Conflict without `Force` | Second manager's conflicting field apply is rejected |
| `Force: true` | Override ownership; new manager takes the field |

## How to run

```bash
cd 25-apply-configurations
go run ./...
```

Requires a cluster and a valid kubeconfig (defaults to `~/.kube/config` or `$KUBECONFIG`).

## Expected output

Four stages of the ConfigMap's evolution:

1. **after alice**: `alice` applies `key1=owned-by-alice`
   ```
   data=map[key1:owned-by-alice]
   manager=alice owns=...
   ```

2. **after bob**: `bob` applies `key2=owned-by-bob`; alice still owns `key1`
   ```
   data=map[key1:owned-by-alice key2:owned-by-bob]
   manager=alice owns=...
   manager=bob owns=...
   ```

3. **bob overwrites alice's field without force**: Attempt fails with a conflict error
   ```
   bob overwrites alice's field without force: err=Apply failed with 1 conflict(s): [...]
   ```

4. **after bob --force**: `bob` force-applies and steals ownership of `key1`
   ```
   data=map[key1:stolen-by-bob key2:owned-by-bob]
   manager=bob owns=...
   ```

## Key files

- `main.go` — Demonstrates the four stages and conflict resolution
- Apply configurations generated from `k8s.io/client-go/applyconfigurations/core/v1`
- `ManagedFields` inspection to show per-manager ownership
