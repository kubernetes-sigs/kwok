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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to compare two maps of functions by their types
func assertFuncMapsEqual(t *testing.T, expected, actual map[string][]any) {
	for key, expFuncs := range expected {
		actFuncs, ok := actual[key]
		assert.True(t, ok, "key %v not found in actual map", key)
		assert.Equal(t, len(expFuncs), len(actFuncs), "length mismatch for key %v", key)
		for i, expFunc := range expFuncs {
			actFunc := actFuncs[i]
			assert.Equal(t, reflect.TypeOf(expFunc), reflect.TypeOf(actFunc), "type mismatch for key %v at index %v", key, i)
		}
	}
	assert.Equal(t, len(expected), len(actual), "number of keys mismatch")
}

// Sample functions for testing
func funcNoParams()                 {}
func funcOneParam(a int)            {}
func funcTwoParams(a int, b string) {}
func funcNotAFunction()             {}

func TestFilterFuncsByParams_EmptyMap(t *testing.T) {
	input := map[string][]any{}
	expected := map[string][]any{}

	output := FuncsToMethods(input)
	assertFuncMapsEqual(t, expected, output)
}

func TestFilterFuncsByParams_NoParamsFunction(t *testing.T) {
	input := map[string][]any{
		"FuncNoParams": {funcNoParams},
	}
	expected := map[string][]any{}

	output := FuncsToMethods(input)
	assertFuncMapsEqual(t, expected, output)
}

func TestFilterFuncsByParams_WithParamsFunctions(t *testing.T) {
	input := map[string][]any{
		"FuncWithOneParam":  {funcOneParam},
		"FuncWithTwoParams": {funcTwoParams},
	}
	expected := map[string][]any{
		"FuncWithOneParam":  {funcOneParam},
		"FuncWithTwoParams": {funcTwoParams},
	}

	output := FuncsToMethods(input)
	assertFuncMapsEqual(t, expected, output)
}

func TestFilterFuncsByParams_MixedEntries(t *testing.T) {
	input := map[string][]any{
		"FuncNoParams":     {funcNoParams},
		"FuncWithOneParam": {funcOneParam},
		"NotAFunction":     {funcNotAFunction},
	}
	expected := map[string][]any{
		"FuncWithOneParam": {funcOneParam},
	}

	output := FuncsToMethods(input)
	assertFuncMapsEqual(t, expected, output)
}

func TestFilterFuncsByParams_MixedFunctionsAndNonFunctions(t *testing.T) {
	input := map[string][]any{
		"FuncWithOneParam": {funcOneParam},
		"NotAFunction":     {"not a function"},
	}
	expected := map[string][]any{
		"FuncWithOneParam": {funcOneParam},
	}

	output := FuncsToMethods(input)
	assertFuncMapsEqual(t, expected, output)
}
