# Case 01 — KubeConfig & Client Creation

## What this case covers

Every client-go program starts by obtaining a `rest.Config`. This case shows how to build one from a kubeconfig file and how the library automatically falls back to in-cluster credentials when running inside a Pod.

## Key concepts

| Concept | Description |
|---|---|
| `clientcmd.BuildConfigFromFlags` | Parses a kubeconfig file and returns a `*rest.Config` |
| `rest.InClusterConfig` | Reads the mounted ServiceAccount token and CA cert inside a Pod |
| `clientcmd.NewNonInteractiveDeferredLoadingClientConfig` | Lazy loader with override support (flags, env) |
| `kubernetes.NewForConfig` | Creates the typed clientset from a `rest.Config` |
| `rest.Config.Host` | The API server URL — useful sanity check on startup |

## How to run

```bash
# Requires a reachable cluster in ~/.kube/config
go run ./...
```

Expected output:

```
Connected to: https://<your-api-server>
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Loads kubeconfig, prints server URL |
| `go.mod` | Module definition with k8s.io/client-go dependency |

## Concepts in depth

### Out-of-cluster vs in-cluster

`clientcmd.BuildConfigFromFlags("", kubeconfigPath)` reads `~/.kube/config` by default. When the same binary runs inside a Pod, `rest.InClusterConfig()` is the correct path — it reads `/var/run/secrets/kubernetes.io/serviceaccount/token` and `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt`.

The idiomatic pattern auto-detects which to use:

```go
cfg, err := rest.InClusterConfig()
if err != nil {
    cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}
```

### rest.Config fields worth knowing

```go
cfg.QPS   = 50   // requests per second
cfg.Burst = 100  // burst above QPS (see case 20)
cfg.Timeout = 30 * time.Second
```

## References

- [client-go examples — out-of-cluster](https://github.com/kubernetes/client-go/tree/master/examples/out-of-cluster-client-configuration)
- [client-go examples — in-cluster](https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration)
- [rest.Config godoc](https://pkg.go.dev/k8s.io/client-go/rest#Config)
- [clientcmd godoc](https://pkg.go.dev/k8s.io/client-go/tools/clientcmd)
