/*
Copyright 2022 The Kubernetes Authors.

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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Stage is an API that describes the staged change of a resource
type Stage struct {
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Spec holds information about the request being evaluated.
	Spec StageSpec
}

// StageSpec defines the specification for Stage.
type StageSpec struct {
	// ResourceRef specifies the Kind and version of the resource.
	ResourceRef StageResourceRef
	// Selector specifies the stags will be applied to the selected resource.
	Selector *StageSelector
	// Weight means when multiple stages share the same ResourceRef and Selector,
	// a random stage will be matched as the next stage based on the weight.
	Weight int
	// WeightFrom means is the expression used to get the value.
	// If it is a number type, convert to int.
	// If it is a string type, the value get will be parsed by strconv.ParseInt.
	WeightFrom *ExpressionFromSource
	// Delay means there is a delay in this stage.
	Delay *StageDelay
	// Next indicates that this stage will be moved to.
	Next StageNext
	// ImmediateNextStage means that the next stage of matching is performed immediately, without waiting for the Apiserver to push.
	ImmediateNextStage bool
}

// StageResourceRef specifies the kind and version of the resource.
type StageResourceRef struct {
	// APIGroup of the referent.
	APIGroup string
	// Kind of the referent.
	Kind string
}

// StageDelay describes the delay time before going to next.
type StageDelay struct {
	// DurationMilliseconds indicates the stage delay time.
	// If JitterDurationMilliseconds is less than DurationMilliseconds, then JitterDurationMilliseconds is used.
	DurationMilliseconds *int64
	// DurationFrom is the expression used to get the value.
	// If it is a time.Time type, getting the value will be minus time.Now() to get DurationMilliseconds
	// If it is a string type, the value get will be parsed by time.ParseDuration.
	DurationFrom *ExpressionFromSource

	// JitterDurationMilliseconds is the duration plus an additional amount chosen uniformly
	// at random from the interval between DurationMilliseconds and JitterDurationMilliseconds.
	JitterDurationMilliseconds *int64
	// JitterDurationFrom is the expression used to get the value.
	// If it is a time.Time type, getting the value will be minus time.Now() to get JitterDurationMilliseconds
	// If it is a string type, the value get will be parsed by time.ParseDuration.
	JitterDurationFrom *ExpressionFromSource
}

// StageNext describes a stage will be moved to.
type StageNext struct {
	// Event means that an event will be sent.
	Event *StageEvent
	// Finalizers means that finalizers will be modified.
	Finalizers *StageFinalizers
	// Delete means that the resource will be deleted if true.
	Delete bool
	// Patches means that the resource will be patched.
	Patches []StagePatch
}

// StagePatch describes the patch for the resource.
type StagePatch struct {
	// Subresource indicates the name of the subresource that will be patched.
	Subresource string
	// Root indicates the root of the template calculated by the patch.
	Root string
	// Template indicates the template for modifying the resource in the next.
	Template string
	// Type indicates the type of the patch.
	Type *StagePatchType
	// Impersonation indicates the impersonating configuration for client when patching status.
	// In most cases this will be empty, in which case the default client service account will be used.
	// When this is not empty, a corresponding rbac change is required to grant `impersonate` privilege.
	// The support for this field is not available in Pod and Node resources.
	Impersonation *ImpersonationConfig
}

// StagePatchType is the type of the patch.
type StagePatchType string

const (
	// StagePatchTypeJSONPatch is the JSON patch type.
	StagePatchTypeJSONPatch StagePatchType = "json"
	// StagePatchTypeMergePatch is the merge patch type.
	StagePatchTypeMergePatch StagePatchType = "merge"
	// StagePatchTypeStrategicMergePatch is the strategic merge patch type.
	StagePatchTypeStrategicMergePatch StagePatchType = "strategic"
)

// ImpersonationConfig describes the configuration for impersonating clients
type ImpersonationConfig struct {
	// Username the target username for the client to impersonate
	Username string
}

// StageFinalizers describes the modifications in the finalizers of a resource.
type StageFinalizers struct {
	// Add means that the Finalizers will be added to the resource.
	Add []FinalizerItem
	// Remove means that the Finalizers will be removed from the resource.
	Remove []FinalizerItem
	// Empty means that the Finalizers for that resource will be emptied.
	Empty bool
}

// FinalizerItem  describes the one of the finalizers.
type FinalizerItem struct {
	// Value is the value of the finalizer.
	Value string
}

// StageEvent describes one event in the Kubernetes.
type StageEvent struct {
	// Type is the type of this event (Normal, Warning), It is machine-readable.
	Type string
	// Reason is why the action was taken. It is human-readable.
	Reason string
	// Message is a human-readable description of the status of this operation.
	Message string
}

// StageSelector is a resource selector. the result of matchLabels and matchAnnotations and
// matchExpressions are ANDed. An empty resource selector matches all objects. A null
// resource selector matches no objects.
type StageSelector struct {
	// MatchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
	// map is equivalent to an element of matchExpressions, whose key field is ".metadata.labels[key]", the
	// operator is "In", and the values array contains only "value". The requirements are ANDed.
	MatchLabels map[string]string
	// MatchAnnotations is a map of {key,value} pairs. A single {key,value} in the matchAnnotations
	// map is equivalent to an element of matchExpressions, whose key field is ".metadata.annotations[key]", the
	// operator is "In", and the values array contains only "value". The requirements are ANDed.
	MatchAnnotations map[string]string
	// MatchExpressions is a list of label selector requirements. The requirements are ANDed.
	MatchExpressions []SelectorExpression
}

// SelectorExpression is a resource selector expression is a set of requirements that must be true for a match.
type SelectorExpression struct {
	*ExpressionCEL
	*SelectorJQ
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

// ExpressionFromSource represents a source for the value of a from.
type ExpressionFromSource struct {
	// ExpressionFrom is the expression used to get the value.
	ExpressionFrom string
}
