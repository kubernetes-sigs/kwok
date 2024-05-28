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

	"github.com/google/cel-go/common/types"
	"k8s.io/apimachinery/pkg/api/resource"
)

// TestNewQuantity tests the NewQuantity function.
func TestNewQuantity(t *testing.T) {
	// Test with a non-nil quantity
	q := resource.NewQuantity(10, resource.DecimalSI)
	quantity := NewQuantity(q)
	if quantity.Quantity.Cmp(*q) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", q, quantity.Quantity)
	}

	// Test with a nil quantity
	quantity = NewQuantity(nil)
	if quantity.Quantity.Cmp(*resource.NewScaledQuantity(0, resource.Nano)) != 0 {
		t.Errorf("Expected quantity to be 0, got %v", quantity.Quantity)
	}
}

// TestNewQuantityFromString tests the NewQuantityFromString function.
func TestNewQuantityFromString(t *testing.T) {
	qStr := "10"
	// Create a new Quantity from a string
	q, err := NewQuantityFromString(qStr)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check if the created Quantity matches the expected value
	expectedQuantity, _ := resource.ParseQuantity(qStr)
	if q.Quantity.Cmp(expectedQuantity) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", expectedQuantity, q.Quantity)
	}

	// Test with an invalid string
	qStr = "invalid"
	_, err = NewQuantityFromString(qStr)
	if err == nil {
		t.Errorf("Expected an error for invalid quantity string")
	}
}

// TestConvertToNative tests the ConvertToNative function.
func TestConvertToNative(t *testing.T) {
	q := resource.NewQuantity(10, resource.DecimalSI)
	quantity := NewQuantity(q)

	v, err := quantity.ConvertToNative(reflect.TypeOf(float64(0)))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedValue := q.AsApproximateFloat64()
	if v.(float64) != expectedValue {
		t.Errorf("Expected value to be %v, got %v", expectedValue, v)
	}

	// Test with unsupported type
	_, err = quantity.ConvertToNative(reflect.TypeOf(""))
	if err == nil {
		t.Errorf("Expected an error for unsupported type conversion")
	}
}

// TestConvertToType tests the ConvertToType function.
func TestConvertToType(t *testing.T) {
	q := resource.NewQuantity(10, resource.DecimalSI)
	quantity := NewQuantity(q)

	v := quantity.ConvertToType(types.DoubleType)
	expectedValue := types.Double(q.AsApproximateFloat64())
	if v != expectedValue {
		t.Errorf("Expected value to be %v, got %v", expectedValue, v)
	}

	v = quantity.ConvertToType(types.StringType)
	expectedValueStr := types.String(q.String())
	if v != expectedValueStr {
		t.Errorf("Expected value to be %v, got %v", expectedValueStr, v)
	}

	v = quantity.ConvertToType(types.TypeType)
	if v != QuantityType {
		t.Errorf("Expected value to be %v, got %v", QuantityType, v)
	}
}

// TestEqual tests the Equal function.
func TestEqual(t *testing.T) {
	q1 := resource.NewQuantity(10, resource.DecimalSI)
	q2 := resource.NewQuantity(10, resource.DecimalSI)
	quantity1 := NewQuantity(q1)
	quantity2 := NewQuantity(q2)

	if !quantity1.Equal(quantity2).(types.Bool) {
		t.Errorf("Expected quantities to be equal")
	}

	q3 := resource.NewQuantity(5, resource.DecimalSI)
	quantity3 := NewQuantity(q3)
	if quantity1.Equal(quantity3).(types.Bool) {
		t.Errorf("Expected quantities to be not equal")
	}
}

// TestAdd tests the Add function.
func TestAdd(t *testing.T) {
	q1 := resource.NewQuantity(10, resource.DecimalSI)
	q2 := resource.NewQuantity(5, resource.DecimalSI)
	quantity1 := NewQuantity(q1)
	quantity2 := NewQuantity(q2)

	result := quantity1.Add(quantity2).(Quantity)
	expected := resource.NewQuantity(15, resource.DecimalSI)
	if result.Quantity.Cmp(*expected) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", expected, result.Quantity)
	}
}

