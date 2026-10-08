# Case 20 — Client-Side Rate Limiting

## What this case covers

The Kubernetes API server enforces server-side rate limits, but the client has its own token-bucket rate limiter to protect both the server and the client itself. This case shows how to configure `QPS`/`Burst`, observe actual throughput, and understand the tradeoffs between aggressiveness and politeness.

## Key concepts

| Concept | Description |
|---|---|
| `rest.Config.QPS` | Average requests per second the client will send (token refill rate) |
| `rest.Config.Burst` | Maximum burst size above QPS (initial token count) |
| `flowcontrol.NewTokenBucketRateLimiter` | The underlying limiter — `QPS` tokens/sec, `Burst` tokens max |
| `flowcontrol.RateLimiter` | Interface: `Accept()` blocks until a token is available |
| API server Priority and Fairness (APF) | Server-side flow control (separate from client limits) |

## How to run

```bash
go run ./...
```

Expected output:

```
Config: QPS=5, Burst=10
Sending 20 requests...
Completed 20 requests in 2.1s (actual: 9.5 req/s)
  First 10 (burst): 0.2s
  Remaining 10: 1.9s (limited to 5 QPS)
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Measure throughput with different QPS/Burst settings |

## Concepts in depth

### Token bucket mechanics

The token bucket starts full (size = `Burst`). Each request consumes one token. Tokens refill at `QPS` per second.

```
Burst = 10, QPS = 5

t=0:  bucket=[10 tokens]  → 10 requests fire instantly
t=1:  bucket=[5 tokens]   → 5 more fire
t=2:  bucket=[5 tokens]   → 5 more fire
...
```

In client-go, both are set on `rest.Config`:

```go
cfg.QPS   = 50   // 50 requests/sec steady-state
cfg.Burst = 100  // can absorb a burst of 100 immediately
```

Default values (if not set): `QPS=5`, `Burst=10` — conservative, designed for human kubectl usage, not for controllers.

### Recommended settings

| Use case | QPS | Burst |
|---|---|---|
| kubectl / interactive tools | 5 | 10 (default) |
| Single controller | 20–50 | 50–100 |
| High-throughput controllers | 100+ | 200+ |
| Mass reconciliation jobs | tune per cluster | tune per cluster |

### API server Priority and Fairness

Since Kubernetes 1.20, the API server implements [Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/) (APF), which shapes incoming requests into flow queues. Client-side limiting is a courtesy to the server; APF is the enforcement layer.

APF flow schema example:
```yaml
# Controllers get a dedicated priority level with generous limits
kind: PriorityLevelConfiguration
metadata: {name: workload-high}
spec:
  type: Limited
  limited:
    assuredConcurrencyShares: 30
    limitResponse:
      type: Queue
```

### Custom rate limiter

You can provide your own rate limiter implementation:

```go
cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 200)
// or set to nil to disable client-side limiting entirely (not recommended)
```

### Observability

Watch for `429 Too Many Requests` responses — they indicate you're hitting either the client limiter (shouldn't happen — the limiter blocks before sending) or the server-side APF limits (lower your QPS or spread load).

## References

- [rest.Config godoc](https://pkg.go.dev/k8s.io/client-go/rest#Config)
- [flowcontrol godoc](https://pkg.go.dev/k8s.io/client-go/util/flowcontrol)
- [Kubernetes API Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/)
- [client-go rate limiting internals](https://github.com/kubernetes/client-go/blob/master/util/flowcontrol/throttle.go)
