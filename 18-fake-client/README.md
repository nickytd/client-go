# Case 18 — Testing with Fake Clients

## What this case covers

The fake clientset is an in-memory, no-cluster-needed implementation of `kubernetes.Interface`. It lets you unit-test controllers and business logic by pre-seeding objects, triggering reconcile functions, and asserting on the API calls made.

## Key concepts

| Concept | Description |
|---|---|
| `fake.NewSimpleClientset(objs...)` | Creates an in-memory clientset pre-populated with objects |
| `clientset.Actions()` | Returns a log of every API call made against the fake |
| `k8stesting.Action` | Interface representing one API call (verb, resource, namespace, object) |
| `k8stesting.ReactionFunc` | Custom handler for specific verbs/resources — inject errors or mutations |
| `fake.NewSimpleClientset` | Accepts `runtime.Object` variadic — any typed API objects |
| `k8stesting.ObjectTracker` | Low-level store; manipulate objects between operations |

## How to run

```bash
go test -v ./...
# No cluster required
```

Expected output:

```
=== RUN   TestCreateConfigMap
--- PASS: TestCreateConfigMap (0.00s)
=== RUN   TestControllerReconcile
--- PASS: TestControllerReconcile (0.00s)
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Tests using the fake clientset — pre-seed, operate, assert actions |

## Concepts in depth

### Basic usage

```go
existing := &corev1.Pod{
    ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
}
fakeClient := fake.NewSimpleClientset(existing)

// Now use fakeClient exactly like a real clientset
pod, err := fakeClient.CoreV1().Pods("default").Get(ctx, "my-pod", metav1.GetOptions{})
```

### Asserting API calls

```go
actions := fakeClient.Actions()
require.Len(t, actions, 1)

getAction := actions[0].(k8stesting.GetAction)
assert.Equal(t, "get", getAction.GetVerb())
assert.Equal(t, "pods", getAction.GetResource().Resource)
assert.Equal(t, "my-pod", getAction.GetName())
```

### Injecting errors (reactor chain)

```go
fakeClient.PrependReactor("create", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
    return true, nil, errors.New("quota exceeded")
})
```

Reactors run in order; `PrependReactor` inserts before the default object-tracker reactor. Return `handled=false` to fall through to the next reactor.

### Testing delete cascading

```go
fakeClient.PrependReactor("delete", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
    // simulate ownerRef GC by manually deleting children
    ...
    return false, nil, nil  // let default reactor also delete the owner
})
```

### Limitations of the fake client

| Limitation | Workaround |
|---|---|
| No admission webhooks | envtest (case 19) |
| No watch events triggered by mutations | Wire events manually via `fakeClient.Tracker().Add(...)` |
| No strategic merge patch semantics | JSON merge patch only |
| No server-side apply field tracking | SSA behaves like a plain update |

For integration tests that need real API server semantics (webhook firing, SSA ownership, CRD validation), use envtest (case 19).

### Watch simulation

```go
watcher := watch.NewFake()
fakeClient.PrependWatchReactor("pods", func(action k8stesting.Action) (bool, watch.Interface, error) {
    return true, watcher, nil
})

// Trigger watch events in your test:
watcher.Add(&corev1.Pod{...})
watcher.Modify(&corev1.Pod{...})
watcher.Delete(&corev1.Pod{...})
```

## References

- [fake clientset godoc](https://pkg.go.dev/k8s.io/client-go/kubernetes/fake)
- [k8s.io/client-go/testing godoc](https://pkg.go.dev/k8s.io/client-go/testing)
- [Testing controllers — Kubebuilder book](https://book.kubebuilder.io/cronjob-tutorial/writing-tests)
