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
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwok/controllers"
	"sigs.k8s.io/kwok/pkg/kwok/metrics/cel"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/maps"
)

// UpdateHandler handles updating metrics on request
type UpdateHandler struct {
	controller      *controllers.Controller
	environment     *cel.Environment
	nodeCacheGetter informer.Getter[*corev1.Node]
	podCacheGetter  informer.Getter[*corev1.Pod]

	handler  http.Handler
	registry *prometheus.Registry

	gauges     maps.SyncMap[string, Gauge]
	counters   maps.SyncMap[string, Counter]
	histograms maps.SyncMap[string, Histogram]
}

// UpdateHandlerConfig is configuration for a single node
type UpdateHandlerConfig struct {
	Controller  *controllers.Controller
	Environment *cel.Environment
}

// NewMetricsUpdateHandler creates new metric update handler based on the config
func NewMetricsUpdateHandler(conf UpdateHandlerConfig) *UpdateHandler {
	registry := prometheus.NewRegistry()
	handler := promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer, promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}),
	)

	h := &UpdateHandler{
		controller:      conf.Controller,
		environment:     conf.Environment,
		nodeCacheGetter: conf.Controller.GetNodeCache(),
		podCacheGetter:  conf.Controller.GetPodCache(),
		registry:        registry,
		handler:         handler,
	}
	return h
}

func (h *UpdateHandler) getPodsInfo(nodeName string) ([]log.ObjectRef, bool) {
	podsInfo, ok := h.controller.ListPods(nodeName)
	return podsInfo, ok
}

func (h *UpdateHandler) getOrRegisterGauge(metricConfig *internalversion.MetricConfig, data cel.Data) (Gauge, string, error) {
	key, labels, err := h.createKeyAndLabels(metricConfig, data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to evaluate labels: %w", err)
	}
	val, ok := h.gauges.Load(key)
	if ok {
		return val, key, nil
	}

	val = NewGauge(
		GaugeOpts{
			Name:        metricConfig.Name,
			Help:        metricConfig.Help,
			ConstLabels: labels,
		},
	)
	h.gauges.Store(key, val)
	err = h.registry.Register(val)
	if err != nil {
		return nil, "", fmt.Errorf("failed to register gauge %q: %w", metricConfig.Name, err)
	}
	return val, key, nil
}

func (h *UpdateHandler) getOrRegisterCounter(metricConfig *internalversion.MetricConfig, data cel.Data) (Counter, string, error) {
	key, labels, err := h.createKeyAndLabels(metricConfig, data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to evaluate labels: %w", err)
	}
	val, ok := h.counters.Load(key)
	if ok {
		return val, key, nil
	}

	val = NewCounter(
		CounterOpts{
			Name:        metricConfig.Name,
			Help:        metricConfig.Help,
			ConstLabels: labels,
		},
	)
	h.counters.Store(key, val)
	err = h.registry.Register(val)
	if err != nil {
		return nil, "", fmt.Errorf("failed to register counter %q: %w", metricConfig.Name, err)
	}

	return val, key, nil
}

func (h *UpdateHandler) getOrRegisterHistogram(metricConfig *internalversion.MetricConfig, data cel.Data) (Histogram, string, error) {
	key, labels, err := h.createKeyAndLabels(metricConfig, data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to evaluate labels: %w", err)
	}
	val, ok := h.histograms.Load(key)
	if ok {
		return val, key, nil
	}

	buckets := make([]float64, 0, len(metricConfig.Buckets))
	for _, b := range metricConfig.Buckets {
		if b.Hidden {
			continue
		}
		buckets = append(buckets, b.Le)
	}

	val = NewHistogram(
		HistogramOpts{
			Name:        metricConfig.Name,
			Help:        metricConfig.Help,
			ConstLabels: labels,
			Buckets:     buckets,
		},
	)
	h.histograms.Store(key, val)
	err = h.registry.Register(val)
	if err != nil {
		return nil, "", fmt.Errorf("failed to register histogram %q: %w", metricConfig.Name, err)
	}

	return val, key, nil
}

