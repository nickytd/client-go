# Case 13 (versioned) — Multi-Version CRD with In-Process Conversion

> **Companion to [`13-typed-crd`](../13-typed-crd).** That case built a typed client for a
> single version and used `codecs.WithoutConversion()` because hand-written types had no
> conversion functions. This case removes that limitation: it registers **two** versions
> (`v1alpha1` and `v1`) in one scheme and converts between them in-process.

## What this case covers

Real CRDs evolve. You ship `v1alpha1`, learn from it, then introduce `v1` with a better-shaped spec — but existing objects and clients still use the old version. Kubernetes handles this by keeping **multiple served versions** and **converting** between them. This case shows the conversion machinery itself, hand-written and running entirely in-process (no API server, no webhook).

The Widget spec evolves like this:

| | `v1alpha1` | `v1` |
|---|---|---|
| color | `Color string` | `Color string` |
| size | `Size int` | `Dimensions{ Width, Height int }` |
| material | — | `Material string` |

## Key concepts

| Concept | Description |
|---|---|
| Served vs storage version | Every served version is reachable via the API; exactly one is the storage version written to etcd |
| `scheme.AddKnownTypes` (×N) | Register each version's Go types under its own GroupVersion in the same scheme |
| `scheme.AddConversionFunc` | Register a typed conversion between two Go types, both directions |
| `scheme.Convert(src, dst, nil)` | Run a registered conversion — the primitive a codec uses on decode |
| Lossy conversion | Down-converting to an older, smaller spec necessarily drops fields |

## How to run

```bash
go run ./...
```

Expected output:

```
level=INFO msg="source v1alpha1" color=blue size=5
level=INFO msg="converted to v1" color=blue width=5 height=5 material=unknown
level=INFO msg="mutated v1" width=8 height=3 material=aluminium
level=INFO msg="converted back to v1alpha1 (lossy: height and material dropped)" color=blue size=8
```

No cluster is required — the whole demo is in-process. The optional
`widget_crd.yaml` shows how the two versions are declared to the API server.

## Key files

| File | Purpose |
|---|---|
| `api/v1alpha1/types.go` | Original Widget type: `Color`, flat `Size` |
| `api/v1/types.go` | New Widget type: `Color`, `Dimensions{Width,Height}`, `Material` |
| `conversion.go` | `RegisterConversions` — the v1alpha1 ↔ v1 functions |
| `main.go` | Build the scheme, register conversions, run up/down conversions |
| `widget_crd.yaml` | Optional CRD serving both versions (v1 is storage) |

## Concepts in depth

### Why multiple versions, and served vs storage

A CRD lists its versions under `spec.versions`. Each has two independent flags:

- **`served: true`** — clients may read and write this version through the API.
- **`storage: true`** — objects are persisted in etcd in this version. **Exactly one** version sets this.

So you can serve `v1alpha1` and `v1` simultaneously while storing everything as `v1`. A client that GETs the `v1alpha1` endpoint gets the stored object converted down to `v1alpha1`; a client that writes `v1alpha1` has it converted up to `v1` for storage. That conversion is exactly what this case implements by hand.

### Registering two versions in one scheme

Each version package registers its own types under its own GroupVersion:

```go
scheme := runtime.NewScheme()
v1alpha1.AddToScheme(scheme) // AddKnownTypes(example.com/v1alpha1, &Widget{}, ...)
v1.AddToScheme(scheme)       // AddKnownTypes(example.com/v1,       &Widget{}, ...)
```

The scheme now maps **two** GVKs to two distinct Go types. Because both are
registered, the scheme's converter can bridge them — once we tell it how.

### The conversion contract

A conversion function is just `func(in *A, out *B) error`, registered for both
directions:

```go
scheme.AddConversionFunc((*v1alpha1.Widget)(nil), (*v1.Widget)(nil),
    func(a, b any, _ conversion.Scope) error {
        return convertV1alpha1ToV1(a.(*v1alpha1.Widget), b.(*v1.Widget))
    })
```

**Up (v1alpha1 → v1)** — fill in the richer shape, defaulting what the old
version can't supply:

```go
out.Spec.Dimensions = v1.Dimensions{Width: in.Spec.Size, Height: in.Spec.Size} // square
out.Spec.Material = "unknown"                                                   // new field default
```

**Down (v1 → v1alpha1)** — this is necessarily **lossy**. The old spec has one
`Size`, so `Height` and `Material` have nowhere to go:

```go
out.Spec.Size = in.Spec.Dimensions.Width // keep Width; Height and Material are dropped
```

Lossy down-conversion is normal and expected: it's why you pick a storage
version carefully and why removing an old served version is a deliberate step.

### Running a conversion

