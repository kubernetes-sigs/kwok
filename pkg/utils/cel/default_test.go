package cel

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to compare two maps of functions by their types
func compareFuncMaps(t *testing.T, expected, actual map[string][]any) {
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
func noParams()                 {}
func oneParam(a int)            {}
func twoParams(a int, b string) {}
func notAFunction()             {}

func TestFuncsToMethods_EmptyMap(t *testing.T) {
	input := map[string][]any{}
	expected := map[string][]any{}

	output := FuncsToMethods(input)
	compareFuncMaps(t, expected, output)
}

func TestFuncsToMethods_NoParamsFunction(t *testing.T) {
	input := map[string][]any{
		"FuncNoParams": {noParams},
	}
	expected := map[string][]any{}

	output := FuncsToMethods(input)
	compareFuncMaps(t, expected, output)
}

func TestFuncsToMethods_WithParamsFunctions(t *testing.T) {
	input := map[string][]any{
		"FuncWithOneParam":  {oneParam},
		"FuncWithTwoParams": {twoParams},
	}
	expected := map[string][]any{
		"FuncWithOneParam":  {oneParam},
		"FuncWithTwoParams": {twoParams},
	}

	output := FuncsToMethods(input)
	compareFuncMaps(t, expected, output)
}

func TestFuncsToMethods_MixedEntries(t *testing.T) {
	input := map[string][]any{
		"FuncNoParams":     {noParams},
		"FuncWithOneParam": {oneParam},
		"NotAFunction":     {notAFunction},
	}
	expected := map[string][]any{
		"FuncWithOneParam": {oneParam},
	}

	output := FuncsToMethods(input)
	compareFuncMaps(t, expected, output)
}

func TestFuncsToMethods_MixedFunctionsAndNonFunctions(t *testing.T) {
	input := map[string][]any{
		"FuncWithOneParam": {oneParam},
		"NotAFunction":     {"not a function"},
	}
	expected := map[string][]any{
		"FuncWithOneParam": {oneParam},
	}

	output := FuncsToMethods(input)
	compareFuncMaps(t, expected, output)
}
