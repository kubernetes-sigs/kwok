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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// StageKind is the kind of the Stage resource.
	StageKind = "Stage"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=stages,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=stages/status,verbs=update;patch

// Stage is an API that describes the staged change of a resource
type Stage struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds information about the request being evaluated.
	Spec StageSpec `json:"spec"`
	// Status holds status for the Stage
	//+k8s:conversion-gen=false
	Status StageStatus `json:"status,omitempty"`
}

// StageStatus holds status for the Stage
type StageStatus struct {
	// Conditions holds conditions for the Stage.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// StageSpec defines the specification for Stage.
type StageSpec struct {
	// ResourceRef specifies the Kind and version of the resource.
	ResourceRef StageResourceRef `json:"resourceRef"`
	// Selector specifies the stags will be applied to the selected resource.
	Selector *StageSelector `json:"selector,omitempty"`
	// Weight means when multiple stages share the same ResourceRef and Selector,
	// a random stage will be matched as the next stage based on the weight.
	// +default=0
	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum=0
	Weight int `json:"weight,omitempty"`
	// Delay means there is a delay in this stage.
	Delay *StageDelay `json:"delay,omitempty"`
	// Next indicates that this stage will be moved to.
	Next StageNext `json:"next"`
	// ImmediateNextStage means that the next stage of matching is performed immediately, without waiting for the Apiserver to push.
	ImmediateNextStage *bool `json:"immediateNextStage,omitempty"`
}

// StageResourceRef specifies the kind and version of the resource.
type StageResourceRef struct {
	// APIGroup of the referent.
	// +default="v1"
	// +kubebuilder:default="v1"
	APIGroup string `json:"apiGroup,omitempty"`
	// Kind of the referent.
	Kind string `json:"kind"`
}

// StageDelay describes the delay time before going to next.
type StageDelay struct {
	// DurationMilliseconds indicates the stage delay time.
	// If JitterDurationMilliseconds is less than DurationMilliseconds, then JitterDurationMilliseconds is used.
	// +kubebuilder:validation:Minimum=0
	DurationMilliseconds *int64 `json:"durationMilliseconds,omitempty"`
	// DurationFrom is the expression used to get the value.
	// If it is a time.Time type, getting the value will be minus time.Now() to get DurationMilliseconds
	// If it is a string type, the value get will be parsed by time.ParseDuration.
	DurationFrom *ExpressionFromSource `json:"durationFrom,omitempty"`

	// JitterDurationMilliseconds is the duration plus an additional amount chosen uniformly
	// at random from the interval between DurationMilliseconds and JitterDurationMilliseconds.
	// +kubebuilder:validation:Minimum=0
	JitterDurationMilliseconds *int64 `json:"jitterDurationMilliseconds,omitempty"`
	// JitterDurationFrom is the expression used to get the value.
	// If it is a time.Time type, getting the value will be minus time.Now() to get JitterDurationMilliseconds
	// If it is a string type, the value get will be parsed by time.ParseDuration.
	JitterDurationFrom *ExpressionFromSource `json:"jitterDurationFrom,omitempty"`
}

// StageNext describes a stage will be moved to.
type StageNext struct {
	// Event means that an event will be sent.
	Event *StageEvent `json:"event,omitempty"`
	// Finalizers means that finalizers will be modified.
	Finalizers *StageFinalizers `json:"finalizers,omitempty"`
	// Delete means that the resource will be deleted if true.
	Delete bool `json:"delete,omitempty"`
	// Patches means that the resource will be patched.
	Patches []StagePatch `json:"patches,omitempty"`

	// StatusTemplate indicates the template for modifying the status of the resource in the next.
	// Deprecated: Use Patches instead.
	//+k8s:conversion-gen=false
	StatusTemplate string `json:"statusTemplate,omitempty"`
	// StatusSubresource indicates the name of the subresource that will be patched. The support for
	// this field is not available in Pod and Node resources.
	// +default="status"
	// +kubebuilder:default=status
	// Deprecated: Use Patches instead.
	//+k8s:conversion-gen=false
	StatusSubresource *string `json:"statusSubresource,omitempty"`
	// StatusPatchAs indicates the impersonating configuration for client when patching status.
	// In most cases this will be empty, in which case the default client service account will be used.
	// When this is not empty, a corresponding rbac change is required to grant `impersonate` privilege.
	// The support for this field is not available in Pod and Node resources.
	// Deprecated: Use Patches instead.
	//+k8s:conversion-gen=false
	StatusPatchAs *ImpersonationConfig `json:"statusPatchAs,omitempty"`
}

// StagePatch describes the patch for the resource.
type StagePatch struct {
	// Subresource indicates the name of the subresource that will be patched.
	Subresource string `json:"subresource,omitempty"`
	// Root indicates the root of the template calculated by the patch.
	Root string `json:"root,omitempty"`
	// Template indicates the template for modifying the resource in the next.
	Template string `json:"template,omitempty"`
	// Type indicates the type of the patch.
	// +kubebuilder:validation:Enum=json;merge;strategic
	Type *StagePatchType `json:"type,omitempty"`
	// Impersonation indicates the impersonating configuration for client when patching status.
	// In most cases this will be empty, in which case the default client service account will be used.
	// When this is not empty, a corresponding rbac change is required to grant `impersonate` privilege.
	// The support for this field is not available in Pod and Node resources.
	Impersonation *ImpersonationConfig `json:"impersonation,omitempty"`
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
	Username string `json:"username"`
}

// StageFinalizers describes the modifications in the finalizers of a resource.
type StageFinalizers struct {
	// Add means that the Finalizers will be added to the resource.
	Add []FinalizerItem `json:"add,omitempty"`
	// Remove means that the Finalizers will be removed from the resource.
	Remove []FinalizerItem `json:"remove,omitempty"`
	// Empty means that the Finalizers for that resource will be emptied.
	Empty bool `json:"empty,omitempty"`
}

// FinalizerItem  describes the one of the finalizers.
type FinalizerItem struct {
	// Value is the value of the finalizer.
	Value string `json:"value,omitempty"`
}

// StageEvent describes one event in the Kubernetes.
type StageEvent struct {
	// Type is the type of this event (Normal, Warning), It is machine-readable.
	Type string `json:"type,omitempty"`
	// Reason is why the action was taken. It is human-readable.
	Reason string `json:"reason,omitempty"`
	// Message is a human-readable description of the status of this operation.
	Message string `json:"message,omitempty"`
}

// StageSelector is a resource selector. the result of matchLabels and matchAnnotations and
// matchExpressions are ANDed. An empty resource selector matches all objects. A null
// resource selector matches no objects.
type StageSelector struct {
	// MatchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
	// map is equivalent to an element of matchExpressions, whose key field is ".metadata.labels[key]", the
	// operator is "In", and the values array contains only "value". The requirements are ANDed.
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
	// MatchAnnotations is a map of {key,value} pairs. A single {key,value} in the matchAnnotations
	// map is equivalent to an element of matchExpressions, whose key field is ".metadata.annotations[key]", the
	// operator is "In", and the values array contains only "value". The requirements are ANDed.
	MatchAnnotations map[string]string `json:"matchAnnotations,omitempty"`
	// MatchExpressions is a list of label selector requirements. The requirements are ANDed.
	MatchExpressions []SelectorRequirement `json:"matchExpressions,omitempty"`
}

// SelectorRequirement is a resource selector requirement is a selector that contains values, a key,
// and an operator that relates the key and values.
type SelectorRequirement struct {
	// The name of the scope that the selector applies to.
	Key string `json:"key"`
	// Represents a scope's relationship to a set of values.
	Operator SelectorOperator `json:"operator"`
	// An array of string values.
	// If the operator is In, NotIn, Intersection or NotIntersection, the values array must be non-empty.
	// If the operator is Exists or DoesNotExist, the values array must be empty.
	Values []string `json:"values,omitempty"`
}

// SelectorOperator is a label selector operator is the set of operators that can be used in a selector requirement.
// +enum
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
	ExpressionFrom string `json:"expressionFrom,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// StageList contains a list of Stage
type StageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stage{}, &StageList{})
}
