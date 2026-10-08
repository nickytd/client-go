# Case 12 — CRD via Dynamic Client

## What this case covers

Custom Resource Definitions (CRDs) extend the Kubernetes API with new resource types. This case installs a CRD programmatically and then performs CRUD on instances of that resource using the dynamic client — no generated Go types required.

## Key concepts

| Concept | Description |
|---|---|
| `apiextensionsv1.CustomResourceDefinition` | The CRD object that registers a new API type |
| `apiextensions-apiserver/pkg/client/clientset/clientset` | Typed client for CRD management |
| `dynamic.Resource(gvr).Namespace(ns)` | Dynamic client for the custom resource |
| `unstructured.Unstructured` | Generic representation of a custom resource instance |
| `spec.versions[].served` | Whether this version is currently served by the API server |
| `spec.versions[].storage` | Which version is stored in etcd (exactly one must be true) |

## How to run

```bash
go run ./...
```

Expected output:

```
CRD widgets.example.com created.
Created widget: my-widget
Got widget: my-widget (color: blue)
Updated widget: my-widget (color: red)
Deleted widget: my-widget
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Install CRD, CRUD custom resource via dynamic client |

## Concepts in depth

### CRD registration

```go
crd := &apiextensionsv1.CustomResourceDefinition{
    ObjectMeta: metav1.ObjectMeta{Name: "widgets.example.com"},
    Spec: apiextensionsv1.CustomResourceDefinitionSpec{
        Group: "example.com",
        Names: apiextensionsv1.CustomResourceDefinitionNames{
            Plural: "widgets", Singular: "widget",
            Kind: "Widget", ShortNames: []string{"wd"},
        },
        Scope: apiextensionsv1.NamespaceScoped,
        Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
            Name: "v1", Served: true, Storage: true,
            Schema: &apiextensionsv1.CustomResourceValidation{
                OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
                    Type: "object",
                    Properties: map[string]apiextensionsv1.JSONSchemaProps{
                        "spec": {
                            Type: "object",
                            Properties: map[string]apiextensionsv1.JSONSchemaProps{
                                "color": {Type: "string"},
                                "size":  {Type: "integer"},
                            },
                        },
                    },
                },
            },
        }},
    },
}
```

### Waiting for the CRD to be established

CRD creation is asynchronous — the API server takes a moment to register the new type. Poll the CRD's `Established` condition before using it:

```go
for {
    crd, _ := apiextClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
    for _, cond := range crd.Status.Conditions {
        if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
            return // ready
        }
    }
    time.Sleep(500 * time.Millisecond)
}
```

### CRUD via dynamic client

```go
gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
dynClient := dynamic.NewForConfig(cfg)

// Create
widget := &unstructured.Unstructured{Object: map[string]interface{}{
    "apiVersion": "example.com/v1", "kind": "Widget",
    "metadata": map[string]interface{}{"name": "my-widget"},
    "spec": map[string]interface{}{"color": "blue"},
}}
_, err := dynClient.Resource(gvr).Namespace("default").Create(ctx, widget, metav1.CreateOptions{})
```

### Dynamic vs typed CRD client

| Approach | Pros | Cons |
|---|---|---|
| Dynamic (this case) | No code-gen, works for any CRD | No compile-time safety, verbose field access |
| Typed (case 13) | Type-safe, ergonomic | Requires hand-written or generated types |
| controller-runtime client (companion repo) | Best ergonomics, scheme-based | Requires controller-runtime dependency |

## References

- [CRD docs — Kubernetes](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/)
- [apiextensionsv1 godoc](https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1)
- [dynamic client godoc](https://pkg.go.dev/k8s.io/client-go/dynamic)
