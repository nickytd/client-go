# Case 14 — Informer for a CRD (Dynamic Informer)

## What this case covers

`dynamicinformer.NewDynamicSharedInformerFactory` provides informers for any resource — including CRDs — without generated types. This case watches Widget custom resources and handles add/update/delete events as unstructured data.

## Key concepts

| Concept | Description |
|---|---|
| `dynamicinformer.NewDynamicSharedInformerFactory` | Factory that creates `GenericInformer` for any GVR |
| `GenericInformer` | Informer interface: `.Informer()` and `.Lister()` |
| `.ForResource(gvr)` | Returns the `GenericInformer` for a specific resource type |
| `unstructured.Unstructured` | All objects from a dynamic informer arrive as `*unstructured.Unstructured` |
| `factory.WaitForCacheSync(stopCh)` | Waits for all registered dynamic informers to sync |

## How to run

```bash
# Requires the Widget CRD (see case 12 or 13)
kubectl apply -f config/crd/widget_crd.yaml
go run ./...
# In another terminal:
kubectl apply -f - <<EOF
apiVersion: example.com/v1
kind: Widget
metadata: {name: test-widget, namespace: default}
spec: {color: blue}
EOF
```

Expected output:

```
Dynamic informer cache synced.
ADDED Widget: default/test-widget (color: blue)
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Dynamic informer factory, GVR registration, unstructured event handling |
| `config/crd/widget_crd.yaml` | Widget CRD manifest |

## Concepts in depth

### Dynamic vs typed informer

| Feature | Dynamic informer | Typed informer (case 07) |
|---|---|---|
| Requires generated types | No | Yes |
| Event objects | `*unstructured.Unstructured` | e.g. `*corev1.Pod` |
| Lister | `GenericLister` (returns `runtime.Object`) | Typed lister (e.g. `PodLister`) |
| Works for CRDs | Yes | Only with generated code |

### Setup pattern

```go
gvr := schema.GroupVersionResource{
    Group: "example.com", Version: "v1", Resource: "widgets",
}

dynFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynClient, 30*time.Second)
widgetInformer := dynFactory.ForResource(gvr)

widgetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        u := obj.(*unstructured.Unstructured)
        color, _, _ := unstructured.NestedString(u.Object, "spec", "color")
        fmt.Printf("ADDED %s (color: %s)\n", u.GetName(), color)
    },
})

dynFactory.Start(stopCh)
dynFactory.WaitForCacheSync(stopCh)
```

### Lister usage

```go
lister := widgetInformer.Lister()

// All Widgets in all namespaces
objs, _ := lister.List(labels.Everything())
for _, obj := range objs {
    u := obj.(*unstructured.Unstructured)
    fmt.Println(u.GetName())
}

// Namespace-scoped
nsLister := lister.ByNamespace("default")
objs, _ = nsLister.List(labels.Everything())
```

### Practical controller pattern

In a real CRD controller, the dynamic informer is often combined with a work queue (case 09):

```go
widgetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc:    func(obj interface{}) { queue.Add(keyOf(obj)) },
    UpdateFunc: func(_, obj interface{}) { queue.Add(keyOf(obj)) },
    DeleteFunc: func(obj interface{}) { queue.Add(keyOf(obj)) },
})
```

For typed CRD controllers in production, controller-runtime (companion repo) is preferred — it handles scheme registration, caching, and reconciliation in a unified framework.

## References

- [dynamicinformer godoc](https://pkg.go.dev/k8s.io/client-go/dynamic/dynamicinformer)
- [GenericInformer godoc](https://pkg.go.dev/k8s.io/client-go/informers#GenericInformer)
- [Kubernetes — Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
