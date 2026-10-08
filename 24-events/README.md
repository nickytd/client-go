# Case 24 — Event Recording

## What this covers

Kubernetes Events are how controllers surface decisions and state changes. They appear in `kubectl describe` output and provide an audit trail of what happened to a resource. This case wires up the modern `events.k8s.io/v1` recorder (`k8s.io/client-go/tools/events`) using a broadcaster: events are simultaneously written to the Kubernetes API and logged locally.

## Key concepts

| Concept | Package | Notes |
|---------|---------|-------|
| `events.NewBroadcaster` | `k8s.io/client-go/tools/events` | Creates a broadcaster for the modern `events.k8s.io/v1` API that multiplexes events to multiple sinks. |
| `StartRecordingToSinkWithContext` + `EventSinkImpl` | `k8s.io/client-go/tools/events` | Configures the broadcaster to write `events.k8s.io/v1` Events back to the API server (sink wraps `EventsV1Interface`). |
| `StartLogging` | `k8s.io/client-go/tools/events` | Adds a local structured logger (a `klog.Logger`) as a sink so events are visible without kubectl. |
| `NewRecorder` | `k8s.io/client-go/tools/events` | Creates a recorder tied to a reporting-controller name. |
| `Eventf` | `k8s.io/client-go/tools/events` | Records a Normal or Warning event. The events/v1 signature is `Eventf(regarding, related, type, reason, action, note, args...)`. |
| `Shutdown` | `k8s.io/client-go/tools/events` | Stops the broadcaster's background goroutines and drains the in-memory queue. **Not** a delivery guarantee — see the gotcha below. |

> **Gotcha — delivery is asynchronous and best-effort.** `recorder.Eventf` enqueues
> each event from a detached goroutine, and `Shutdown` only drains the in-memory
> queue; neither guarantees an event has reached the API server. A long-running
> controller never notices (it shuts the broadcaster down only at process exit,
> long after emitting). But a short-lived program that emits events and
> immediately calls `Shutdown` races its own shutdown against delivery and drops
> events. The fix this case uses: treat the API as the source of truth and poll
> (`wait.PollUntilContextTimeout` over `EventsV1().Events(...).List`) until both
> events are observed, *then* shut down.

## How to run

```bash
cd 24-events
go run ./...
```

The program creates a ConfigMap named `event-demo`, records a Normal event ("Reconciled") and a Warning event ("Degraded"), then deletes the ConfigMap. Events persist for their TTL (default ~1 hour).

To inspect the events immediately, run in another terminal:
```bash
kubectl describe configmap event-demo
```

Or watch the structured log output directly in the first terminal — events print to stdout as they are recorded.

## Expected output

Structured log lines (from `StartLogging`, via klog):
```
I1007 10:48:49.361768   41313 event_broadcaster.go:338] "Event occurred" object="default/event-demo" kind="ConfigMap" apiVersion="v1" type="Normal" reason="Reconciled" action="Reconcile" note="processed ConfigMap event-demo"
I1007 10:48:49.362008   41313 event_broadcaster.go:338] "Event occurred" object="default/event-demo" kind="ConfigMap" apiVersion="v1" type="Warning" reason="Degraded" action="Reconcile" note="example warning at 10:48AM"
```

Plus the final hint line:
```
INFO emitted events object=default/event-demo inspect="kubectl describe configmap event-demo"
```

Both events appear on every run: the program waits for the API to confirm them before shutting down.

## Key files

- `main.go` — broadcaster setup, event recording, and a condition-based wait that confirms delivery via the API before `Shutdown`.
