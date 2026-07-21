/*
Copyright 2024 The Kubernetes Authors.

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
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/utils/cel"
)

// EnvironmentConfig holds configuration for a cel program
type EnvironmentConfig struct {
	EnableResultCache bool

	Now                    func() time.Time
	StartedContainersTotal func(nodeName string) int64
	WorkloadForPod         func(pod *corev1.Pod) *Workload

	ContainerResourceUsage func(resourceName, podNamespace, podName, containerName string) float64
	PodResourceUsage       func(resourceName, podNamespace, podName string) float64
	NodeResourceUsage      func(resourceName, nodeName string) float64

	ContainerResourceCumulativeUsage func(resourceName, podNamespace, podName, containerName string) float64
	PodResourceCumulativeUsage       func(resourceName, podNamespace, podName string) float64
	NodeResourceCumulativeUsage      func(resourceName, nodeName string) float64
}

// NewEnvironment returns a Environment that is able to evaluate node metrics
func NewEnvironment(conf EnvironmentConfig) (*Environment, error) {
	const (
		nowOldName                    = "now"                    // deprecated
		startedContainersTotalOldName = "startedContainersTotal" // deprecated

		nowName                    = "Now"
		startedContainersTotalName = "StartedContainersTotal"

		usageName           = "Usage"
		cumulativeUsageName = "CumulativeUsage"
	)
	types := slices.Clone(cel.DefaultTypes)
	conversions := slices.Clone(cel.DefaultConversions)
	funcs := maps.Clone(cel.DefaultFuncs)
	methods := maps.Clone(cel.FuncsToMethods(cel.DefaultFuncs))
	types = append(types, Workload{})
	methods["SinceSecond"] = append(methods["SinceSecond"], func(workload *Workload) float64 {
		return time.Since(workload.CreationTimestamp.Time).Seconds()
	})

	if conf.Now != nil {
		funcs[nowOldName] = []any{conf.Now}
		funcs[nowName] = []any{conf.Now}
	}

	if conf.ContainerResourceUsage != nil {
		methods[usageName] = append(methods[usageName], func(pod corev1.Pod, resourceName string, containerName string) float64 {
			return conf.ContainerResourceUsage(resourceName, pod.Namespace, pod.Name, containerName)
		})
	}

	if conf.PodResourceUsage != nil {
		methods[usageName] = append(methods[usageName], func(pod corev1.Pod, resourceName string) float64 {
			return conf.PodResourceUsage(resourceName, pod.Namespace, pod.Name)
		})
	}

	if conf.NodeResourceUsage != nil {
		methods[usageName] = append(methods[usageName], func(node corev1.Node, resourceName string) float64 {
			return conf.NodeResourceUsage(resourceName, node.Name)
		})
	}

	if conf.ContainerResourceCumulativeUsage != nil {
		methods[cumulativeUsageName] = append(methods[cumulativeUsageName], func(pod corev1.Pod, resourceName string, containerName string) float64 {
			return conf.ContainerResourceCumulativeUsage(resourceName, pod.Namespace, pod.Name, containerName)
		})
	}

	if conf.PodResourceCumulativeUsage != nil {
		methods[cumulativeUsageName] = append(methods[cumulativeUsageName], func(pod corev1.Pod, resourceName string) float64 {
			return conf.PodResourceCumulativeUsage(resourceName, pod.Namespace, pod.Name)
		})
	}

	if conf.NodeResourceCumulativeUsage != nil {
		methods[cumulativeUsageName] = append(methods[cumulativeUsageName], func(node corev1.Node, resourceName string) float64 {
			return conf.NodeResourceCumulativeUsage(resourceName, node.Name)
		})
	}

	if conf.StartedContainersTotal != nil {
		startedContainersTotal := conf.StartedContainersTotal
		startedContainersTotalByNode := func(node corev1.Node) float64 {
			return float64(conf.StartedContainersTotal(node.Name))
		}
		methods[startedContainersTotalOldName] = append(methods[startedContainersTotalOldName], startedContainersTotal, startedContainersTotalByNode)
		funcs[startedContainersTotalOldName] = append(funcs[startedContainersTotalOldName], startedContainersTotal, startedContainersTotalByNode)

		methods[startedContainersTotalName] = append(methods[startedContainersTotalName], startedContainersTotal, startedContainersTotalByNode)
		funcs[startedContainersTotalName] = append(funcs[startedContainersTotalName], startedContainersTotal, startedContainersTotalByNode)
	}

	env, err := cel.NewEnvironment(cel.EnvironmentConfig{
		Types:       types,
		Conversions: conversions,
		Methods:     methods,
		Funcs:       funcs,
		Vars: map[string]any{
			"node":      corev1.Node{},
			"pod":       corev1.Pod{},
			"container": corev1.Container{},
			"workload":  Workload{},
		},
	})
	if err != nil {
		return nil, err
	}
	e := &Environment{
		env:  env,
		conf: conf,
	}

	if conf.EnableResultCache {
		e.resultCacheVer = new(int64)
	}

	return e, nil
}

// Environment is environment in which cel programs are executed
type Environment struct {
	env *cel.Environment

	conf           EnvironmentConfig
	resultCacheVer *int64
}

// Compile is responsible for compiling a cel program
func (e *Environment) Compile(src string) (*Evaluator, error) {
	program, err := e.env.Compile(src)
	if err != nil {
		return nil, fmt.Errorf("failed to compile metric expression: %w", err)
	}

	evaluator := &Evaluator{
		program:        program,
		latestCacheVer: e.resultCacheVer,
		workloadForPod: e.conf.WorkloadForPod,
	}
	return evaluator, nil
}

// ClearResultCache clears the result cache
func (e *Environment) ClearResultCache() {
	if e.resultCacheVer == nil {
		return
	}
	atomic.AddInt64(e.resultCacheVer, 1)
}

// Evaluator evaluates a cel program
type Evaluator struct {
	program cel.Program

	latestCacheVer *int64
	cacheVer       int64
	workloadForPod func(*corev1.Pod) *Workload

	cache    map[string]cel.Val
	cacheMut sync.Mutex
}

func resultUniqueKey(node *corev1.Node, pod *corev1.Pod, container *corev1.Container, workload *Workload) string {
	tmp := make([]string, 0, 5)
	if node != nil {
		tmp = append(tmp, string(node.UID), node.ResourceVersion)
	}
	if pod != nil {
		tmp = append(tmp, string(pod.UID), pod.ResourceVersion)
	}
	if container != nil {
		tmp = append(tmp, container.Name)
	}
	if workload != nil {
		tmp = append(tmp, workload.UID, fmt.Sprint(workload.Replicas), fmt.Sprint(workload.PodIndex), fmt.Sprint(workload.Version))
	}
	return strings.Join(tmp, "/")
}

func (e *Evaluator) evaluate(ctx context.Context, data Data) (cel.Val, error) {
	if data.Workload == nil && data.Pod != nil {
		if e.workloadForPod != nil {
			data.Workload = e.workloadForPod(data.Pod)
		}
		if data.Workload == nil {
			data.Workload = WorkloadForPod(data.Pod)
		}
	}
	var key string
	if e.latestCacheVer != nil {
		e.cacheMut.Lock()
		defer e.cacheMut.Unlock()

		latestCacheVer := atomic.LoadInt64(e.latestCacheVer)
		if e.cache == nil || latestCacheVer != e.cacheVer {
			e.cache = map[string]cel.Val{}
			e.cacheVer = latestCacheVer
		}

		key = resultUniqueKey(data.Node, data.Pod, data.Container, data.Workload)
		if val, ok := e.cache[key]; ok {
			return val, nil
		}
	}

	refVal, _, err := e.program.ContextEval(ctx, map[string]any{
		"node":      data.Node,
		"pod":       data.Pod,
		"container": data.Container,
		"workload":  data.Workload,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate metric expression: %w", err)
	}
	if key != "" {
		e.cache[key] = refVal
	}
	return refVal, nil
}

// EvaluateFloat64 evaluates a cel program and returns a float64.
func (e *Evaluator) EvaluateFloat64(ctx context.Context, data Data) (float64, error) {
	refVal, err := e.evaluate(ctx, data)
	if err != nil {
		return 0, err
	}

	return cel.AsFloat64(refVal)
}

// EvaluateString evaluates a cel program and returns a string
func (e *Evaluator) EvaluateString(ctx context.Context, data Data) (string, error) {
	refVal, err := e.evaluate(ctx, data)
	if err != nil {
		return "", err
	}

	return cel.AsString(refVal)
}

// Data is a data structure that is passed to the cel program
type Data struct {
	Node      *corev1.Node
	Pod       *corev1.Pod
	Container *corev1.Container
	Workload  *Workload
}

// Workload is the metric-bearing top-level workload for a pod.
type Workload struct {
	APIVersion        string      `json:"apiVersion"`
	Kind              string      `json:"kind"`
	Namespace         string      `json:"namespace"`
	Name              string      `json:"name"`
	UID               string      `json:"uid"`
	Replicas          int64       `json:"replicas"`
	PodIndex          int64       `json:"podIndex"`
	CreationTimestamp metav1.Time `json:"creationTimestamp"`
	Version           int64       `json:"-"`
}

// WorkloadResolver optionally resolves a pod to its metric-bearing workload.
type WorkloadResolver interface {
	ResolveWorkload(pod *corev1.Pod) *Workload
}

// GetCreationTimestamp allows workload.SinceSecond() in CEL expressions.
func (w Workload) GetCreationTimestamp() metav1.Time {
	return w.CreationTimestamp
}

// WorkloadForPod returns the compatibility workload for an unresolved pod.
func WorkloadForPod(pod *corev1.Pod) *Workload {
	return &Workload{
		APIVersion:        corev1.SchemeGroupVersion.String(),
		Kind:              "Pod",
		Namespace:         pod.Namespace,
		Name:              pod.Name,
		UID:               string(pod.UID),
		Replicas:          1,
		CreationTimestamp: pod.CreationTimestamp,
	}
}
