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
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// TestNewEnvironment tests the NewEnvironment function.
func TestNewEnvironment(t *testing.T) {
	// Mock configuration
	conf := EnvironmentConfig{
		Conversions: []interface{}{},
		Types:       []interface{}{},
		Vars:        map[string]interface{}{},
		Funcs:       map[string][]interface{}{},
		Methods:     map[string][]interface{}{},
	}

	// Create a new environment
	env, err := NewEnvironment(conf)
	if err != nil {
		t.Fatalf("Error creating new environment: %v", err)
	}

	// Check if the environment is created successfully
	if env == nil {
		t.Fatal("Expected non-nil environment, got nil")
	}
}

// TestCompile tests the Compile function.
func TestCompile(t *testing.T) {
	// Mock configuration
	conf := EnvironmentConfig{
		Conversions: []interface{}{},
		Types:       []interface{}{},
		Vars:        map[string]interface{}{},
		Funcs:       map[string][]interface{}{},
		Methods:     map[string][]interface{}{},
	}

	// Create a new environment
	env, err := NewEnvironment(conf)
	if err != nil {
		t.Fatalf("Error creating new environment: %v", err)
	}

	// Compile a simple expression
	expression := "1 + 2"
	program, err := env.Compile(expression)
	if err != nil {
		t.Fatalf("Error compiling expression '%s': %v", expression, err)
	}

	// Check if the program is compiled successfully
	if program == nil {
		t.Fatalf("Expected non-nil program, got nil")
	}
}

// TestAsFloat64 tests the AsFloat64 function.
func TestAsFloat64(t *testing.T) {
	// Test with supported types
	testCases := []struct {
		input    ref.Val
		expected float64
	}{
		{types.Double(3.14), 3.14},
		{types.Int(10), 10},
		{types.Uint(5), 5},
		{types.Bool(true), 1},
	}

	for _, tc := range testCases {
		result, err := AsFloat64(tc.input)
		if err != nil {
			t.Errorf("Error converting %T to float64: %v", tc.input, err)
		}

		if result != tc.expected {
			t.Errorf("Expected %T to be converted to %f, got %f", tc.input, tc.expected, result)
		}
	}

	// Test with unsupported type
	_, err := AsFloat64(types.String("test"))
	if err == nil {
		t.Error("Expected an error for unsupported type conversion")
	}
}

// TestAsString tests the AsString function.
func TestAsString(t *testing.T) {
	// Test with supported type
	input := types.String("test")
	expected := "test"

	result, err := AsString(input)
	if err != nil {
		t.Errorf("Error converting %T to string: %v", input, err)
	}

	if result != expected {
		t.Errorf("Expected %T to be converted to '%s', got '%s'", input, expected, result)
	}

	// Test with unsupported type
	_, err = AsString(types.Int(10))
	if err == nil {
		t.Error("Expected an error for unsupported type conversion")
	}
}