`scheme.Convert` runs the registered function:

```go
up := &v1.Widget{}
scheme.Convert(src /* *v1alpha1.Widget */, up, nil)
```

This is the same primitive a **codec** uses internally. In case 13 the codec was
built `WithoutConversion()` because no conversion functions existed; register
them (as here) and a full codec can decode `v1alpha1` bytes straight into a `v1`
object, running `Convert` as part of the decode.

### In-process (client-side) vs. server-side conversion

The conversion in this case is **real**, but it only runs **inside this program**.
It transforms Go structs in your own memory, because this process imported both
version packages, registered the functions into its own scheme, and called
`scheme.Convert`. Its reach ends at the process boundary.

That is different from what the **API server** must do for a multi-version CRD.
The server stores every object in the single **storage version** but has to serve
*any* served version a client asks for:

```
client A ──write v1alpha1──► API server ──convert to v1──► etcd   (storage version)
client B ──read  v1────────► API server ◄──read v1────────┘
client C ──read  v1alpha1──► API server ──convert v1→v1alpha1──► client C
```

So the server itself must convert between versions — on writes and on reads, for
*all* clients. It cannot reuse your `conversion.go`: `kube-apiserver` is a
generic binary that has never heard of `example.com/Widget`, has no Go structs
for it, and cannot import your code. Your functions are compiled into *your*
binary, not the server's.

| | In-process (`scheme.Convert`) | Server-side |
|---|---|---|
| Runs the conversion code | this program | a webhook you host (still your code) |
| Who asks for it | this program, directly | the API server, over HTTPS |
| Scope | objects in this process | every client of the cluster + etcd storage |
| Needs TLS / CA wiring | no | yes |

Both run the *same kind* of conversion function. The difference is **location
and reach** — in-process serves only you; the server-side path makes that
identical logic reachable by the generic API server.

### How conversion is registered server-side

Because the API server can't import your Go code, you expose the same conversion
logic as a **webhook** and point the CRD at it. Registration is a field on the
CRD, `spec.conversion`:

```yaml
spec:
  conversion:
    strategy: Webhook              # ask an external endpoint to convert
    webhook:
      conversionReviewVersions: ["v1"]
      clientConfig:
        service:                   # the webhook runs as a Service in-cluster
          namespace: widget-system
          name: widget-conversion-webhook
          path: /convert
        caBundle: <base64 PEM>     # CA that signed the webhook's TLS cert
```

With this in place, whenever the API server needs to convert a Widget (storing a
written object, or serving a read in a different version) it issues an HTTPS
`POST` to `/convert` with a `ConversionReview` payload:

```
API server ─POST ConversionReview { objects:[...], desiredAPIVersion:"example.com/v1" }─► webhook
API server ◄──── ConversionReview { convertedObjects:[...] } ◄──────────────────────────  webhook
```

The webhook is a small HTTPS server you write. It decodes the `ConversionReview`,
runs **exactly the conversion functions from `conversion.go`** to produce objects
in `desiredAPIVersion`, and writes them back in the response. The server-side
registration is therefore: *(1)* host the logic behind TLS, *(2)* name that
endpoint in `spec.conversion.webhook`, *(3)* give the API server a `caBundle` so
it trusts the endpoint's certificate.

### What controller-gen / conversion-gen generate

In a real project you would not hand-write these. `conversion-gen` produces
`zz_generated.conversion.go` with field-by-field functions, and the registration
boilerplate, from the two type packages. Writing it by hand once makes the
generated output readable.

### Why this case uses `strategy: None`

`widget_crd.yaml` sets `conversion.strategy: None`, so it needs no webhook, no
TLS, and no CA bundle — it runs on a plain cluster. The trade-off: with `None`
the API server does **not** transform the spec. It only relabels
`apiVersion`/`kind` and returns the stored bytes verbatim.

> ⚠️ `strategy: None` is only correct when all versions are **structurally
> identical** (a pure rename). This case's versions are *not* — `v1` restructures
> `size` into `dimensions` and adds `material`. So against a real cluster, GETting
> a stored `v1` object as `v1alpha1` under `None` would return v1-shaped fields
> mislabelled as v1alpha1. The manifest is included only to show how both versions
> are **declared**; the actual conversion in this case is the in-process code in
> `main.go`. A production multi-version CRD with these spec changes would require
> the `Webhook` strategy above.

## References

- [Versions in CustomResourceDefinitions — Kubernetes](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)
- [Webhook conversion — Kubernetes](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion)
- [runtime.Scheme.Convert godoc](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#Scheme.Convert)
- [conversion-gen](https://github.com/kubernetes/code-generator/tree/master/cmd/conversion-gen)