// TestSubtract tests the Subtract function.
func TestSubtract(t *testing.T) {
	q1 := resource.NewQuantity(10, resource.DecimalSI)
	q2 := resource.NewQuantity(5, resource.DecimalSI)
	quantity1 := NewQuantity(q1)
	quantity2 := NewQuantity(q2)

	result := quantity1.Subtract(quantity2).(Quantity)
	expected := resource.NewQuantity(5, resource.DecimalSI)
	if result.Quantity.Cmp(*expected) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", expected, result.Quantity)
	}
}

// TestNegate tests the Negate function.
func TestNegate(t *testing.T) {
	q := resource.NewQuantity(10, resource.DecimalSI)
	quantity := NewQuantity(q)

	result := quantity.Negate().(Quantity)
	expected := resource.NewQuantity(-10, resource.DecimalSI)
	if result.Quantity.Cmp(*expected) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", expected, result.Quantity)
	}
}

// TestNegate tests the Divide function.
func TestDivide(t *testing.T) {
	// Create a Quantity to divide
	q := resource.NewQuantity(10, resource.DecimalSI)
	quantity := NewQuantity(q)

	// Divide the Quantity by an integer
	result := quantity.Divide(types.Int(2)).(Quantity)
	expected := resource.NewQuantity(5, resource.DecimalSI)

	// Check if the result matches the expected value
	if result.Quantity.Cmp(*expected) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", expected, result.Quantity)
	}

	// Divide the Quantity by a double
	doubleResult := quantity.Divide(types.Double(2.5)).(Quantity)
	expectedDouble := resource.NewQuantity(4, resource.DecimalSI)

	// Check if the result matches the expected value
	if doubleResult.Quantity.Cmp(*expectedDouble) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", expectedDouble, doubleResult.Quantity)
	}
}

// TestNegate tests the Multiply function.
func TestMultiply(t *testing.T) {
	q := resource.NewQuantity(10, resource.DecimalSI)
	quantity := NewQuantity(q)

	// Multiply by an integer
	result := quantity.Multiply(types.Int(2)).(Quantity)
	expected := resource.NewQuantity(20, resource.DecimalSI)
	if result.Quantity.Cmp(*expected) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", expected, result.Quantity)
	}

	// Multiply by a double
	result = quantity.Multiply(types.Double(2.5)).(Quantity)
	expected = resource.NewQuantity(25, resource.DecimalSI)
	if result.Quantity.Cmp(*expected) != 0 {
		t.Errorf("Expected quantity to be %v, got %v", expected, result.Quantity)
	}
}

// TestNegate tests the TestCompare function.
func TestCompare(t *testing.T) {
	q1 := resource.NewQuantity(10, resource.DecimalSI)
	q2 := resource.NewQuantity(5, resource.DecimalSI)
	quantity1 := NewQuantity(q1)
	quantity2 := NewQuantity(q2)

	result := quantity1.Compare(quantity2).(types.Int)
	expected := int64(1)
	if result != types.Int(expected) {
		t.Errorf("Expected comparison result to be %v, got %v", expected, result)
	}

	quantity3 := NewQuantity(q1)
	result = quantity1.Compare(quantity3).(types.Int)
	expected = int64(0)
	if result != types.Int(expected) {
		t.Errorf("Expected comparison result to be %v, got %v", expected, result)
	}

	quantity4 := NewQuantity(resource.NewQuantity(20, resource.DecimalSI))
	result = quantity1.Compare(quantity4).(types.Int)
	expected = int64(-1)
	if result != types.Int(expected) {
		t.Errorf("Expected comparison result to be %v, got %v", expected, result)
	}
}

// TestNegate tests the TestIsZeroValue function.
func TestIsZeroValue(t *testing.T) {
	q := resource.NewQuantity(0, resource.DecimalSI)
	quantity := NewQuantity(q)
	if !quantity.IsZeroValue() {
		t.Errorf("Expected quantity to be zero value")
	}

	nonZeroQ := resource.NewQuantity(10, resource.DecimalSI)
	nonZeroQuantity := NewQuantity(nonZeroQ)
	if nonZeroQuantity.IsZeroValue() {
		t.Errorf("Expected quantity to be non-zero value")
	}
}
