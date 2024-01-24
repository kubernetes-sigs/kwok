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

package cel

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/wzshiming/easycel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeEvaluatorConfig holds configuration for a cel program
type NodeEvaluatorConfig struct {
	EnableEvaluatorCache bool
	EnableResultCache    bool

	Now                    func() time.Time
	StartedContainersTotal func(nodeName string) int64

	ContainerResourceUsage func(resourceName, podNamespace, podName, containerName string) float64
	PodResourceUsage       func(resourceName, podNamespace, podName string) float64
	NodeResourceUsage      func(resourceName, nodeName string) float64

	ContainerResourceCumulativeUsage func(resourceName, podNamespace, podName, containerName string) float64
	PodResourceCumulativeUsage       func(resourceName, podNamespace, podName string) float64
	NodeResourceCumulativeUsage      func(resourceName, nodeName string) float64
}

// NewEnvironment returns a MetricEvaluator that is able to evaluate node metrics
func NewEnvironment(conf NodeEvaluatorConfig) (*Environment, error) {
	registry := easycel.NewRegistry("kwok.metric.ext.node",
		easycel.WithTagName("json"),
	)

	e := &Environment{
		registry: registry,
		conf:     conf,
	}

	if conf.EnableEvaluatorCache {
		e.cacheEvaluator = map[string]*Evaluator{}
	}

	if conf.EnableResultCache {
		e.resultCacheVer = new(int64)
	}

	err := e.init()
	if err != nil {
		return nil, err
	}

	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}
	e.env = env

	return e, nil
}

// Environment is environment in which cel programs are executed
type Environment struct {
	registry       *easycel.Registry
	env            *easycel.Environment
	conf           NodeEvaluatorConfig
	cacheEvaluator map[string]*Evaluator
	cacheMut       sync.Mutex
	resultCacheVer *int64
}

func (e *Environment) init() error {
	conversions := []any{
		func(t metav1.Time) types.Timestamp {
			return types.Timestamp{Time: t.Time}
		},
		func(t *metav1.Time) types.Timestamp {
			if t == nil {
				return types.Timestamp{}
			}
			return types.Timestamp{Time: t.Time}
		},
		func(t metav1.Duration) types.Duration {
			return types.Duration{Duration: t.Duration}
		},
		func(t *metav1.Duration) types.Duration {
			if t == nil {
				return types.Duration{}
			}
			return types.Duration{Duration: t.Duration}
		},
		func(t resource.Quantity) Quantity {
			return NewQuantity(&t)
		},
		NewResourceList,
	}

	types := []any{
		corev1.Node{},
		corev1.NodeSpec{},
		corev1.NodeStatus{},
		corev1.Pod{},
		corev1.PodSpec{},
		corev1.ResourceRequirements{},
		corev1.PodStatus{},
		corev1.Container{},
		metav1.ObjectMeta{},
		Quantity{},
		ResourceList{},
	}

	vars := map[string]any{
		"node":      corev1.Node{},
		"pod":       corev1.Pod{},
		"container": corev1.Container{},
	}

	funcs := map[string][]any{}

	methods := map[string][]any{}

	const (
		nowOldName                    = "now"                    // deprecated
		startedContainersTotalOldName = "startedContainersTotal" // deprecated

		nowName                    = "Now"
		startedContainersTotalName = "StartedContainersTotal"
		mathRandName               = "Rand"
		sinceSecondName            = "SinceSecond"
		unixSecondName             = "UnixSecond"

		quantityName = "Quantity"

		usageName           = "Usage"
		cumulativeUsageName = "CumulativeUsage"
	)
	if e.conf.Now != nil {
		funcs[nowOldName] = append(funcs[nowOldName], e.conf.Now)
		funcs[nowName] = append(funcs[nowName], e.conf.Now)
	} else {
		funcs[nowOldName] = append(funcs[nowOldName], timeNow)
		funcs[nowName] = append(funcs[nowName], timeNow)
	}

	funcs[mathRandName] = append(funcs[mathRandName], mathRand)

	methods[sinceSecondName] = append(methods[sinceSecondName], sinceSecond[*corev1.Node], sinceSecond[*corev1.Pod])
	funcs[sinceSecondName] = append(funcs[sinceSecondName], sinceSecond[*corev1.Node], sinceSecond[*corev1.Pod])

	methods[unixSecondName] = append(methods[unixSecondName], unixSecond)
	funcs[unixSecondName] = append(funcs[unixSecondName], unixSecond)

	funcs[quantityName] = append(funcs[quantityName], NewQuantityFromString)

	if e.conf.ContainerResourceUsage != nil {
		methods[usageName] = append(methods[usageName], func(pod corev1.Pod, resourceName string, containerName string) float64 {
			return e.conf.ContainerResourceUsage(resourceName, pod.Namespace, pod.Name, containerName)
		})
	}

	if e.conf.PodResourceUsage != nil {
		methods[usageName] = append(methods[usageName], func(pod corev1.Pod, resourceName string) float64 {
			return e.conf.PodResourceUsage(resourceName, pod.Namespace, pod.Name)
		})
	}

	if e.conf.NodeResourceUsage != nil {
		methods[usageName] = append(methods[usageName], func(node corev1.Node, resourceName string) float64 {
			return e.conf.NodeResourceUsage(resourceName, node.Name)
		})
	}

	if e.conf.ContainerResourceCumulativeUsage != nil {
		methods[cumulativeUsageName] = append(methods[cumulativeUsageName], func(pod corev1.Pod, resourceName string, containerName string) float64 {
			return e.conf.ContainerResourceCumulativeUsage(resourceName, pod.Namespace, pod.Name, containerName)
		})
	}

	if e.conf.PodResourceCumulativeUsage != nil {
		methods[cumulativeUsageName] = append(methods[cumulativeUsageName], func(pod corev1.Pod, resourceName string) float64 {
			return e.conf.PodResourceCumulativeUsage(resourceName, pod.Namespace, pod.Name)
		})
	}

	if e.conf.NodeResourceCumulativeUsage != nil {
		methods[cumulativeUsageName] = append(methods[cumulativeUsageName], func(node corev1.Node, resourceName string) float64 {
			return e.conf.NodeResourceCumulativeUsage(resourceName, node.Name)
		})
	}

	if e.conf.StartedContainersTotal != nil {
		startedContainersTotal := e.conf.StartedContainersTotal
		startedContainersTotalByNode := func(node corev1.Node) float64 {
			return float64(e.conf.StartedContainersTotal(node.Name))
		}
		methods[startedContainersTotalOldName] = append(methods[startedContainersTotalOldName], startedContainersTotal, startedContainersTotalByNode)
		funcs[startedContainersTotalOldName] = append(funcs[startedContainersTotalOldName], startedContainersTotal, startedContainersTotalByNode)

		methods[startedContainersTotalName] = append(methods[startedContainersTotalName], startedContainersTotal, startedContainersTotalByNode)
		funcs[startedContainersTotalName] = append(funcs[startedContainersTotalName], startedContainersTotal, startedContainersTotalByNode)
	}

	for _, convert := range conversions {
		err := e.registry.RegisterConversion(convert)
		if err != nil {
			return fmt.Errorf("failed to register convert %T: %w", convert, err)
		}
	}
	for _, typ := range types {
		err := e.registry.RegisterType(typ)
		if err != nil {
			return fmt.Errorf("failed to register type %T: %w", typ, err)
		}
	}
	for name, val := range vars {
		err := e.registry.RegisterVariable(name, val)
		if err != nil {
			return fmt.Errorf("failed to register variable %s: %w", name, err)
		}
	}
	for name, list := range funcs {
		for _, fun := range list {
			err := e.registry.RegisterFunction(name, fun)
			if err != nil {
				return fmt.Errorf("failed to register function %s: %w", name, err)
			}
		}
	}
	for name, list := range methods {
		for _, fun := range list {
			err := e.registry.RegisterMethod(name, fun)
			if err != nil {
				return fmt.Errorf("failed to register method %s: %w", name, err)
			}
		}
	}

	return nil
}

