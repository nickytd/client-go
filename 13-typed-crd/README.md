# Case 13 — Typed CRD Client (Hand-Written Types)

> **Next:** [`13-typed-crd-with-version`](../13-typed-crd-with-version) extends this with a second
> API version (`v1`) and hand-written conversion functions, lifting the `.WithoutConversion()`
> limitation noted below.

## What this case covers

Typed clients give you compile-time safety and IDE completion for custom resources. This case hand-writes the Go types and builds a typed REST client for a CRD — showing exactly what code generators like `controller-gen` produce automatically.

## Key concepts

| Concept | Description |
|---|---|
| `runtime.Object` | Interface every API type must implement: `DeepCopyObject()` |
| `runtime.Scheme` | Registry of GVK ↔ Go type mappings |
| `scheme.Builder` | Helper to register types with a scheme |
| `rest.RESTClientFor` | Builds a low-level REST client configured for a specific API group |
| `serializer.NewCodecFactory` | Codec for encoding/decoding API objects |
| `rest.Resource(resource).Namespace(ns)` | Typed REST request builder |

## How to run

```bash
# Requires the Widget CRD to be installed first
kubectl apply -f widget_crd.yaml
go run ./...
```

Expected output:

```
level=INFO msg="created Widget" name=my-widget color=blue
level=INFO msg="got Widget" name=my-widget
level=INFO msg="listed Widgets" count=1
level=INFO msg="deleted Widget" name=my-widget
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | Hand-written Widget types, scheme registration, typed REST client |
| `widget_crd.yaml` | CRD manifest (apply before running) |

## Concepts in depth

### Minimum type requirements

Every API type must implement `runtime.Object`:

```go
type Widget struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   WidgetSpec   `json:"spec,omitempty"`
    Status WidgetStatus `json:"status,omitempty"`
}

type WidgetList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items []Widget  `json:"items"`
}

func (w *Widget) DeepCopyObject() runtime.Object     { c := *w; return &c }
func (w *WidgetList) DeepCopyObject() runtime.Object { c := *w; return &c }
```

### Scheme registration

```go
var SchemeBuilder = &scheme.Builder{GroupVersion: schema.GroupVersion{
    Group: "example.com", Version: "v1alpha1",
}}

func init() {
    SchemeBuilder.Register(&Widget{}, &WidgetList{})
}
```

### Where is the connection between the CRD and the typed object?

This is the central idea of the case — and the thing that distinguishes it from
case 12's dynamic client. The Go `Widget` struct never references the CRD
directly. Instead, the two are bound through a shared coordinate, the
**GroupVersionKind (GVK)**, via the **scheme**:

```go
scheme.AddKnownTypes(schemeGVK, &Widget{}, &WidgetList{})
//                   └─ GVK ─┘  └── Go types ──┘
```

That single call creates a bidirectional map inside the scheme:

```
GVK  example.com/v1alpha1, Kind=Widget  ⇄  Go type  *Widget
```

Everything else is wiring that *uses* this map. Three coordinates must agree,
but only one of them is checked by the Go compiler:

| Coordinate | Where in `main.go` | Must match |
|---|---|---|
| **GVK** (group/version + kind) | `schemeGVK`, the `TypeMeta` on the object | ↔ Go type, via `scheme.AddKnownTypes` — the compiler verifies `Widget` is a `runtime.Object` |
| **Resource** (plural name) | `Resource("widgets")` on the request | → the CRD manifest's `names.plural` (runtime only) |
| **`spec` fields** | `WidgetSpec` struct tags (`color`, `size`) | → the CRD's `openAPIV3Schema` (runtime only) |

So the **type ↔ GVK** half lives in `AddKnownTypes` and is compiler-assisted;
the **GVK ↔ CRD** half is plain string matching the API server validates at
request time. Case 13 buys you typed field access and (de)serialization — it
does **not** guarantee your struct matches the installed CRD. Get the plural or
a struct tag wrong and the code still compiles; it fails against the cluster.

### Building the typed REST client

```go
s := runtime.NewScheme()
SchemeBuilder.AddToScheme(s)
metav1.AddToGroupVersion(s, schema.GroupVersion{Group: "example.com", Version: "v1alpha1"})

cfg.GroupVersion = &schema.GroupVersion{Group: "example.com", Version: "v1alpha1"}
cfg.APIPath = "/apis"
cfg.NegotiatedSerializer = serializer.NewCodecFactory(s)

restClient, _ := rest.RESTClientFor(cfg)

// Typed GET
var w Widget
restClient.Get().Resource("widgets").Namespace("default").Name("my-widget").Do(ctx).Into(&w)
```

### What is a codec?

A **codec** (coder/decoder) is the machinery that converts between a Go struct
and the bytes on the wire:

```
*Widget  ⇄  {"apiVersion":"example.com/v1alpha1","kind":"Widget","spec":{...}}
```

It is more than `encoding/json`. On **decode** it reads `apiVersion` + `kind`
out of the bytes, looks that GVK up in the **scheme**, and decodes into the
matching Go type — the same scheme map described above. On **encode** it writes
those identifiers back into the payload. The scheme is the registry (GVK → Go
type); the codec is the worker that consults it.

```go
codecs := serializer.NewCodecFactory(scheme)   // build codecs FROM the scheme
cfg.NegotiatedSerializer = codecs.WithoutConversion()
```

- **`CodecFactory`** (not a single codec): a factory that hands out codecs per
  wire format. *"Negotiated"* means the client and server pick the format via
  HTTP `Content-Type`/`Accept` headers — JSON for CRDs, often protobuf for core
  types.
- **`.WithoutConversion()`**: a full codec can convert between API versions
  (e.g. `v1alpha1` ↔ `v1`) using registered conversion functions. Hand-written
  types have none, so we ask for a codec that skips conversion and encodes the
  single version as-is. Generated multi-version clients drop this.

This codec is exactly what makes `.Body(w)` and `.Into(&w)` typed. Case 12's
dynamic client has no scheme and no codec in user code — it uses a built-in
unstructured serializer that produces a `map[string]any`, which is why it has
no typed field access.

### What controller-gen generates

Running `controller-gen object:headerFile=... paths=./...` generates:
- `zz_generated_deepcopy.go` — `DeepCopyObject()` and `DeepCopy()` for all types
- (optionally) `zz_generated_register.go` — scheme registration

The code in this case is exactly what those generators produce — understanding it makes generated code transparent.

### Alternative: controller-runtime client

The `controller-runtime` client (companion repo, cases 03–05) wraps this complexity with `client.Client` — a scheme-based client that automatically uses the right REST path and codec for any registered type.

## References

- [runtime.Object godoc](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#Object)
- [scheme.Builder godoc](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime/schema)
- [Kubernetes code generation](https://github.com/kubernetes/code-generator)
- [controller-gen](https://book.kubebuilder.io/reference/controller-gen)
