/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwok/controllers"
	"sigs.k8s.io/kwok/pkg/kwok/metrics/cel"
	"sigs.k8s.io/kwok/pkg/log"
)

// Constants holding metric kinds.
const (
	KindGauge     = "gauge"
	KindHistogram = "histogram"
	KindCounter   = "counter"
)

// UpdateHandler handles updating metrics on request
type UpdateHandler struct {
	name        string
	metric      *internalversion.Metric
	controller  *controllers.Controller
	environment *cel.Environment

	handler  http.Handler
	registry *prometheus.Registry
	updates  []func() error
}

// UpdateHandlerConfig is configuration for a single node
type UpdateHandlerConfig struct {
	NodeName    string
	Metrics     *internalversion.Metric
	Controller  *controllers.Controller
	Environment *cel.Environment
}

// NewMetricsUpdateHandler creates new metric update handler based on the config
func NewMetricsUpdateHandler(conf UpdateHandlerConfig) (*UpdateHandler, error) {
	registry := prometheus.NewRegistry()
	handler := promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer, promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}),
	)

	h := &UpdateHandler{
		name:        conf.NodeName,
		metric:      conf.Metrics,
		controller:  conf.Controller,
		environment: conf.Environment,
		registry:    registry,
		handler:     handler,
	}
	err := h.init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics update handler: %w", err)
	}
	return h, nil
}

func (h *UpdateHandler) getNodeInfo(nodeName string) (*controllers.NodeInfo, bool) {
	nodeInfo, ok := h.controller.GetNode(nodeName)
	return nodeInfo, ok
}

