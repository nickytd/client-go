# Case 22 — Retry on Conflict

## What this covers

Demonstrates **optimistic concurrency** using the Kubernetes Go client. Shows how `retry.RetryOnConflict` handles write conflicts that occur when multiple goroutines attempt concurrent updates to the same resource.

## Key concepts

| Concept | Purpose |
|---------|---------|
| `resourceVersion` | A version tag on each stored object; incremented on every server-side mutation. Clients include it in Update requests for optimistic concurrency control. |
| `apierrors.IsConflict()` | Detects a 409 Conflict error, which signals that the resourceVersion is stale and the Update was rejected. |
| `retry.DefaultRetry` | Backoff tuned for a *few* unrelated writers: `Steps: 5`, ~10ms base, factor 1.0. With many writers on one object it is too small — see the note below. If all steps are consumed and the write still conflicts, `RetryOnConflict` returns the last conflict error (it does **not** silently drop the write). |
| `wait.Backoff` | The retry schedule `RetryOnConflict` consumes: `Steps` (max attempts), `Duration`/`Factor` (delay growth), `Jitter` (randomization). This case passes a custom one with more steps and real jitter. |
| `retry.RetryOnConflict()` | Wrapper that automatically re-GETs a fresh object and retries the mutation when a Conflict occurs, converging under contention. |
| re-GET-then-mutate | The pattern: fetch → mutate in memory → attempt Update → if Conflict, loop back and fetch again. |

## How to run

Requires a live Kubernetes cluster and valid `$KUBECONFIG`:

```bash
cd 22-retry-on-conflict
go run ./...
```

## Expected output

```
2026/01/02 15:04:05 INFO done counter=10 want=10 conflictsRetried=N
```

Where `N` is the number of times a conflict was detected and retried (the leading
timestamp is from slog's default text handler and will differ each run). `N` varies
by run depending on goroutine scheduling and server timing. With 10 concurrent
workers on a typical cluster, `N >= 0`.

> **Why not `retry.DefaultRetry`?** `DefaultRetry` allows only `Steps: 5`, which is
> tuned for a handful of *unrelated* writers. With 10 workers contending for the *same*
> object this is a thundering herd: each round only one write lands and the other nine
> re-GET and collide again, so a worker that keeps losing burns a step every round. The
> unluckiest few exhaust 5 steps and give up — `RetryOnConflict` returns the conflict
> error and `counter` lands below 10. (Those writes aren't *lost*; they never
> succeeded.) `main.go` instead passes a custom `wait.Backoff{Steps: 20, Factor: 1.5,
> Jitter: 0.5}`: enough rounds for every worker to win once, and strong jitter to
> de-synchronize the herd so they stop colliding in lockstep.
>
> **QPS/Burst.** `main.go` also raises the client limits (`config.QPS = 50`,
> `config.Burst = 100`) above the defaults (QPS=5, Burst=10). With the defaults, the
> workers exceed 5 req/s and client-go throttles each request by ~1s
> (`"client-side throttling"` logs), which both adds noise and widens the race window.
> See [case 20](../20-rate-limiting/) for this knob in depth.

## Key files

- **main.go** — Creates 10 concurrent workers, each incrementing a ConfigMap counter. Without `RetryOnConflict`, most would fail with 409; with it, every increment converges.
- **README.md** — This file.
- **go.mod** — Module declaration and dependencies.
