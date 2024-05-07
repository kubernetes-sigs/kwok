/*
Copyright 2023 The Kubernetes Authors.

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
	"context"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/cel"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/sets"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// StagesManagerConfig is the configuration for a stages manager
type StagesManagerConfig struct {
	Env         *cel.Environment
	StageGetter resources.DynamicGetter[[]*internalversion.Stage]
	StartFunc   func(ctx context.Context, ref internalversion.StageResourceRef, lifecycle resources.Getter[lifecycle.Lifecycle]) error
}

// StagesManager is a stages manager
// It is a dynamic getter for stages and start a stage controller
type StagesManager struct {
	env         *cel.Environment
	stageGetter resources.DynamicGetter[[]*internalversion.Stage]
	startFunc   func(ctx context.Context, ref internalversion.StageResourceRef, lifecycle resources.Getter[lifecycle.Lifecycle]) error
	cache       map[internalversion.StageResourceRef]context.CancelCauseFunc
}

// NewStagesManager creates a stage controller manager
func NewStagesManager(conf StagesManagerConfig) *StagesManager {
	return &StagesManager{
		env:         conf.Env,
		stageGetter: conf.StageGetter,
		startFunc:   conf.StartFunc,
		cache:       map[internalversion.StageResourceRef]context.CancelCauseFunc{},
	}
}

// Start starts the stages manager
func (c *StagesManager) Start(ctx context.Context) error {
	go c.run(ctx)

	return nil
}

func (c *StagesManager) run(ctx context.Context) {
	sync := c.stageGetter.Sync()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sync:
			c.manage(ctx)
		}
	}
}

//nolint:govet
func (c *StagesManager) manage(ctx context.Context) {
	set := sets.NewSets[internalversion.StageResourceRef]()
	for _, stage := range c.stageGetter.Get() {
		set.Insert(stage.Spec.ResourceRef)
	}

	logger := log.FromContext(ctx)

	for ref := range set {
		_, ok := c.cache[ref]
		if ok {
			continue
		}

		lifecycle := resources.NewFilter[lifecycle.Lifecycle, []*internalversion.Stage](c.stageGetter, func(stages []*internalversion.Stage) lifecycle.Lifecycle {
			return slices.FilterAndMap(stages, func(stage *internalversion.Stage) (*lifecycle.Stage, bool) {
				if stage.Spec.ResourceRef != ref {
					return nil, false
				}

				lifecycleStage, err := lifecycle.NewStage(stage, c.env)
				if err != nil {
					logger.Error("failed to create lifecycle stage", err, "ref", ref)
					return nil, false
				}
				return lifecycleStage, true
			})
		})

		cancelctx, cancel := context.WithCancelCause(ctx)
		err := c.startFunc(cancelctx, ref, lifecycle)
		if err != nil {
			logger.Error("failed to start controller", err, "ref", ref)
			continue
		}

		logger.Info("Start stage controller", "ref", ref)
		c.cache[ref] = cancel
	}

	for ref, cancel := range c.cache {
		if _, ok := set[ref]; ok {
			continue
		}

		logger.Info("Stop stage controller", "ref", ref)
		cancel(context.Canceled)
		delete(c.cache, ref)
	}
}
