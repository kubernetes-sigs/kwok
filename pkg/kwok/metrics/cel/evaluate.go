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
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/wzshiming/easycel"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeEvaluatorConfig holds configuration for a cel program
type NodeEvaluatorConfig struct {
	Now                    func() time.Time
	StartedContainersTotal func(nodeName string) int64
}

// NewEnvironment returns a MetricEvaluator that is able to evaluate node metrics
func NewEnvironment(conf NodeEvaluatorConfig) (*Environment, error) {
	registry := easycel.NewRegistry("kwok.metric.ext.node",
		easycel.WithTagName("json"),
	)

	e := &Environment{
		registry:       registry,
		conf:           conf,
		cacheEvaluator: make(map[string]*Evaluator),
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
}

func (e *Environment) init() error {
	conversions := []any{
		func(t metav1.Time) types.Timestamp {
			return types.Timestamp{Time: t.Time}
		},
	}
	types := []any{
		corev1.Node{},
		metav1.ObjectMeta{},
	}

	vars := map[string]any{
		"node": corev1.Node{},
	}

	funcs := map[string][]any{}

	methods := map[string][]any{}

	if e.conf.Now != nil {
		funcs["now"] = append(funcs["now"], e.conf.Now)
	} else {
		funcs["now"] = append(funcs["now"], time.Now)
	}

	unixSecond := func(t time.Time) int64 {
		return t.Unix()
	}
	methods["unixSecond"] = append(methods["unixSecond"], unixSecond)
	funcs["unixSecond"] = append(funcs["unixSecond"], unixSecond)

	if e.conf.StartedContainersTotal != nil {
		startedContainersTotal := e.conf.StartedContainersTotal
		startedContainersTotalByNode := func(node corev1.Node) int64 {
			return e.conf.StartedContainersTotal(node.Name)
		}
		methods["startedContainersTotal"] = append(methods["startedContainersTotal"], startedContainersTotal, startedContainersTotalByNode)
		funcs["startedContainersTotal"] = append(funcs["startedContainersTotal"], startedContainersTotal, startedContainersTotalByNode)
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
	if evaluator, ok := e.cacheEvaluator[src]; ok {
		return evaluator, nil
	}
	program, err := e.env.Program(src)
	if err != nil {
		return nil, fmt.Errorf("failed to compile metric expression: %w", err)
	}

	evaluator := &Evaluator{
		program: program,
	}
	e.cacheEvaluator[src] = evaluator
	return evaluator, nil
}

// Evaluator evaluates a cel program
type Evaluator struct {
	program cel.Program
}

// EvaluateFloat64 evaluates a cel program and returns a metric value and returns float64.
func (e *Evaluator) EvaluateFloat64(node *corev1.Node) (float64, error) {
	refVal, _, err := e.program.Eval(map[string]any{
		"node": node,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to evaluate metric expression: %w", err)
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
	default:
		return 0, fmt.Errorf("unsupported metric value type: %T", v)
	}
}

// EvaluateString evaluates a cel program and returns a string
func (e *Evaluator) EvaluateString(node *corev1.Node) (string, error) {
	refVal, _, err := e.program.Eval(map[string]any{
		"node": node,
	})
	if err != nil {
		return "", fmt.Errorf("failed to evaluate metric expression: %w", err)
	}

	v, ok := refVal.(types.String)
	if !ok {
		return "", fmt.Errorf("unsupported metric type: %T", v)
	}
	return string(v), nil
}