func (h *UpdateHandler) updateGauge(ctx context.Context, metricConfig *internalversion.MetricConfig, nodeName string) ([]string, error) {
	eval, err := h.environment.Compile(metricConfig.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to compile metric value %s: %w", metricConfig.Value, err)
	}

	logger := log.FromContext(ctx).With("node", nodeName)

	node, ok := h.nodeCacheGetter.Get(nodeName)
	if !ok {
		logger.Warn("node not found")
		return nil, nil
	}
	data := cel.Data{
		Node: node,
	}

	switch metricConfig.Dimension {
	case internalversion.DimensionNode:
		gauge, key, err := h.getOrRegisterGauge(metricConfig, data)
		if err != nil {
			return nil, err
		}

		result, err := eval.EvaluateFloat64(data)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate metric %q: %w", metricConfig.Name, err)
		}
		gauge.Set(result)
		return []string{key}, nil
	case internalversion.DimensionPod:
		pods, ok := h.getPodsInfo(nodeName)
		if !ok {
			logger.Warn("pods not found")
			return nil, nil
		}

		keys := make([]string, 0, len(pods))
		for _, podInfo := range pods {
			pod, ok := h.podCacheGetter.GetWithNamespace(podInfo.Name, podInfo.Namespace)
			if !ok {
				logger.Warn("pod not found", "pod", podInfo)
				continue
			}
			data.Pod = pod
			gauge, key, err := h.getOrRegisterGauge(metricConfig, data)
			if err != nil {
				return nil, err
			}

			result, err := eval.EvaluateFloat64(data)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate metric %q: %w", metricConfig.Name, err)
			}
			gauge.Set(result)
			keys = append(keys, key)
		}
		return keys, nil
	case internalversion.DimensionContainer:
		pods, ok := h.getPodsInfo(nodeName)
		if !ok {
			logger.Warn("pods not found")
			return nil, nil
		}

		keys := make([]string, 0, len(pods))
		for _, podInfo := range pods {
			pod, ok := h.podCacheGetter.GetWithNamespace(podInfo.Name, podInfo.Namespace)
			if !ok {
				logger.Warn("pod not found", "pod", podInfo)
				continue
			}
			data.Pod = pod
			for _, container := range pod.Spec.Containers {
				container := container
				data.Container = &container
				gauge, key, err := h.getOrRegisterGauge(metricConfig, data)
				if err != nil {
					return nil, err
				}
				result, err := eval.EvaluateFloat64(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate metric %q: %w", metricConfig.Name, err)
				}
				gauge.Set(result)
				keys = append(keys, key)
			}
		}
		return keys, nil
	default:
		return nil, fmt.Errorf("unknown dimension %q", metricConfig.Dimension)
	}
}

func (h *UpdateHandler) updateCounter(ctx context.Context, metricConfig *internalversion.MetricConfig, nodeName string) ([]string, error) {
	eval, err := h.environment.Compile(metricConfig.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to compile metric value %s: %w", metricConfig.Value, err)
	}

	logger := log.FromContext(ctx).With("node", nodeName)

	node, ok := h.nodeCacheGetter.Get(nodeName)
	if !ok {
		logger.Warn("node not found")
		return nil, nil
	}
	data := cel.Data{
		Node: node,
	}

	switch metricConfig.Dimension {
	case internalversion.DimensionNode:
		counter, key, err := h.getOrRegisterCounter(metricConfig, data)
		if err != nil {
			return nil, err
		}

		result, err := eval.EvaluateFloat64(data)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate metric %q: %w", metricConfig.Name, err)
		}
		counter.Set(result)
		return []string{key}, nil
	case internalversion.DimensionPod:
		pods, ok := h.getPodsInfo(nodeName)
		if !ok {
			logger.Warn("pods not found")
			return nil, nil
		}

		keys := make([]string, 0, len(pods))
		for _, podInfo := range pods {
			pod, ok := h.podCacheGetter.GetWithNamespace(podInfo.Name, podInfo.Namespace)
			if !ok {
				logger.Warn("pod not found", "pod", podInfo)
				continue
			}
			data.Pod = pod
			counter, key, err := h.getOrRegisterCounter(metricConfig, data)
			if err != nil {
				return nil, err
			}

			result, err := eval.EvaluateFloat64(data)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate metric %q: %w", metricConfig.Name, err)
			}
			counter.Set(result)
			keys = append(keys, key)
		}
		return keys, nil
	case internalversion.DimensionContainer:
		pods, ok := h.getPodsInfo(nodeName)
		if !ok {
			logger.Warn("pods not found")
			return nil, nil
		}

		keys := make([]string, 0, len(pods))
		for _, podInfo := range pods {
			pod, ok := h.podCacheGetter.GetWithNamespace(podInfo.Name, podInfo.Namespace)
			if !ok {
				logger.Warn("pod not found", "pod", podInfo)
				continue
			}
			data.Pod = pod
			for _, container := range pod.Spec.Containers {
				container := container
				data.Container = &container
				counter, key, err := h.getOrRegisterCounter(metricConfig, data)
				if err != nil {
					return nil, err
				}
				result, err := eval.EvaluateFloat64(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate metric %q: %w", metricConfig.Name, err)
				}
				counter.Set(result)
				keys = append(keys, key)
			}
		}
		return keys, nil
	default:
		return nil, fmt.Errorf("unknown dimension %q", metricConfig.Dimension)
	}
}