// Compile is responsible for compiling a cel program
func (e *Environment) Compile(src string) (*Evaluator, error) {
	if e.cacheEvaluator != nil {
		e.cacheMut.Lock()
		defer e.cacheMut.Unlock()

		if evaluator, ok := e.cacheEvaluator[src]; ok {
			return evaluator, nil
		}
	}
	program, err := e.env.Program(src)
	if err != nil {
		return nil, fmt.Errorf("failed to compile metric expression: %w", err)
	}

	evaluator := &Evaluator{
		latestCacheVer: e.resultCacheVer,
		program:        program,
	}
	if e.cacheEvaluator != nil {
		e.cacheEvaluator[src] = evaluator
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
	latestCacheVer *int64
	cacheVer       int64

	cache    map[string]ref.Val
	cacheMut sync.Mutex
	program  cel.Program
}

func resultUniqueKey(node *corev1.Node, pod *corev1.Pod, container *corev1.Container) string {
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
	return strings.Join(tmp, "/")
}

func (e *Evaluator) evaluate(data Data) (ref.Val, error) {
	var key string
	if e.latestCacheVer != nil {
		e.cacheMut.Lock()
		defer e.cacheMut.Unlock()

		if e.cache == nil || *e.latestCacheVer != e.cacheVer {
			e.cache = map[string]ref.Val{}
			e.cacheVer = *e.latestCacheVer
		}

		key = resultUniqueKey(data.Node, data.Pod, data.Container)
		if val, ok := e.cache[key]; ok {
			return val, nil
		}
	}
	refVal, _, err := e.program.Eval(map[string]any{
		"node":      data.Node,
		"pod":       data.Pod,
		"container": data.Container,
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
func (e *Evaluator) EvaluateFloat64(data Data) (float64, error) {
	refVal, err := e.evaluate(data)
	if err != nil {
		return 0, err
	}

	switch v := refVal.(type) {
	case types.Duration:
		return float64(v.Duration), nil
	case types.Int:
		return float64(v), nil
	case types.Double:
		return float64(v), nil
	case types.Uint:
		return float64(v), nil
	case types.Bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case Quantity:
		return v.Quantity.AsApproximateFloat64(), nil
	default:
		return 0, fmt.Errorf("unsupported metric value type: %T", v)
	}
}

// EvaluateString evaluates a cel program and returns a string
func (e *Evaluator) EvaluateString(data Data) (string, error) {
	refVal, err := e.evaluate(data)
	if err != nil {
		return "", err
	}

	v, ok := refVal.(types.String)
	if !ok {
		return "", fmt.Errorf("unsupported metric type: %T", v)
	}
	return string(v), nil
}

// Data is a data structure that is passed to the cel program
type Data struct {
	Node      *corev1.Node
	Pod       *corev1.Pod
	Container *corev1.Container
}
