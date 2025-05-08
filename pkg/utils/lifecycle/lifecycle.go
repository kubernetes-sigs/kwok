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

package lifecycle

import (
	"context"
	"fmt"
	"maps"
	"math/rand"
	"reflect"
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/cel"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

// NewLifecycle returns a new Lifecycle.
func NewLifecycle(stages []*internalversion.Stage) (Lifecycle, error) {
	lcs := Lifecycle{}
	for _, stage := range stages {
		lc := NewStage(stage)
		if lc == nil {
			continue
		}
		lcs = append(lcs, lc)
	}
	return lcs, nil
}

// Lifecycle is a list of lifecycle stage.
type Lifecycle []*Stage

func (s Lifecycle) match(ctx context.Context, event *Event) ([]*Stage, error) {
	out := []*Stage{}
	for _, stage := range s {
		ok, err := stage.match(ctx, event)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, stage)
		}
	}
	return out, nil
}

// ListAllPossible returns all possible stages.
func (s Lifecycle) ListAllPossible(ctx context.Context, event *Event) ([]*Stage, error) {
	stages, err := s.match(ctx, event)
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return nil, nil
	}
	if len(stages) == 1 {
		return stages, nil
	}

	var weights = make([]int64, 0, len(stages))
	var totalWeights int64
	var countError int
	for _, stage := range stages {
		w, ok, err := stage.Weight(ctx, event)
		if err != nil {
			return nil, err
		}
		if ok {
			totalWeights += w
			weights = append(weights, w)
		} else {
			weights = append(weights, -1)
			countError++
		}
	}

	if countError == len(stages) {
		return stages, nil
	}

	if totalWeights == 0 {
		if countError == 0 {
			return stages, nil
		}
		stagesWithWeights := make([]*Stage, 0, len(stages))
		for i, stage := range stages {
			if weights[i] < 0 {
				continue
			}
			stagesWithWeights = append(stagesWithWeights, stage)
		}
		return stagesWithWeights, nil
	}

	stagesWithWeights := make([]*Stage, 0, len(stages))
	for i, stage := range stages {
		if weights[i] <= 0 {
			continue
		}
		stagesWithWeights = append(stagesWithWeights, stage)
	}
	return stagesWithWeights, nil
}

// Match returns matched stage.
func (s Lifecycle) Match(ctx context.Context, event *Event) (*Stage, error) {
	stages, err := s.match(ctx, event)
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return nil, nil
	}
	if len(stages) == 1 {
		return stages[0], nil
	}

	var weights = make([]int64, 0, len(stages))
	var totalWeights int64
	var countError int
	for _, stage := range stages {
		w, ok, err := stage.Weight(ctx, event)
		if err != nil {
			return nil, err
		}
		if ok {
			totalWeights += w
			weights = append(weights, w)
		} else {
			weights = append(weights, -1)
			countError++
		}
	}

	if countError == len(stages) {
		//nolint:gosec
		return stages[rand.Intn(len(stages))], nil
	}

	if totalWeights == 0 {
		if countError == 0 {
			//nolint:gosec
			return stages[rand.Intn(len(stages))], nil
		}

		stagesWithWeights := make([]*Stage, 0, len(stages))
		for i, stage := range stages {
			if weights[i] < 0 {
				continue
			}
			stagesWithWeights = append(stagesWithWeights, stage)
		}

		//nolint:gosec
		off := rand.Intn(len(stagesWithWeights))
		return stagesWithWeights[off], nil
	}

	//nolint:gosec
	off := rand.Int63n(totalWeights)
	for i, stage := range stages {
		if weights[i] <= 0 {
			continue
		}
		off -= weights[i]
		if off < 0 {
			return stage, nil
		}
	}
	return stages[len(stages)-1], nil
}

// NewStage returns a new Stage.
func NewStage(s *internalversion.Stage) *Stage {
	if s.Spec.Selector == nil {
		return nil
	}

	stage := &Stage{
		name:   s.Name,
		config: s,
	}

	return stage
}

