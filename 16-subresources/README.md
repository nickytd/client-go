# Case 16 — Subresources: Status, Exec, Log Streaming

## What this case covers

Kubernetes exposes secondary REST endpoints (subresources) for operations that don't fit the standard CRUD model: updating only the status, streaming logs, and executing commands in a container. This case demonstrates all three.

## Key concepts

| Concept | Description |
|---|---|
| `/status` subresource | Separate endpoint to update only `status`, not `spec` |
| `/log` subresource | Streams pod container logs |
| `/exec` subresource | Runs a command in a running container |
| `rest.Config.APIPath` | Base path; subresources are appended to the resource URL |
| `remotecommand.NewSPDYExecutor` | SPDY protocol executor for exec/attach/port-forward |
| `remotecommand.StreamOptions` | Wires stdin/stdout/stderr/tty |

## How to run

```bash
# Requires a running pod named 'test-pod' in namespace 'default'
kubectl run test-pod --image=busybox --command -- sleep 3600
go run ./...
```

Expected output:

```
Updated status.phase to: Running
Log output: <pod log lines>
Exec output: total 8\ndrwxr-xr-x ...
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Status update, log streaming, exec via SPDY |

## Concepts in depth

### Status subresource

The status subresource enforces separation of concerns: controllers write to `/status`, users write to the spec. Updating the main resource endpoint ignores the `status` field; updating `/status` ignores the `spec` field.

```go
// Only updates status — spec changes in the object are ignored
pod.Status.Phase = corev1.PodRunning
clientset.CoreV1().Pods("default").UpdateStatus(ctx, pod, metav1.UpdateOptions{})

// Only updates spec — status changes in the object are ignored
clientset.CoreV1().Pods("default").Update(ctx, pod, metav1.UpdateOptions{})
```

For custom resources, enable the status subresource in the CRD spec:
```yaml
subresources:
  status: {}
```

### Log streaming

```go
req := clientset.CoreV1().Pods("default").GetLogs("test-pod", &corev1.PodLogOptions{
    Container: "busybox",
    Follow:    true,      // stream instead of one-shot
    TailLines: ptr(int64(100)),
})
stream, _ := req.Stream(ctx)
defer stream.Close()
io.Copy(os.Stdout, stream)
```

### Exec into a pod

```go
req := clientset.CoreV1().RESTClient().Post().
    Resource("pods").Name("test-pod").Namespace("default").
    SubResource("exec").
    VersionedParams(&corev1.PodExecOptions{
        Command: []string{"ls", "-la"},
        Stdout:  true, Stderr: true,
    }, scheme.ParameterCodec)

exec, _ := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
exec.StreamWithContext(ctx, remotecommand.StreamOptions{
    Stdout: os.Stdout,
    Stderr: os.Stderr,
})
```

### Other subresources

| Subresource | Resource | Purpose |
|---|---|---|
| `/status` | Pods, Deployments, … | Update status field only |
| `/log` | Pods | Stream container logs |
| `/exec` | Pods | Run a command in a container |
| `/attach` | Pods | Attach to a running process |
| `/portforward` | Pods | Forward a local port to the pod |
| `/scale` | Deployments, ReplicaSets | Get/set replica count |
| `/eviction` | Pods | Trigger pod eviction |

## References

- [Kubernetes — Pod subresources](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/)
- [remotecommand godoc](https://pkg.go.dev/k8s.io/client-go/tools/remotecommand)
- [PodLogOptions godoc](https://pkg.go.dev/k8s.io/api/core/v1#PodLogOptions)
- [Subresources in custom resources](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#subresources)
