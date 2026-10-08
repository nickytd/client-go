// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 09: Rate-limiting work queue
//
// Demonstrates:
//   - workqueue.NewTypedRateLimitingQueue
//   - workqueue.DefaultTypedControllerRateLimiter (exponential backoff)
//   - Add / Get / Done / Forget cycle
//   - Requeue on error with NumRequeues check
package main

import (
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/util/workqueue"
)

func main() {
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
	defer queue.ShutDown()

	go func() {
		for i := range 5 {
			key := fmt.Sprintf("default/pod-%d", i)
			queue.Add(key)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	for range 5 {
		key, quit := queue.Get()
		if quit {
			return
		}

		err := process(key)
		if err != nil {
			if queue.NumRequeues(key) < 3 {
				slog.Info("requeueing", "key", key, "attempt", queue.NumRequeues(key)+1)
				queue.AddRateLimited(key)
			} else {
				slog.Info("dropping after max retries", "key", key)
				queue.Forget(key)
			}
		} else {
			queue.Forget(key)
		}
		queue.Done(key)
	}
}

func process(key string) error {
	slog.Info("processed", "key", key)
	return nil
}
