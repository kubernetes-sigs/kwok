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

package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

func NewStagesFromYaml(data []byte) ([]*internalversion.Stage, error) {
	var podStageStatus []*internalversion.Stage
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewBuffer(data))
	for {
		var stage internalversion.Stage
		err := decoder.Decode(&stage)
		if errors.Is(err, io.EOF) {
			break
		}
		podStageStatus = append(podStageStatus, &stage)
	}
	return podStageStatus, nil
}

func NewLifecycle(stages []*internalversion.Stage) (Lifecycle, error) {
	lcs := Lifecycle{}
	for _, stage := range stages {
		lc, err := newLifecycleFromStage(stage)
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

type Lifecycle []*LifecycleStage

func (s Lifecycle) match(label, annotation labels.Set, data interface{}) ([]*LifecycleStage, error) {
	out := []*LifecycleStage{}
	for _, stage := range s {
		ok, err := stage.match(label, annotation, data)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, stage)
		}
	}
	return out, nil
}

func (s Lifecycle) Match(label, annotation labels.Set, data interface{}) (*LifecycleStage, error) {
	data, err := expression.ToJsonStandard(data)
	if err != nil {
		return nil, err
	}
	stages, err := s.match(label, annotation, data)
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return nil, nil
	}
	if len(stages) == 1 {
		return stages[0], nil
	}
	totalWeights := 0
	for _, stage := range stages {
		totalWeights += stage.weight
	}
	if totalWeights == 0 {
		//nolint:gosec
		return stages[rand.Intn(len(stages))], nil
	}

	//nolint:gosec
	off := rand.Intn(totalWeights)
	for _, stage := range stages {
		if stage.weight == 0 {
			continue
		}
		off -= stage.weight
		if off < 0 {
			return stage, nil
		}
	}
	return stages[len(stages)-1], nil
}

func newLifecycleFromStage(s *internalversion.Stage) (*LifecycleStage, error) {
	stage := &LifecycleStage{
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
			requirement, err := expression.NewRequirement(express.Key, express.Operator, express.Values)
			if err != nil {
				return nil, err
			}
			stage.matchExpressions = append(stage.matchExpressions, requirement)
		}
	}

	stage.next = &s.Spec.Next
	if delay := s.Spec.Delay; delay != nil {
		var durationFrom *string
		if delay.DurationFrom != nil {
			durationFrom = &delay.DurationFrom.ExpressionFrom
		}
		var delayDuration time.Duration
		if delay.DurationMilliseconds != nil {
			delayDuration = time.Duration(*delay.DurationMilliseconds) * time.Millisecond
		}
		duration, err := expression.NewDurationFrom(&delayDuration, durationFrom)
		if err != nil {
			return nil, err
		}
		stage.duration = duration

		if delay.JitterDurationMilliseconds != nil || delay.JitterDurationFrom != nil {
			var jitterDurationFrom *string
			if delay.JitterDurationFrom != nil {
				jitterDurationFrom = &delay.JitterDurationFrom.ExpressionFrom
			}
			var jitterDuration *time.Duration
			if delay.JitterDurationMilliseconds != nil {
				jitterDuration = format.Ptr(time.Duration(*delay.JitterDurationMilliseconds) * time.Millisecond)
			}
			jitterDurationGetter, err := expression.NewDurationFrom(jitterDuration, jitterDurationFrom)
			if err != nil {
				return nil, err
			}
			stage.jitterDuration = jitterDurationGetter
		}
	}

	if weight := s.Spec.Weight; weight > 1 {
		stage.weight = weight
	} else {
		stage.weight = 0
	}

	return stage, nil
}

type LifecycleStage struct {
	name             string
	matchLabels      labels.Selector
	matchAnnotations labels.Selector
	matchExpressions []*expression.Requirement

	weight int
	next   *internalversion.StageNext

	duration       expression.DurationGetter
	jitterDuration expression.DurationGetter
}

func (s *LifecycleStage) match(label, annotation labels.Set, jsonStandard interface{}) (bool, error) {
	if s.matchLabels != nil {
		if !s.matchLabels.Matches(label) {
			return false, nil
		}
	}
	if s.matchAnnotations != nil {
		if !s.matchAnnotations.Matches(annotation) {
			return false, nil
		}
	}

	if s.matchExpressions != nil {
		for _, requirement := range s.matchExpressions {
			ok, err := requirement.Matches(context.Background(), jsonStandard)
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

func (s *LifecycleStage) Delay(ctx context.Context, v interface{}, now time.Time) (time.Duration, bool) {
	if s.duration == nil {
		return 0, false
	}

	duration, ok := s.duration.Get(ctx, v, now)
	if !ok {
		return 0, false
	}

	if s.jitterDuration == nil {
		return duration, true
	}

	jitterDuration, ok := s.jitterDuration.Get(ctx, v, now)
	if !ok {
		return duration, true
	}

	if jitterDuration < duration {
		return jitterDuration, true
	}

	//nolint:gosec
	return duration + time.Duration(rand.Int63n(int64(jitterDuration-duration))), true
}

func (s *LifecycleStage) Next() *internalversion.StageNext {
	return s.next
}

func (s *LifecycleStage) Name() string {
	return s.name
}
