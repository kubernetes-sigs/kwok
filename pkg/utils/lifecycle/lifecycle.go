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
	"math/rand"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/cel"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

// NewLifecycle returns a new Lifecycle.
func NewLifecycle(stages []*internalversion.Stage, env *cel.Environment) (Lifecycle, error) {
	lcs := Lifecycle{}
	for _, stage := range stages {
		lc, err := NewStage(stage, env)
		if err != nil {
			return nil, fmt.Errorf("lifecycle stage: %w", err)
		}
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
		w, ok := stage.Weight(ctx, event)
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
		w, ok := stage.Weight(ctx, event)
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
func NewStage(s *internalversion.Stage, env *cel.Environment) (*Stage, error) {
	stage := &Stage{
		name: s.Name,
	}
	selector := s.Spec.Selector
	if selector == nil {
		return nil, nil
	}

	if selector.MatchLabels != nil {
		stage.matchLabels = labels.SelectorFromSet(selector.MatchLabels)
	}
	if selector.MatchAnnotations != nil {
		stage.matchAnnotations = labels.SelectorFromSet(selector.MatchAnnotations)
	}
	if selector.MatchExpressions != nil {
		for _, express := range selector.MatchExpressions {
			switch {
			case express.JQ != nil:
				requirement, err := expression.NewRequirement(express.JQ.Key, express.JQ.Operator, express.JQ.Values)
				if err != nil {
					return nil, err
				}
				stage.matchExpressions = append(stage.matchExpressions, requirement)
			case express.CEL != nil:
				program, err := env.Compile(express.CEL.Expression)
				if err != nil {
					return nil, err
				}
				stage.matchConditions = append(stage.matchConditions, program)
			default:
				return nil, fmt.Errorf("invalid expression")
			}
		}
	}

	stage.next = &s.Spec.Next
	if delay := s.Spec.Delay; delay != nil {
		var durationFrom *internalversion.ExpressionFromSource
		if delay.DurationFrom != nil {
			durationFrom = delay.DurationFrom
		}
		var delayDuration time.Duration
		if delay.DurationMilliseconds != nil {
			delayDuration = time.Duration(*delay.DurationMilliseconds) * time.Millisecond
		}
		duration, err := NewDurationFrom(&delayDuration, env, durationFrom)
		if err != nil {
			return nil, err
		}
		stage.duration = duration

		if delay.JitterDurationMilliseconds != nil || delay.JitterDurationFrom != nil {
			var jitterDurationFrom *internalversion.ExpressionFromSource
			if delay.JitterDurationFrom != nil && delay.JitterDurationFrom.JQ != nil {
				jitterDurationFrom = delay.JitterDurationFrom
			}
			var jitterDuration *time.Duration
			if delay.JitterDurationMilliseconds != nil {
				jitterDuration = format.Ptr(time.Duration(*delay.JitterDurationMilliseconds) * time.Millisecond)
			}
			jitterDurationGetter, err := NewDurationFrom(jitterDuration, env, jitterDurationFrom)
			if err != nil {
				return nil, err
			}
			stage.jitterDuration = jitterDurationGetter
		}
	}

	var weightFrom *internalversion.ExpressionFromSource
	if s.Spec.WeightFrom != nil {
		weightFrom = s.Spec.WeightFrom
	}

	weightGetter, err := NewIntFrom(format.Ptr[int64](int64(s.Spec.Weight)), env, weightFrom)
	if err != nil {
		return nil, err
	}

	stage.weight = weightGetter

	stage.immediateNextStage = s.Spec.ImmediateNextStage

	return stage, nil
}

// Stage is a resource lifecycle stage manager
type Stage struct {
	name             string
	matchLabels      labels.Selector
	matchAnnotations labels.Selector
	matchExpressions []*expression.Requirement
	matchConditions  []cel.Program

	weight intGetter
	next   *internalversion.StageNext

	duration       durationGetter
	jitterDuration durationGetter

	immediateNextStage bool
}

// Event represents a lifecycle event that can be matched against stage conditions
// It contains the resource's labels, annotations, raw data, and a cached JSON representation
type Event struct {
	Labels       labels.Set
	Annotations  labels.Set
	Data         any
	jsonStandard any
}

// toJSONStandard converts the event's Data to a standard JSON format that can be used by JQ expressions.
// It caches the result for subsequent calls to avoid repeated conversions.
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

func (s *Stage) match(ctx context.Context, event *Event) (bool, error) {
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
		for _, requirement := range s.matchExpressions {
			ok, err := requirement.Matches(ctx, event.toJSONStandard())
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
	}

	if s.matchConditions != nil {
		for _, program := range s.matchConditions {
			val, _, err := program.ContextEval(ctx, map[string]any{
				"self": event,
			})
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
func (s *Stage) Delay(ctx context.Context, event *Event, now time.Time) (time.Duration, bool) {
	if s.duration == nil {
		return 0, false
	}

	duration, ok := s.duration.Get(ctx, event, now)
	if !ok {
		return 0, false
	}

	if s.jitterDuration == nil {
		return duration, true
	}

	jitterDuration, ok := s.jitterDuration.Get(ctx, event, now)
	if !ok {
		return duration, true
	}

	if jitterDuration <= duration {
		return jitterDuration, true
	}

	//nolint:gosec
	duration += time.Duration(rand.Int63n(int64(jitterDuration - duration)))

	return duration, true
}

// DelayRangePossible returns possible range of delay.
func (s *Stage) DelayRangePossible(ctx context.Context, event *Event, now time.Time) ([]time.Duration, bool) {
	if s.duration == nil {
		return nil, false
	}

	duration, ok := s.duration.Get(ctx, event, now)
	if !ok {
		return nil, false
	}

	if s.jitterDuration == nil {
		return []time.Duration{duration}, true
	}

	jitterDuration, ok := s.jitterDuration.Get(ctx, event, now)
	if !ok {
		return []time.Duration{duration}, true
	}

	if jitterDuration <= duration {
		return []time.Duration{jitterDuration}, true
	}

	return []time.Duration{duration, jitterDuration}, true
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
func (s *Stage) Weight(ctx context.Context, event *Event) (int64, bool) {
	return s.weight.Get(ctx, event)
}
