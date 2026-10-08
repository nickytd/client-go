// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 30: Client & workqueue metrics
//
// client-go exposes pluggable metrics hooks. This registers a Prometheus-backed
// workqueue MetricsProvider, drives a queue, and serves /metrics so the
// standard workqueue series (depth, adds, retries, work duration) are scrapeable
// — the same telemetry real controllers export.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/util/workqueue"
)

// promProvider implements workqueue.MetricsProvider backed by Prometheus.
// Each constructor returns a registered collector adapted to the tiny metric
// interfaces workqueue expects.
type promProvider struct{ reg *prometheus.Registry }

func (p promProvider) gauge(name string) workqueue.GaugeMetric {
	g := prometheus.NewGauge(prometheus.GaugeOpts{Subsystem: "workqueue", Name: name})
	p.reg.MustRegister(g)
	return g
}
func (p promProvider) settable(name string) workqueue.SettableGaugeMetric {
	g := prometheus.NewGauge(prometheus.GaugeOpts{Subsystem: "workqueue", Name: name})
	p.reg.MustRegister(g)
	return g
}
func (p promProvider) counter(name string) workqueue.CounterMetric {
	c := prometheus.NewCounter(prometheus.CounterOpts{Subsystem: "workqueue", Name: name})
	p.reg.MustRegister(c)
	return c
}
func (p promProvider) histogram(name string) workqueue.HistogramMetric {
	h := prometheus.NewHistogram(prometheus.HistogramOpts{Subsystem: "workqueue", Name: name})
	p.reg.MustRegister(h)
	return h
}

func (p promProvider) NewDepthMetric(string) workqueue.GaugeMetric      { return p.gauge("depth") }
func (p promProvider) NewAddsMetric(string) workqueue.CounterMetric     { return p.counter("adds_total") }
func (p promProvider) NewLatencyMetric(string) workqueue.HistogramMetric {
	return p.histogram("queue_duration_seconds")
}
func (p promProvider) NewWorkDurationMetric(string) workqueue.HistogramMetric {
	return p.histogram("work_duration_seconds")
}
func (p promProvider) NewUnfinishedWorkSecondsMetric(string) workqueue.SettableGaugeMetric {
	return p.settable("unfinished_work_seconds")
}
func (p promProvider) NewLongestRunningProcessorSecondsMetric(string) workqueue.SettableGaugeMetric {
	return p.settable("longest_running_processor_seconds")
}
func (p promProvider) NewRetriesMetric(string) workqueue.CounterMetric {
	return p.counter("retries_total")
}

func main() {
	reg := prometheus.NewRegistry()
	workqueue.SetProvider(promProvider{reg: reg})

	// A named queue so the provider's constructors fire.
	queue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "demo"})

	// Drive some traffic: add, get, finish — generating adds/depth/duration.
	go func() {
		for i := range 20 {
			queue.Add(fmt.Sprintf("item-%d", i))
		}
		for {
			item, shutdown := queue.Get()
			if shutdown {
				return
			}
			time.Sleep(5 * time.Millisecond) // simulate work
			queue.Done(item)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	slog.Info("serving metrics", "url", "http://localhost:8080/metrics")
	slog.Info("scrape it", "cmd", "curl -s localhost:8080/metrics | grep workqueue")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("error serving metrics", "err", err)
	}
}
