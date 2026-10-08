# Case 19 — Integration Testing with envtest

## What this case covers

`envtest` starts a real Kubernetes API server (etcd + kube-apiserver binaries) in a test process, giving you full API server semantics — CRD validation, admission webhooks, watch streams, server-side apply — without a real cluster. This is the standard integration testing approach for controller-runtime based controllers.

## Key concepts

| Concept | Description |
|---|---|
| `envtest.Environment` | Manages the API server and etcd lifecycle in tests |
| `env.Start()` | Starts binaries, returns `*rest.Config` |
| `env.Stop()` | Terminates binaries and cleans up temp dirs |
| `KUBEBUILDER_ASSETS` | Env var pointing to the etcd/kube-apiserver binaries |
| `setup-envtest` | CLI tool to download the right binary versions |
| `env.CRDInstallOptions` | Install CRD manifests before running tests |
| `env.WebhookInstallOptions` | Configure webhook TLS for webhook integration tests |

## How to run

```bash
# Install binaries first (one-time)
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
export KUBEBUILDER_ASSETS=$(setup-envtest use 1.32 -p path --bin-dir /tmp/envtest-bins)

go test -v ./...
```

Expected output:

```
=== RUN   TestEnvtest
Starting envtest API server...
=== RUN   TestEnvtest/create_and_list_configmaps
=== RUN   TestEnvtest/watch_events
--- PASS: TestEnvtest (4.23s)
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | envtest setup/teardown in TestMain, real API server integration tests |

## Concepts in depth

### TestMain pattern

```go
var cfg *rest.Config

func TestMain(m *testing.M) {
    env := &envtest.Environment{
        CRDDirectoryPaths: []string{"config/crd"},
    }
    var err error
    cfg, err = env.Start()
    if err != nil {
        panic(err)
    }
    code := m.Run()
    env.Stop()
    os.Exit(code)
}
```

Each test gets a fresh namespace (via `clientset.CoreV1().Namespaces().Create(...)`) to isolate state, since the API server is shared across tests in a `TestMain`.

### Installing CRDs

```go
env := &envtest.Environment{
    CRDDirectoryPaths:     []string{"config/crd"},          // directory of YAML files
    ErrorIfCRDPathMissing: true,
}
```

Or install programmatically:

```go
crds, err := envtest.InstallCRDs(cfg, envtest.CRDInstallOptions{
    Paths: []string{"config/crd/widget_crd.yaml"},
})
```

### What envtest gives you that fake clients don't

| Feature | fake.Clientset | envtest |
|---|---|---|
| Watch events on mutations | Manual only | Automatic |
| CRD schema validation | No | Yes |
| Admission webhooks | No | Yes (with TLS setup) |
| Server-side apply ownership | No | Yes |
| Real resource version semantics | No | Yes |
| List continuation tokens | No | Yes |

### Binary versions

`setup-envtest` downloads versioned binaries into a local cache:

```bash
setup-envtest list                        # available versions
setup-envtest use 1.32 -p path           # download and print path
setup-envtest use 1.32 --bin-dir ~/bins  # download to specific dir
```

Binaries are version-pinned — if your controller targets 1.32, test against 1.32.

### CI integration

In CI, install binaries to a known path and set `KUBEBUILDER_ASSETS`:

```yaml
- run: |
    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    echo "KUBEBUILDER_ASSETS=$(setup-envtest use 1.32 -p path)" >> $GITHUB_ENV
- run: go test ./...
```

## References

- [envtest godoc](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)
- [setup-envtest](https://github.com/kubernetes-sigs/controller-runtime/tree/main/tools/setup-envtest)
- [Kubebuilder — Writing Tests](https://book.kubebuilder.io/cronjob-tutorial/writing-tests)
- [controller-runtime envtest docs](https://book.kubebuilder.io/reference/envtest)
