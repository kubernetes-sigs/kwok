/*
Copyright 2025 The Kubernetes Authors.

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

package internalversion

// ExpressionJQ is the expression which will be evaluated by JQ.
type ExpressionJQ struct {
	// Expression represents the expression which will be evaluated by JQ.
	Expression string
}

// SelectorJQ is a resource selector requirement is a selector that contains values, a key,
// and an operator that relates the key and values.
type SelectorJQ struct {
	// Key represents the expression which will be evaluated by JQ.
	Key string
	// Represents a scope's relationship to a set of values.
	Operator SelectorOperator
	// An array of string values.
	// If the operator is In, NotIn, Intersection or NotIntersection, the values array must be non-empty.
	// If the operator is Exists or DoesNotExist, the values array must be empty.
	Values []string
}

// SelectorOperator is a label selector operator is the set of operators that can be used in a selector requirement.
type SelectorOperator string

// The following are valid selector operators.
const (
	// SelectorOpIn is the set inclusion operator.
	SelectorOpIn SelectorOperator = "In"
	// SelectorOpNotIn is the negated set inclusion operator.
	SelectorOpNotIn SelectorOperator = "NotIn"
	// SelectorOpExists is the existence operator.
	SelectorOpExists SelectorOperator = "Exists"
	// SelectorOpDoesNotExist is the negated existence operator.
	SelectorOpDoesNotExist SelectorOperator = "DoesNotExist"
)