func (s *Stage) init(event *Event) error {
	if s.env != nil {
		return nil
	}
	types := slices.Clone(cel.DefaultTypes)
	conversions := slices.Clone(cel.DefaultConversions)
	funcs := maps.Clone(cel.DefaultFuncs)
	methods := maps.Clone(cel.FuncsToMethods(cel.DefaultFuncs))
	env, err := cel.NewEnvironment(cel.EnvironmentConfig{
		Types:       types,
		Conversions: conversions,
		Methods:     methods,
		Funcs:       funcs,
		Vars:        event.toCELStandardTypeOnly(),
	})
	if err != nil {
		return err
	}
	s.env = env

	selector := s.config.Spec.Selector
	config := s.config

	if selector.MatchLabels != nil {
		s.matchLabels = labels.SelectorFromSet(selector.MatchLabels)
	}
	if selector.MatchAnnotations != nil {
		s.matchAnnotations = labels.SelectorFromSet(selector.MatchAnnotations)
	}
	if selector.MatchExpressions != nil {
		for _, express := range selector.MatchExpressions {
			switch {
			case express.JQ != nil:
				requirement, err := expression.NewRequirement(express.JQ.Key, express.JQ.Operator, express.JQ.Values)
				if err != nil {
					return err
				}
				s.matchExpressions = append(s.matchExpressions, requirement)
			case express.CEL != nil:
				program, err := env.Compile(express.CEL.Expression)
				if err != nil {
					return err
				}
				s.matchConditions = append(s.matchConditions, program)
			default:
				return fmt.Errorf("invalid expression")
			}
		}
	}

	s.next = &config.Spec.Next
	if delay := config.Spec.Delay; delay != nil {
		var durationFrom *internalversion.ExpressionFrom
		if delay.DurationFrom != nil {
			durationFrom = delay.DurationFrom
		}
		var delayDuration time.Duration
		if delay.DurationMilliseconds != nil {
			delayDuration = time.Duration(*delay.DurationMilliseconds) * time.Millisecond
		}
		duration, err := newDurationFrom(&delayDuration, env, durationFrom)
		if err != nil {
			return err
		}
		s.duration = duration

		if delay.JitterDurationMilliseconds != nil || delay.JitterDurationFrom != nil {
			var jitterDurationFrom *internalversion.ExpressionFrom
			if delay.JitterDurationFrom != nil && delay.JitterDurationFrom.JQ != nil {
				jitterDurationFrom = delay.JitterDurationFrom
			}
			var jitterDuration *time.Duration
			if delay.JitterDurationMilliseconds != nil {
				jitterDuration = format.Ptr(time.Duration(*delay.JitterDurationMilliseconds) * time.Millisecond)
			}
			jitterDurationGetter, err := newDurationFrom(jitterDuration, env, jitterDurationFrom)
			if err != nil {
				return err
			}
			s.jitterDuration = jitterDurationGetter
		}
	}

	var weightFrom *internalversion.ExpressionFrom
	if config.Spec.WeightFrom != nil {
		weightFrom = config.Spec.WeightFrom
	}

	weightGetter, err := newInt64From(format.Ptr[int64](int64(config.Spec.Weight)), env, weightFrom)
	if err != nil {
		return err
	}

	s.weight = weightGetter

	s.immediateNextStage = config.Spec.ImmediateNextStage

	return nil
}

// Stage is a resource lifecycle stage manager
type Stage struct {
	name             string
	matchLabels      labels.Selector
	matchAnnotations labels.Selector
	matchExpressions []*expression.Requirement
	matchConditions  []cel.Program

	weight int64Getter
	next   *internalversion.StageNext

	duration       durationGetter
	jitterDuration durationGetter

	immediateNextStage bool

	config *internalversion.Stage

	env *cel.Environment
}

