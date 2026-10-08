# Case 09 — Rate-Limiting Work Queue

## What this case covers

The work queue decouples event ingestion from reconciliation. It de-duplicates items (multiple events for the same object collapse to one reconcile), enforces rate limiting, and implements exponential back-off on errors — all without a real cluster.

## Key concepts

| Concept | Description |
|---|---|
| `workqueue.NewTypedRateLimitingQueue` | Creates a type-safe queue with a rate limiter attached |
| `workqueue.DefaultTypedControllerRateLimiter` | Exponential backoff (base 5ms, max 1000s) + per-item rate limit |
| `queue.Add(item)` | Enqueues an item; no-op if already queued |
| `queue.Get()` | Pops one item; blocks if empty |
| `queue.Done(item)` | Marks processing complete; must always be called after `Get` |
| `queue.Forget(item)` | Resets the failure count for an item |
| `queue.AddRateLimited(item)` | Re-enqueues with backoff on failure |
| `queue.NumRequeues(item)` | Returns how many times an item has been re-queued |

## How to run

```bash
go run ./...
# No cluster required
```

Expected output:

```
Processing: pod/default/my-pod (attempt 1)
Processing: pod/default/my-pod (attempt 2)
Processing: pod/default/my-pod (attempt 3)
Giving up on pod/default/my-pod after max retries
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Queue lifecycle: add, process, requeue on error, max retries, shutdown |

## Concepts in depth

### The Get/Done contract

```go
item, quit := queue.Get()
if quit {
    return
}
defer queue.Done(item)  // MUST be called — releases the item from the "processing" set

// If processing fails:
if err != nil {
    queue.AddRateLimited(item)  // re-enqueue with backoff
    return
}
queue.Forget(item)  // success — reset failure count
```

Calling `Done` without `Forget` on success is safe but wasteful — the item will be processed again due to a lingering retry counter.

### Rate limiter types

| Type | Behaviour |
|---|---|
| `DefaultTypedControllerRateLimiter` | Per-item exponential backoff + global bucket rate limiter |
| `TypedBucketRateLimiter` | Token bucket — steady global rate cap |
| `TypedItemExponentialFailureRateLimiter` | Per-item backoff only |
| `NewTypedMaxOfRateLimiter` | Takes the max delay across multiple limiters |

### Deduplication

`queue.Add("default/my-pod")` while `my-pod` is already queued (but not being processed) is a no-op. This is crucial: a burst of 100 events for the same object results in exactly one reconcile.

### Max retries pattern

```go
const maxRetries = 5
if queue.NumRequeues(key) >= maxRetries {
    queue.Forget(key)
    // log and move on — don't infinite-loop
    return
}
queue.AddRateLimited(key)
```

### ShutDown vs ShutDownWithDrain

`ShutDown()` — marks queue closed; `Get()` returns `quit=true` after the queue empties.
`ShutDownWithDrain()` — waits for all in-flight `Get` items to have `Done` called before closing.

## References

- [workqueue godoc](https://pkg.go.dev/k8s.io/client-go/util/workqueue)
- [client-go workqueue example](https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go)
- [Rate limiting in Kubernetes controllers](https://macias.info/entry/202103211739_k8s_informers.md)
