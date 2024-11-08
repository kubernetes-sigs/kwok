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
	"maps"
	"math"
	"testing"
	"time"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"k8s.io/apimachinery/pkg/api/resource"
)

var expectedTime = time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)

func TestQuantityCalculation(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "parse tebibyte quantity",
			expression: `Quantity("1Ti")`,
			expected:   `1Ti`,
		},
		{
			name:       "add tebibyte",
			expression: `Quantity("1Ti") + Quantity("1Ti")`,
			expected:   `2Ti`,
		},
		{
			name:       "subtract gibibyte",
			expression: `Quantity("1Ti") - Quantity("512Gi")`,
			expected:   `512Gi`,
		},
		{
			name:       "negate tebibyte",
			expression: `-Quantity("1Ti")`,
			expected:   `-1Ti`,
		},
		{
			name:       "tebibyte multiply int",
			expression: `Quantity("1Ti") * 2`,
			expected:   `2Ti`,
		},
		{
			name:       "tebibyte multiply uint",
			expression: `Quantity("1Ti") * 2u`,
			expected:   `2Ti`,
		},
		{
			name:       "tebibyte multiply float",
			expression: `Quantity("1Ti") * 0.5`,
			expected:   `512Gi`,
		},
		{
			name:       "tebibyte divide int",
			expression: `Quantity("1Ti") / 2`,
			expected:   `512Gi`,
		},
		{
			name:       "tebibyte divide uint",
			expression: `Quantity("1Ti") / 2u`,
			expected:   `512Gi`,
		},
		{
			name:       "tebibyte divide float",
			expression: `Quantity("1Ti") / 2.0`,
			expected:   `512Gi`,
		},
		{
			name:       "parse millicpu quantity",
			expression: `Quantity("1m")`,
			expected:   `1m`,
		},
		{
			name:       "millicpu multiply int",
			expression: `Quantity("1m") * 2`,
			expected:   `2m`,
		},
		{
			name:       "millicpu multiply float",
			expression: `Quantity("1m") * 1.0`,
			expected:   `1m`,
		},
		{
			name:       "millicpu floor to zero",
			expression: `Quantity("1m") * 0.5`,
			expected:   `0`,
		},
		{
			name:       "millicpu divide int",
			expression: `Quantity("5m") / 2`,
			expected:   `2m`,
		},
		{
			name:       "millicpu divide float",
			expression: `Quantity("1m") / 0.1`,
			expected:   `10m`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluateAndCompareToQuantity(t, tt.expression, tt.expected)
		})
	}
}

func TestQuantityCalculationError(t *testing.T) {
	tests := []struct {
		name       string
		expression string
	}{
		{
			name:       "millicpu add invalid int",
			expression: `Quantity("1m") + 1`,
		},
		{
			name:       "millicpu subtract invalid string",
			expression: `Quantity("1m") - "1"`,
		},
		{
			name:       "millicpu multiply invalid string",
			expression: `Quantity("1m") * "2.0"`,
		},
		{
			name:       "millicpu divide invalid string",
			expression: `Quantity("5m") / "2"`,
		},
		{
			name:       "millicpu divide by zero int",
			expression: `Quantity("1m") / 0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluateAndShouldError(t, tt.expression)
		})
	}
}

func TestQuantityComparison(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{
			name:       "compare tebibyte",
			expression: `Quantity("1Ti") == Quantity("1Ti")`,
			expected:   true,
		},
		{
			name:       "compare tebibyte to gibibyte",
			expression: `Quantity("1Ti") == Quantity("1024Gi")`,
			expected:   true,
		},
		{
			name:       "compare tebibyte to gibibyte",
			expression: `Quantity("1") != Quantity("1000m")`,
			expected:   false,
		},
		{
			name:       "compare tebibyte to gibibyte",
			expression: `Quantity("1Ti") > Quantity("1024Gi")`,
			expected:   false,
		},
		{
			name:       "compare tebibyte to gibibyte",
			expression: `Quantity("1Ti") < Quantity("1024Gi")`,
			expected:   false,
		},
		{
			name:       "compare tebibyte to gibibyte",
			expression: `Quantity("1Ti") >= Quantity("1024Gi")`,
			expected:   true,
		},
		{
			name:       "compare tebibyte to gibibyte",
			expression: `Quantity("1Ti") <= Quantity("1024Gi")`,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluateAndCompareToBool(t, tt.expression, tt.expected)
		})
	}
}

func evaluateAndCompareToQuantity(t *testing.T, expression, expectedStr string) {
	refVal, err := runExpression(t, expression)
	if err != nil {
		t.Errorf("failed to run CEL expression: %v", err)
		return
	}

	actual, err := AsFloat64(refVal)
	if err != nil {
		t.Errorf("failed to convert to float64: %v", err)
	}

	expectedQuantity := resource.MustParse(expectedStr)
	expected := expectedQuantity.AsApproximateFloat64()

	compareFloat64(t, actual, expected)
}

func evaluateAndCompareToBool(t *testing.T, expression string, expected bool) {
	refVal := runAndCheckExpression(t, expression, types.BoolType)

	actual := refVal.Value().(bool)
	if actual != expected {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func evaluateAndShouldError(t *testing.T, expression string) {
	_, err := runExpression(t, expression)
	if err == nil {
		t.Errorf("expected error, but got none")
	}
}

func runAndCheckExpression(t *testing.T, expr string, expectedType ref.Type) ref.Val {
	return runAndCheckExpressionWithData(t, expr, map[string]any{}, expectedType)
}

func runExpression(t *testing.T, expression string) (ref.Val, error) {
	return runExpressionWithData(t, expression, map[string]any{})
}

func runAndCheckExpressionWithData(t *testing.T, expr string, data map[string]any, expectedType ref.Type) ref.Val {
	refVal, err := runExpressionWithData(t, expr, data)
	if err != nil {
		t.Fatalf("unexpected error running expression %q with data %v: %v", expr, data, err)
	}
	if refVal.Type() != expectedType {
		t.Fatalf("expected %v, got %v", expectedType, refVal.Type())
	}
	return refVal
}

func runExpressionWithData(t *testing.T, expression string, data map[string]any) (ref.Val, error) {
	funcs := maps.Clone(DefaultFuncs)

	// make Now() return a fixed value
	funcs[nowName] = []any{func() time.Time {
		return expectedTime
	}}

	env, err := NewEnvironment(EnvironmentConfig{
		Types:       DefaultTypes,
		Conversions: DefaultConversions,
		Methods:     FuncsToMethods(funcs),
		Funcs:       funcs,
		Vars:        data,
	})
	if err != nil {
		t.Fatalf("failed to instantiate node Evaluator: %v", err)
	}

	eval, err := env.Compile(expression)
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	refVal, _, err := eval.Eval(data)
	return refVal, err
}

func compareFloat64(t *testing.T, actual, expected float64) {
	const epsilon = 1e-9
	if math.Abs(actual-expected) > epsilon {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}