// Event represents a lifecycle event that can be matched against stage conditions
// It contains the resource's labels, annotations, raw data, and a cached JSON representation
type Event struct {
	Labels       labels.Set
	Annotations  labels.Set
	Data         any
	jsonStandard any
}

func (m *Event) toJSONStandard() any {
	if m.jsonStandard != nil {
		return m.jsonStandard
	}
	j, err := expression.ToJSONStandard(m.Data)
	if err != nil {
		// This should never happen as all resources should be convertible to JSON
		panic(fmt.Errorf("failed to convert data to JSON standard: %v, error: %w", m.Data, err))
	}
	m.jsonStandard = j
	return j
}

func (m *Event) toCELStandard() map[string]any {
	if v, ok := m.Data.(*unstructured.Unstructured); ok {
		return map[string]any{
			"self": v.Object,
		}
	}
	return map[string]any{
		"self": m.Data,
	}
}

func (m *Event) toCELStandardTypeOnly() map[string]any {
	if _, ok := m.Data.(*unstructured.Unstructured); ok {
		return map[string]any{
			"self": map[string]any{},
		}
	}

	return map[string]any{
		"self": reflect.Zero(reflect.TypeOf(m.Data)).Interface(),
	}
}

func (s *Stage) match(ctx context.Context, event *Event) (bool, error) {
	err := s.init(event)
	if err != nil {
		return false, err
	}
	if s.matchLabels != nil {
		if !s.matchLabels.Matches(event.Labels) {
			return false, nil
		}
	}
	if s.matchAnnotations != nil {
		if !s.matchAnnotations.Matches(event.Annotations) {
			return false, nil
		}
	}

	if s.matchExpressions != nil {
		v := event.toJSONStandard()
		for _, requirement := range s.matchExpressions {
			ok, err := requirement.Matches(ctx, v)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
	}

	if s.matchConditions != nil {
		v := event.toCELStandard()
		for _, program := range s.matchConditions {
			val, _, err := program.ContextEval(ctx, v)
			if err != nil {
				return false, err
			}
			ok, err := cel.AsBool(val)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
	}

	return true, nil
}

// Delay returns the delay duration of the stage.
// It's not a constant value, it can be a random value.
func (s *Stage) Delay(ctx context.Context, event *Event, now time.Time) (time.Duration, bool, error) {
	if s.duration == nil {
		return 0, false, nil
	}

	duration, ok, err := s.duration.Get(ctx, event, now)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}

	if s.jitterDuration == nil {
		return duration, true, nil
	}

	jitterDuration, ok, err := s.jitterDuration.Get(ctx, event, now)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return duration, true, nil
	}

	if jitterDuration <= duration {
		return jitterDuration, true, nil
	}

	//nolint:gosec
	duration += time.Duration(rand.Int63n(int64(jitterDuration - duration)))

	return duration, true, nil
}

// DelayRangePossible returns possible range of delay.
func (s *Stage) DelayRangePossible(ctx context.Context, event *Event, now time.Time) ([]time.Duration, bool, error) {
	if s.duration == nil {
		return nil, false, nil
	}

	duration, ok, err := s.duration.Get(ctx, event, now)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	if s.jitterDuration == nil {
		return []time.Duration{duration}, true, nil
	}

	jitterDuration, ok, err := s.jitterDuration.Get(ctx, event, now)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return []time.Duration{duration}, true, nil
	}

	if jitterDuration <= duration {
		return []time.Duration{jitterDuration}, true, nil
	}

	return []time.Duration{duration, jitterDuration}, true, nil
}

// Next returns the next of the stage.
func (s *Stage) Next() *Next {
	return newNext(s.next)
}

// Name returns the name of the stage
func (s *Stage) Name() string {
	return s.name
}

// ImmediateNextStage returns whether the stage is immediate next stage.
func (s *Stage) ImmediateNextStage() bool {
	return s.immediateNextStage
}

// Weight returns the weight of the stage.
func (s *Stage) Weight(ctx context.Context, event *Event) (int64, bool, error) {
	return s.weight.Get(ctx, event)
}
