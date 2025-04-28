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

package cel

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/wzshiming/easycel"
)

// EnvironmentConfig holds configuration for a cel program
type EnvironmentConfig struct {
	Conversions []any
	Types       []any
	Vars        map[string]any
	Funcs       map[string][]any
	Methods     map[string][]any
}

// NewEnvironment returns a Environment with the given configuration
func NewEnvironment(conf EnvironmentConfig) (*Environment, error) {
	registry := easycel.NewRegistry("kwok.ext",
		easycel.WithTagName("json"),
	)

	for _, convert := range conf.Conversions {
		err := registry.RegisterConversion(convert)
		if err != nil {
			return nil, fmt.Errorf("failed to register convert %T: %w", convert, err)
		}
	}
	for _, typ := range conf.Types {
		err := registry.RegisterType(typ)
		if err != nil {
			return nil, fmt.Errorf("failed to register type %T: %w", typ, err)
		}
	}
	for name, val := range conf.Vars {
		err := registry.RegisterVariable(name, val)
		if err != nil {
			return nil, fmt.Errorf("failed to register variable %s: %w", name, err)
		}
	}
	for name, list := range conf.Funcs {
		for _, fun := range list {
			err := registry.RegisterFunction(name, fun)
			if err != nil {
				return nil, fmt.Errorf("failed to register function %s: %w", name, err)
			}
		}
	}
	for name, list := range conf.Methods {
		for _, fun := range list {
			err := registry.RegisterMethod(name, fun)
			if err != nil {
				return nil, fmt.Errorf("failed to register method %s: %w", name, err)
			}
		}
	}
	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	e := &Environment{
		env:          env,
		cacheProgram: map[string]cel.Program{},
	}
	return e, nil
}

// Environment is environment in which cel programs are executed
type Environment struct {
	env          *easycel.Environment
	cacheProgram map[string]cel.Program
	cacheMut     sync.Mutex
}

// Compile is responsible for compiling a cel program
func (e *Environment) Compile(src string) (cel.Program, error) {
	e.cacheMut.Lock()
	defer e.cacheMut.Unlock()

	if program, ok := e.cacheProgram[src]; ok {
		return program, nil
	}

	program, err := e.env.Program(src)
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}

	e.cacheProgram[src] = program

	return program, nil
}

// AsFloat64 returns the float64 value of a ref.Val
func AsFloat64(refVal ref.Val) (float64, error) {
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
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// AsString returns the string value of a ref.Val
func AsString(refVal ref.Val) (string, error) {
	v, ok := refVal.(types.String)
	if !ok {
		return "", fmt.Errorf("unsupported type: %T", v)
	}
	return string(v), nil
}

// AsBool returns the bool value of a ref.Val
func AsBool(refVal ref.Val) (bool, error) {
	v, ok := refVal.(types.Bool)
	if !ok {
		return false, fmt.Errorf("unsupported type: %T", v)
	}
	return bool(v), nil
}

// AsDuration returns the time.Duration value of a ref.Val
func AsDuration(refVal ref.Val) (time.Duration, error) {
	switch v := refVal.(type) {
	case types.String:
		return time.ParseDuration(string(v))
	case types.Duration:
		return v.Duration, nil
	case types.Int:
		return time.Duration(v), nil
	case types.Double:
		return time.Duration(v), nil
	case types.Uint:
		return time.Duration(v), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// AsInt returns the int64 value of a ref.Val
func AsInt(refVal ref.Val) (int64, error) {
	switch v := refVal.(type) {
	case types.Duration:
		return int64(v.Duration), nil
	case types.Int:
		return int64(v), nil
	case types.Double:
		return int64(v), nil
	case types.Uint:
		return int64(v), nil
	case types.Bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}
