# Case 26 — List Pagination

## What this covers

This case demonstrates how Kubernetes API pagination works. When listing large numbers of resources, the API can chunk responses using `Limit` and `Continue` tokens to control memory usage on both the client and server.

## Key concepts

| Concept | Explanation |
|---------|-------------|
| `ListOptions.Limit` | Max items per response page |
| `ListOptions.Continue` | Opaque token to fetch the next page |
| `ListMeta.Continue` | Token present in response when more items remain |
| Loop until empty | Iterate until `Continue` is empty |
| Bounded memory | Process one page at a time instead of loading all items |

## How to run

```bash
# Requires a running Kubernetes cluster and valid kubeconfig
cd 26-pagination
go run ./...
```

## Expected output

The program creates 7 ConfigMaps with label `demo=pagination`, then lists them with a page size of 3. Expected output:

```
page 1: 3 items, continue="eyJv..."
page 2: 3 items, continue="eyJv..."
page 3: 1 items, continue=""
done: 7 items across 3 pages
```

The `continue` token is truncated to the first 12 characters for readability.

## Key files

- `main.go` — seed ConfigMaps, iterate through pages with `Limit` and `Continue`, cleanup