func (h *UpdateHandler) updateHistogram(ctx context.Context, metricConfig *internalversion.MetricConfig, nodeName string) ([]string, error) {
	logger := log.FromContext(ctx).With("node", nodeName)

	node, ok := h.nodeCacheGetter.Get(nodeName)
	if !ok {
		logger.Warn("node not found")
		return nil, nil
	}
	data := cel.Data{
		Node: node,
	}

	switch metricConfig.Dimension {
	case internalversion.DimensionNode:
		histogram, key, err := h.getOrRegisterHistogram(metricConfig, data)
		if err != nil {
			return nil, err
		}

		for _, b := range metricConfig.Buckets {
			eval, err := h.environment.Compile(b.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to compile program for Le(%v) %q: %w", b.Le, b.Value, err)
			}
			value, err := eval.EvaluateFloat64(data)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate metric with Le(%v): %w", b.Le, err)
			}
			histogram.Set(b.Le, uint64(value))
		}
		return []string{key}, nil
	case internalversion.DimensionPod:
		pods, ok := h.getPodsInfo(nodeName)
		if !ok {
			logger.Warn("pods not found")
			return nil, nil
		}

		keys := make([]string, 0, len(pods))
		for _, podInfo := range pods {
			pod, ok := h.podCacheGetter.GetWithNamespace(podInfo.Name, podInfo.Namespace)
			if !ok {
				logger.Warn("pod not found", "pod", podInfo)
				continue
			}
			data.Pod = pod
			histogram, key, err := h.getOrRegisterHistogram(metricConfig, data)
			if err != nil {
				return nil, err
			}

			for _, b := range metricConfig.Buckets {
				eval, err := h.environment.Compile(b.Value)
				if err != nil {
					return nil, fmt.Errorf("failed to compile program for Le(%v) %q: %w", b.Le, b.Value, err)
				}
				value, err := eval.EvaluateFloat64(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate metric with Le(%v): %w", b.Le, err)
				}
				histogram.Set(b.Le, uint64(value))
			}
			keys = append(keys, key)
		}
		return keys, nil
	case internalversion.DimensionContainer:
		pods, ok := h.getPodsInfo(nodeName)
		if !ok {
			logger.Warn("pods not found")
			return nil, nil
		}

		keys := make([]string, 0, len(pods))
		for _, podInfo := range pods {
			pod, ok := h.podCacheGetter.GetWithNamespace(podInfo.Name, podInfo.Namespace)
			if !ok {
				logger.Warn("pod not found", "pod", podInfo)
				continue
			}
			data.Pod = pod
			for _, container := range pod.Spec.Containers {
				container := container
				data.Container = &container
				histogram, key, err := h.getOrRegisterHistogram(metricConfig, data)
				if err != nil {
					return nil, err
				}

				for _, b := range metricConfig.Buckets {
					eval, err := h.environment.Compile(b.Value)
					if err != nil {
						return nil, fmt.Errorf("failed to compile program for Le(%v) %q: %w", b.Le, b.Value, err)
					}
					value, err := eval.EvaluateFloat64(data)
					if err != nil {
						return nil, fmt.Errorf("failed to evaluate metric with Le(%v): %w", b.Le, err)
					}
					histogram.Set(b.Le, uint64(value))
				}
				keys = append(keys, key)
			}
		}
		return keys, nil
	default:
		return nil, fmt.Errorf("unknown dimension %q", metricConfig.Dimension)
	}
}

