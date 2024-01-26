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
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	// QuantityType singleton.
	QuantityType = cel.ObjectType("kubernetes.Quantity",
		traits.AdderType,
		traits.ComparerType,
		traits.DividerType,
		traits.MultiplierType,
		traits.NegatorType,
		traits.SubtractorType,
	)
)

// Quantity is a wrapper around k8s.io/apimachinery/pkg/api/resource.Quantity
type Quantity struct {
	Quantity *resource.Quantity
}

// NewQuantity creates a new Quantity
func NewQuantity(q *resource.Quantity) Quantity {
	if q == nil {
		q = resource.NewScaledQuantity(0, resource.Nano)
	}
	return Quantity{Quantity: q}
}

// NewQuantityFromString creates a new Quantity from a string
func NewQuantityFromString(s string) (Quantity, error) {
	r, err := resource.ParseQuantity(s)
	if err != nil {
		return Quantity{}, err
	}
	return NewQuantity(&r), nil
}

func newQuantityFromNanoInt64(v int64) Quantity {
	r := resource.NewScaledQuantity(v, resource.Nano)
	return NewQuantity(r)
}

func newQuantityFromFloat64(v float64) Quantity {
	r := resource.NewScaledQuantity(int64(v*10e9), resource.Nano)
	return NewQuantity(r)
}

func (q Quantity) nano() int64 {
	return q.Quantity.ScaledValue(resource.Nano)
}

func (q Quantity) float() float64 {
	return q.Quantity.AsApproximateFloat64()
}

// ConvertToNative implements the ref.Val interface.
func (q Quantity) ConvertToNative(typeDesc reflect.Type) (any, error) {
	switch typeDesc.Kind() {
	case reflect.Float32, reflect.Float64:
		v := q.Quantity.AsApproximateFloat64()
		return reflect.ValueOf(v).Convert(typeDesc).Interface(), nil
	}
	return nil, fmt.Errorf("type conversion error from Quantity to '%v'", typeDesc)
}

// ConvertToType implements the ref.Val interface.
func (q Quantity) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case types.DoubleType:
		v := q.Quantity.AsApproximateFloat64()
		return types.Double(v)
	case types.StringType:
		v := q.Quantity.String()
		return types.String(v)
	case types.TypeType:
		return QuantityType
	}
	return types.NewErr("type conversion error from '%s' to '%s'", QuantityType, typeVal)
}

// Equal implements the ref.Val interface.
func (q Quantity) Equal(other ref.Val) ref.Val {
	otherQuantity, ok := other.(Quantity)
	if !ok {
		return types.MaybeNoSuchOverloadErr(other)
	}
	return types.Bool(q.Quantity.Equal(*otherQuantity.Quantity))
}

// Type implements the ref.Val interface.
func (q Quantity) Type() ref.Type {
	return QuantityType
}

// Value implements the ref.Val interface.
func (q Quantity) Value() any {
	return q.Quantity
}

// Add implements the traits.Adder interface.
func (q Quantity) Add(other ref.Val) ref.Val {
	otherQuantity, ok := other.(Quantity)
	if !ok {
		return types.MaybeNoSuchOverloadErr(other)
	}
	r := q.Quantity.DeepCopy()
	r.Add(*otherQuantity.Quantity)
	return NewQuantity(&r)
}

// Subtract implements the traits.Subtractor interface.
func (q Quantity) Subtract(subtrahend ref.Val) ref.Val {
	otherQuantity, ok := subtrahend.(Quantity)
	if !ok {
		return types.MaybeNoSuchOverloadErr(subtrahend)
	}
	r := q.Quantity.DeepCopy()
	r.Sub(*otherQuantity.Quantity)
	return NewQuantity(&r)
}

// Negate implements the traits.Negater interface.
func (q Quantity) Negate() ref.Val {
	r := q.Quantity.DeepCopy()
	r.Neg()
	return NewQuantity(&r)
}

// Divide implements the traits.Divider interface.
func (q Quantity) Divide(other ref.Val) ref.Val {
	switch other.Type() {
	case types.IntType:
		otherInt := other.(types.Int)
		return newQuantityFromNanoInt64(q.nano() / int64(otherInt))
	case types.UintType:
		otherUint := other.(types.Uint)
		return newQuantityFromNanoInt64(q.nano() / int64(otherUint))
	case types.DoubleType:
		otherDouble := other.(types.Double)
		return newQuantityFromFloat64(q.float() / float64(otherDouble))
	}

	return types.MaybeNoSuchOverloadErr(other)
}

// Multiply implements the traits.Multiplier interface.
func (q Quantity) Multiply(other ref.Val) ref.Val {
	switch other.Type() {
	case types.IntType:
		otherInt := other.(types.Int)
		return newQuantityFromNanoInt64(q.nano() * int64(otherInt))
	case types.UintType:
		otherUint := other.(types.Uint)
		return newQuantityFromNanoInt64(q.nano() * int64(otherUint))
	case types.DoubleType:
		otherDouble := other.(types.Double)
		return newQuantityFromFloat64(q.float() * float64(otherDouble))
	}

	return types.MaybeNoSuchOverloadErr(other)
}

// Compare implements the traits.Comparer interface.
func (q Quantity) Compare(other ref.Val) ref.Val {
	otherQuantity, ok := other.(Quantity)
	if !ok {
		return types.MaybeNoSuchOverloadErr(other)
	}
	return types.Int(q.Quantity.Cmp(*otherQuantity.Quantity))
}

// IsZeroValue implements the traits.Zeroer interface.
func (q Quantity) IsZeroValue() bool {
	return q.Quantity.IsZero()
}
