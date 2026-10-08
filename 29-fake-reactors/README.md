# Case 29 — Fake Reactors & Dynamic Fake

This case demonstrates how to use **PrependReactor** to inject failures and custom responses into the fake client, and introduces the **dynamic fake client** for testing unstructured/CRD code.

See also [`18-fake-client`](../18-fake-client)

## What this covers

- **PrependReactor** for error injection and response mocking
- Observing actions while delegating to default fake behavior
- **Dynamic fake client** for unstructured and CRD testing without generated types

## Key concepts

| Concept | Description |
|---------|-------------|
| `PrependReactor(verb, resource, ReactionFunc)` | Register a reaction handler for a specific verb/resource pair |
| `ReactionFunc` signature | `func(action Action) (handled bool, obj Object, err error)` |
| `handled=true` | Short-circuits the fake; the reactor's response is final |
| `handled=false` | Falls through to the default fake tracker behavior |
| `dynamicfake.NewSimpleDynamicClient` | Create a dynamic fake client for unstructured objects |
| Scheme registration | `corev1.AddToScheme(scheme)` registers built-in types |

## How to run

```bash
cd 29-fake-reactors
go test ./...
```

No cluster needed — all tests are self-contained.

## Expected output

```
PASS
ok  github.com/home-lab/client-go/29-fake-reactors
```

All three tests pass:
- `TestReactorInjectsError` — error injection with `handled=true`
- `TestReactorCountsCalls` — observation with `handled=false`
- `TestDynamicFake` — unstructured create/get with dynamic fake

## Key files

- `main.go` — Package declaration and documentation
- `reactors_test.go` — Test implementations with reactor patterns and dynamic fake examples
