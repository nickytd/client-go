# Case 28 — Watch Bookmarks

See also [`05-list-watch`](../05-list-watch)

## What this covers

Using watch bookmarks to resume watches from a safe point after disruptions, avoiding "resourceVersion too old" errors when reconnecting to the API server.

## Key concepts

| Concept | Usage |
|---------|-------|
| `ListOptions.ResourceVersion` | Resume point for a new watch; obtained from `List()` or a bookmark event |
| `ListOptions.AllowWatchBookmarks` | Request the server to send periodic `Bookmark` events |
| `watch.Bookmark` event | Server-sent event carrying only an up-to-date `resourceVersion` |
| Bookmark payload | Contains only metadata; no object data |
| Avoiding 410 Gone | Persist the latest bookmark to reconnect without "too old" errors |

## How to run

```bash
cd 28-watch-bookmarks
go run ./...
```

Requires a running Kubernetes cluster and valid `KUBECONFIG` (or `~/.kube/config`).

## Expected output

The program establishes a watch on ConfigMaps in the `default` namespace, then creates and deletes 3 test ConfigMaps and idles until a bookmark arrives (press Ctrl-C to stop):

```
watching resourceVersion=149138 bookmarks=true
watching (Ctrl-C to stop)
event type=ADDED name=bookmark-demo-0 resourceVersion=149139
event type=DELETED name=bookmark-demo-0 resourceVersion=149140
...
churn done; idling until a bookmark arrives (Ctrl-C to stop)
bookmark (safe resume point) resourceVersion=149232
```

**Note:** Bookmark cadence is cluster-dependent and variable (observed anywhere from ~1 to ~10 minutes), so the `bookmark` line may take a while to appear. The resource-version resume mechanic is the teaching point regardless. Press Ctrl-C to stop — the watch streams over a long-lived HTTP connection, and each read blocks until the server sends the next event.

## Key files

- `main.go` — Sets up kubeconfig, creates a watch with `AllowWatchBookmarks: true`, generates test events, and logs `watch.Bookmark` events as they arrive.
