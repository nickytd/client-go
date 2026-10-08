# Case 17 — Leader Election

## What this case covers

Running multiple replicas of a controller for high availability requires only one replica to act at a time. Leader election via the Kubernetes Lease resource ensures exactly one replica holds the "leader" lock, with automatic failover when the leader dies.

## Key concepts

| Concept | Description |
|---|---|
| `leaderelection.RunOrDie` | Blocks the calling goroutine, running the leader election loop |
| `leaderelection.LeaderElectionConfig` | Tuning parameters and lifecycle callbacks |
| `LeaseDuration` | How long a leader holds the lock if it stops renewing |
| `RenewDeadline` | How long the leader retries renewing before giving up |
| `RetryPeriod` | How often candidates retry acquiring the lock |
| `ReleaseOnCancel` | Release the lease immediately when the context is cancelled |
| `resourcelock.LeaseLock` | The lock object backed by a coordination.k8s.io Lease |
| `OnStartedLeading` | Called when this replica becomes leader — start your work here |
| `OnStoppedLeading` | Called when this replica loses leadership — stop your work |
| `OnNewLeader` | Called on all replicas when any replica becomes leader |

## How to run

```bash
# Run two instances in separate terminals to see failover.
# REPLICA_ID is the lock identity — it MUST differ per instance,
# otherwise both candidates look identical and election can't exclude them.
REPLICA_ID=replica-1 go run ./...
# In another terminal:
REPLICA_ID=replica-2 go run ./...
# Kill replica-1 and observe replica-2 become leader after LeaseDuration.
```

Expected output (replica-1):

```
I am the leader — doing work  id=replica-1
```

Expected output (replica-2, after replica-1 is killed):

```
new leader elected  id=replica-2 leader=replica-1
I am the leader — doing work  id=replica-2
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | LeaderElectionConfig, RunOrDie, graceful handoff |

## Concepts in depth

### How Kubernetes leader election works

1. All replicas attempt to create/update a Lease object in Kubernetes
2. The Lease carries `holderIdentity` (who owns it) and `renewTime` (last heartbeat)
3. The current leader continuously renews the Lease before `LeaseDuration` expires
4. If the Lease isn't renewed within `LeaseDuration`, any candidate can claim it

```
LeaseDuration = 5s   ← How long a dead leader stays "leader"
RenewDeadline = 2s   ← How long the leader retries before giving up
RetryPeriod   = 1s   ← How often followers try to acquire
```

Rule of thumb: `RenewDeadline < LeaseDuration`, `RetryPeriod < RenewDeadline/2`.

### Lock resource types

| Type | Notes |
|---|---|
| `Lease` (recommended) | Dedicated resource, minimal size, preferred since 1.14 |
| `ConfigMap` | Legacy; used before Lease existed |
| `Endpoints` | Legacy; not recommended |

```go
lock := &resourcelock.LeaseLock{
    LeaseMeta: metav1.ObjectMeta{
        Name:      "client-go-lab-leader",
        Namespace: "default",
    },
    Client: clientset.CoordinationV1(),
    LockConfig: resourcelock.ResourceLockConfig{
        Identity: id, // REPLICA_ID (unique per candidate), falling back to hostname
    },
}
```

### Lifecycle callbacks

```go
leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
    Lock:            lock,
    LeaseDuration:   5 * time.Second,
    RenewDeadline:   2 * time.Second,
    RetryPeriod:     1 * time.Second,
    ReleaseOnCancel: true,
    Callbacks: leaderelection.LeaderCallbacks{
        OnStartedLeading: func(ctx context.Context) {
            // Became leader — start your work. ctx is cancelled on lost leadership.
            slog.Info("I am the leader — doing work", "id", id)
            time.Sleep(30 * time.Second)
        },
        OnStoppedLeading: func() {
            // Leadership lost — exit so the process restarts clean.
            slog.Info("lost leadership — stopping", "id", id)
            os.Exit(0)
        },
        OnNewLeader: func(identity string) {
            // Fires on every candidate; skip the notice when it's us.
            if identity != id {
                slog.Info("new leader elected", "id", id, "leader", identity)
            }
        },
    },
})
```

### controller-runtime equivalent

controller-runtime's `manager.Options.LeaderElection = true` configures this automatically — see case 14 in the companion repo.

## References

- [leaderelection godoc](https://pkg.go.dev/k8s.io/client-go/tools/leaderelection)
- [Kubernetes — Leader Election](https://kubernetes.io/docs/concepts/cluster-administration/controller/)
- [resourcelock godoc](https://pkg.go.dev/k8s.io/client-go/tools/leaderelection/resourcelock)