func (h *UpdateHandler) updateMetric(ctx context.Context, metricConfig *internalversion.MetricConfig, nodeName string) ([]string, error) {
	switch metricConfig.Kind {
	case internalversion.KindGauge:
		return h.updateGauge(ctx, metricConfig, nodeName)
	case internalversion.KindCounter:
		return h.updateCounter(ctx, metricConfig, nodeName)
	case internalversion.KindHistogram:
		return h.updateHistogram(ctx, metricConfig, nodeName)
	default:
		return nil, fmt.Errorf("unknown metric kind %q", metricConfig.Kind)
	}
}

// createKeyAndLabels creates a key and labels for a metric.
// The key is used to unregister the metric.
// The labels are used to set a value to the metric.
func (h *UpdateHandler) createKeyAndLabels(metricConfig *internalversion.MetricConfig, data cel.Data) (string, prometheus.Labels, error) {
	if len(metricConfig.Labels) == 0 {
		return uniqueKey(metricConfig.Name, metricConfig.Kind, nil), nil, nil
	}

	labels := prometheus.Labels{}
	for _, label := range metricConfig.Labels {
		eval, err := h.environment.Compile(label.Value)
		if err != nil {
			return "", nil, fmt.Errorf("failed to compile metric label value %q: %w", label.Value, err)
		}

		value, err := eval.EvaluateString(data)
		if err != nil {
			return "", nil, fmt.Errorf("failed to evaluate metric label %q: %w", label.Name, err)
		}

		key := label.Name
		labels[key] = value
	}

	return uniqueKey(metricConfig.Name, metricConfig.Kind, labels), labels, nil
}

func uniqueKey(name string, kind internalversion.Kind, labels map[string]string) string {
	builder := strings.Builder{}
	_, _ = builder.WriteString(name)
	_, _ = builder.WriteString("|")
	_, _ = builder.WriteString(string(kind))
	if len(labels) != 0 {
		_, _ = builder.WriteString("|")
		keys := maps.Keys(labels)
		sort.Strings(keys)
		for _, k := range keys {
			v := labels[k]
			_, _ = builder.WriteString(k)
			_, _ = builder.WriteString(":")
			_, _ = builder.WriteString(v)
			_, _ = builder.WriteString(",")
		}
	}
	return builder.String()
}

// Update updates metrics for a node
func (h *UpdateHandler) Update(ctx context.Context, nodeName string, metrics []internalversion.MetricConfig) {
	logger := log.FromContext(ctx)
	has := map[string]struct{}{}
	// Sync metrics
	h.environment.ClearResultCache()
	for _, metric := range metrics {
		metric := metric
		metricName := metric.Name
		keys, err := h.updateMetric(ctx, &metric, nodeName)
		if err != nil {
			logger.Error("failed to update metrics", err,
				"metric", metricName,
				"node", nodeName,
			)
		}
		for _, key := range keys {
			has[key] = struct{}{}
		}
	}

	// Remove old metrics

	for _, key := range h.gauges.Keys() {
		if _, ok := has[key]; !ok {
			old, ok := h.gauges.LoadAndDelete(key)
			if ok {
				h.registry.Unregister(old)
			}
		}
	}
	for _, key := range h.counters.Keys() {
		if _, ok := has[key]; !ok {
			old, ok := h.counters.LoadAndDelete(key)
			if ok {
				h.registry.Unregister(old)
			}
		}
	}
	for _, key := range h.histograms.Keys() {
		if _, ok := has[key]; !ok {
			old, ok := h.histograms.LoadAndDelete(key)
			if ok {
				h.registry.Unregister(old)
			}
		}
	}
}

func (h *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve metrics
	h.handler.ServeHTTP(w, r)
}
