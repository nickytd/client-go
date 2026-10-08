# Case 30 — Client & Workqueue Metrics

See also [`09-workqueue`](../09-workqueue), [`20-rate-limiting`](../20-rate-limiting)

## What this case covers

client-go exposes pluggable metrics hooks. This case registers a Prometheus-backed
`workqueue.MetricsProvider`, drives a named queue, and serves `/metrics` — making the
standard workqueue series (depth, adds, retries, latency, work duration) scrapeable
by Prometheus. This is exactly the telemetry real controllers export in production.

No cluster is required: the case is entirely self-contained.

## Key concepts

| Concept | Description |
|---|---|
| `workqueue.MetricsProvider` | 7-method interface: one constructor per metric series |
| `workqueue.SetProvider` | Registers your provider globally (first call wins) |
| Prometheus registry + `promhttp.HandlerFor` | Serves a dedicated registry (not the global default) |
| Scrapeable series | `depth`, `adds_total`, `retries_total`, `queue_duration_seconds`, `work_duration_seconds`, `unfinished_work_seconds`, `longest_running_processor_seconds` |

## How to run

```bash
go run ./...
```

Then in another terminal:

```bash
curl -s localhost:8080/metrics | grep workqueue
```

No cluster needed — this case runs entirely in-process.

## Expected output

```
# HELP workqueue_adds_total
# TYPE workqueue_adds_total counter
workqueue_adds_total 20
# HELP workqueue_depth
# TYPE workqueue_depth gauge
workqueue_depth 0
# HELP workqueue_longest_running_processor_seconds
# TYPE workqueue_longest_running_processor_seconds gauge
workqueue_longest_running_processor_seconds 0
# HELP workqueue_queue_duration_seconds
# TYPE workqueue_queue_duration_seconds histogram
workqueue_queue_duration_seconds_bucket{le="0.005"} 1
workqueue_queue_duration_seconds_bucket{le="0.01"} 2
...
workqueue_queue_duration_seconds_count 20
# HELP workqueue_retries_total
# TYPE workqueue_retries_total counter
workqueue_retries_total 0
# HELP workqueue_work_duration_seconds
# TYPE workqueue_work_duration_seconds histogram
workqueue_work_duration_seconds_bucket{le="0.005"} 20
...
workqueue_work_duration_seconds_count 20
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | `promProvider` implementing `MetricsProvider`, queue driver, HTTP server |