func (h *UpdateHandler) init() error {
	metric := h.metric

	for _, m := range metric.Spec.Metrics {
		m := m

		preparedLabels, err := h.prepareLabels(&m)
		if err != nil {
			return fmt.Errorf("failed to prepare labels for metric %q: %w", m.Name, err)
		}

		switch m.Kind {
		case KindGauge:
			var val prometheus.Gauge

			registerFunc := func(node *corev1.Node) error {
				labels, err := h.createLabels(preparedLabels, node)
				if err != nil {
					return fmt.Errorf("failed to evaluate labels: %w", err)
				}
				val = prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name:        m.Name,
						Help:        m.Help,
						ConstLabels: labels,
					},
				)

				err = h.registry.Register(val)
				if err != nil {
					return fmt.Errorf("failed to register gauge %q: %w", m.Name, err)
				}
				return nil
			}

			eval, err := h.environment.Compile(m.Value)
			if err != nil {
				return fmt.Errorf("failed to compile metric %q: %w", metric.Name, err)
			}

			var once sync.Once
			registered := false
			updateFunc := func() error {
				nodeInfo, ok := h.getNodeInfo(metric.Name)
				if !ok {
					if !registered {
						return fmt.Errorf("failed to find node %q", metric.Name)
					}
					val.Set(0)
					return nil
				}

				var err error
				once.Do(func() {
					err = registerFunc(nodeInfo.Node)
					if err != nil {
						registered = true
					}
				})
				if err != nil {
					return err
				}

				result, err := eval.EvaluateFloat64(nodeInfo.Node)
				if err != nil {
					return fmt.Errorf("failed to evaluate metric %q: %w", m.Name, err)
				}
				val.Set(result)
				return nil
			}
			h.updates = append(h.updates, updateFunc)
		case KindCounter:
			counter := atomic.Int64{}

			registerFunc := func(node *corev1.Node) error {
				labels, err := h.createLabels(preparedLabels, node)
				if err != nil {
					return fmt.Errorf("failed to evaluate labels: %w", err)
				}
				val := prometheus.NewCounterFunc(
					prometheus.CounterOpts{
						Name:        m.Name,
						Help:        m.Help,
						ConstLabels: labels,
					},
					func() float64 {
						return float64(counter.Load())
					},
				)

				err = h.registry.Register(val)
				if err != nil {
					return fmt.Errorf("failed to register counter %q: %w", m.Name, err)
				}
				return nil
			}

			eval, err := h.environment.Compile(m.Value)
			if err != nil {
				return fmt.Errorf("failed to compile metric %q: %w", metric.Name, err)
			}

			var once sync.Once
			registered := false
			updateFunc := func() error {
				nodeInfo, ok := h.getNodeInfo(metric.Name)
				if !ok {
					if !registered {
						return fmt.Errorf("failed to find node %q", metric.Name)
					}

					counter.Store(0)
					return nil
				}

				var err error
				once.Do(func() {
					err = registerFunc(nodeInfo.Node)
					if err != nil {
						registered = true
					}
				})
				if err != nil {
					return err
				}

				result, err := eval.EvaluateFloat64(nodeInfo.Node)
				if err != nil {
					return fmt.Errorf("failed to evaluate metric %q: %w", m.Name, err)
				}
				counter.Store(int64(result))
				return nil
			}
			h.updates = append(h.updates, updateFunc)
		case KindHistogram:
			var his *Histogram
			registerFunc := func(node *corev1.Node) error {
				labels, err := h.createLabels(preparedLabels, node)
				if err != nil {
					return fmt.Errorf("failed to evaluate labels: %w", err)
				}

				buckets := make([]float64, 0, len(m.Buckets))
				for _, b := range m.Buckets {
					if b.Hidden {
						continue
					}
					buckets = append(buckets, b.Le)
				}

				his = NewHistogram(
					HistogramOpts{
						Name:        m.Name,
						Help:        m.Help,
						ConstLabels: labels,
						Buckets:     buckets,
					})

				err = h.registry.Register(his)
				if err != nil {
					return fmt.Errorf("failed to register histogram %q: %w", m.Name, err)
				}
				return nil
			}

			programs := make(map[float64]*cel.Evaluator, len(m.Buckets))
			for _, b := range m.Buckets {
				eval, err := h.environment.Compile(b.Value)
				if err != nil {
					return fmt.Errorf("failed to compile program for Le(%v) %q: %w", b.Le, b.Value, err)
				}

				programs[b.Le] = eval
			}

			var once sync.Once
			registered := false
			updateFunc := func() error {
				nodeInfo, ok := h.getNodeInfo(metric.Name)
				if !ok {
					if !registered {
						return fmt.Errorf("failed to find node %q", metric.Name)
					}

					for _, b := range m.Buckets {
						his.Set(b.Le, 0)
					}

					return nil
				}

				var err error
				once.Do(func() {
					err = registerFunc(nodeInfo.Node)
					if err != nil {
						registered = true
					}
				})
				if err != nil {
					return err
				}

				for _, b := range m.Buckets {
					prog, ok := programs[b.Le]
					if !ok {
						return fmt.Errorf("failed to find program for metric %v", b.Le)
					}
					value, err := prog.EvaluateFloat64(nodeInfo.Node)
					if err != nil {
						return fmt.Errorf("failed to evaluate metric with Le(%v): %w", b.Le, err)
					}
					his.Set(b.Le, uint64(value))
				}
				return nil
			}
			h.updates = append(h.updates, updateFunc)
		}
	}

	return nil
}

func (h *UpdateHandler) prepareLabels(m *internalversion.MetricConfig) (map[string]*cel.Evaluator, error) {
	var labels map[string]*cel.Evaluator
	if len(m.Labels) > 0 {
		labels = make(map[string]*cel.Evaluator, len(m.Labels))
	}

	for _, lbl := range m.Labels {
		eval, err := h.environment.Compile(lbl.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to compile metric label value %q: %w", lbl.Value, err)
		}
		labels[lbl.Name] = eval
	}

	return labels, nil
}

func (h *UpdateHandler) createLabels(labels map[string]*cel.Evaluator, node *corev1.Node) (prometheus.Labels, error) {
	var constLabels prometheus.Labels
	if labels != nil {
		constLabels = make(prometheus.Labels, len(labels))
	}
	for name, eval := range labels {
		var err error
		constLabels[name], err = eval.EvaluateString(node)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate metric label %q for node \"%s/%s\": %w", name, node.Namespace, node.Name, err)
		}
	}
	return constLabels, nil
}

func (h *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.FromContext(r.Context())
	// Update metrics
	for _, u := range h.updates {
		err := u()
		if err != nil {
			logger.Error("failed to update metrics", err)
		}
	}

	// Serve metrics
	h.handler.ServeHTTP(w, r)
}
