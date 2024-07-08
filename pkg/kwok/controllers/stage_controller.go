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
	"fmt"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// StageController is a fake resources implementation that can be used to test
type StageController struct {
	clock                                 clock.Clock
	dynamicClient                         dynamic.Interface
	impersonatingDynamicClient            client.DynamicClientImpersonator
	schema                                strategicpatch.LookupPatchMeta
	gvr                                   schema.GroupVersionResource
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	renderer                              gotpl.Renderer
	preprocessChan                        chan *unstructured.Unstructured
	playStageParallelism                  uint
	lifecycle                             resources.Getter[lifecycle.Lifecycle]
	delayQueue                            queue.WeightDelayingQueue[resourceStageJob[*unstructured.Unstructured]]
	backoff                               wait.Backoff
	delayQueueMapping                     maps.SyncMap[string, resourceStageJob[*unstructured.Unstructured]]
	recorder                              record.EventRecorder
}

// StageControllerConfig is the configuration for the StageController
type StageControllerConfig struct {
	Clock                                 clock.Clock
	DynamicClient                         dynamic.Interface
	ImpersonatingDynamicClient            client.DynamicClientImpersonator
	Schema                                strategicpatch.LookupPatchMeta
	GVR                                   schema.GroupVersionResource
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	Lifecycle                             resources.Getter[lifecycle.Lifecycle]
	PlayStageParallelism                  uint
	FuncMap                               gotpl.FuncMap
	Recorder                              record.EventRecorder
}

// NewStageController creates a new fake resources controller
func NewStageController(conf StageControllerConfig) (*StageController, error) {
	if conf.PlayStageParallelism <= 0 {
		return nil, fmt.Errorf("playStageParallelism must be greater than 0")
	}

	disregardStatusWithAnnotationSelector, err := labelsParse(conf.DisregardStatusWithAnnotationSelector)
	if err != nil {
		return nil, err
	}

	disregardStatusWithLabelSelector, err := labelsParse(conf.DisregardStatusWithLabelSelector)
	if err != nil {
		return nil, err
	}

	if conf.Clock == nil {
		conf.Clock = clock.RealClock{}
	}

	c := &StageController{
		clock:                                 conf.Clock,
		dynamicClient:                         conf.DynamicClient,
		impersonatingDynamicClient:            conf.ImpersonatingDynamicClient,
		schema:                                conf.Schema,
		gvr:                                   conf.GVR,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		delayQueue:                            queue.NewWeightDelayingQueue[resourceStageJob[*unstructured.Unstructured]](conf.Clock),
		backoff:                               defaultBackoff(),
		lifecycle:                             conf.Lifecycle,
		playStageParallelism:                  conf.PlayStageParallelism,
		preprocessChan:                        make(chan *unstructured.Unstructured),
		recorder:                              conf.Recorder,
	}

	c.renderer = gotpl.NewRenderer(conf.FuncMap)
	return c, nil
}

// Start starts the fake resource controller
// It will modify the resources status to we want
func (c *StageController) Start(ctx context.Context, events <-chan informer.Event[*unstructured.Unstructured]) error {
	go c.preprocessWorker(ctx)
	for i := uint(0); i < c.playStageParallelism; i++ {
		go c.playStageWorker(ctx)
	}
	go c.watchResources(ctx, events)
	return nil
}

// deleteResource deletes a resource
func (c *StageController) deleteResource(ctx context.Context, resource *unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"resource", log.KObj(resource),
	)

	nri := c.dynamicClient.Resource(c.gvr)
	var cli dynamic.ResourceInterface = nri
	if ns := resource.GetNamespace(); ns != "" {
		cli = nri.Namespace(ns)
	}
	err := cli.Delete(ctx, resource.GetName(), deleteOpt)
	if err != nil {
		return err
	}

	logger.Info("Delete resource")
	return nil
}

// preprocessWorker receives the resource from the preprocessChan and preprocess it
func (c *StageController) preprocessWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop preprocess worker")
			return
		case resource := <-c.preprocessChan:
			err := c.preprocess(ctx, resource)
			if err != nil {
				logger.Error("Failed to preprocess resource", err,
					"resource", log.KObj(resource),
				)
			}
		}
	}
}

// preprocess the resource and send it to the playStageWorker
func (c *StageController) preprocess(ctx context.Context, resource *unstructured.Unstructured) error {
	key := log.KObj(resource).String()

	logger := log.FromContext(ctx)
	logger = logger.With(
		"resource", key,
	)

	resourceJob, ok := c.delayQueueMapping.Load(key)
	if ok {
		if resourceJob.Resource.GetResourceVersion() == resource.GetResourceVersion() {
			logger.Debug("Skip resource",
				"reason", "resource version not changed",
				"stage", resourceJob.Stage.Name(),
			)
			return nil
		}
	}

	data, err := expression.ToJSONStandard(resource)
	if err != nil {
		return err
	}

	lc := c.lifecycle.Get()
	stage, err := lc.Match(ctx, resource.GetLabels(), resource.GetAnnotations(), data)
	if err != nil {
		return fmt.Errorf("stage match: %w", err)
	}
	if stage == nil {
		logger.Debug("Skip resource",
			"reason", "not match any stages",
		)
		return nil
	}

	now := c.clock.Now()
	delay, _ := stage.Delay(ctx, data, now)

	if delay != 0 {
		stageName := stage.Name()
		logger.Debug("Delayed play stage",
			"delay", delay,
			"stage", stageName,
		)
	}

	item := resourceStageJob[*unstructured.Unstructured]{
		Resource:   resource,
		Stage:      stage,
		Key:        key,
		RetryCount: new(uint64),
	}

	// we add a normal(fresh) stage job with weight 0,
	// resulting in that it will always be processed with high priority compared to those retry ones
	c.addStageJob(ctx, item, delay, 0)
	return nil
}

// playStageWorker receives the resource from the playStageChan and play the stage
func (c *StageController) playStageWorker(ctx context.Context) {
	logger := log.FromContext(ctx)

	for ctx.Err() == nil {
		resource, ok := c.delayQueue.GetOrWaitWithDone(ctx.Done())
		if !ok {
			return
		}
		c.delayQueueMapping.Delete(resource.Key)
		needRetry, err := c.playStage(ctx, resource.Resource, resource.Stage)
		if err != nil {
			logger.Error("failed to apply stage", err,
				"resource", resource.Key,
				"stage", resource.Stage.Name(),
			)
		}
		if needRetry {
			retryCount := atomic.AddUint64(resource.RetryCount, 1) - 1
			logger.Info("retrying for failed job",
				"resource", resource.Key,
				"stage", resource.Stage.Name(),
				"retry", retryCount,
			)
			// for failed jobs, we re-push them into the queue with a lower weight
			// and a backoff period to avoid blocking normal tasks
			retryDelay := backoffDelayByStep(retryCount, c.backoff)
			c.addStageJob(ctx, resource, retryDelay, 1)
		}
	}
}

// playStage plays the stage.
// The returned boolean indicates whether the applying action needs to be retried.
func (c *StageController) playStage(ctx context.Context, resource *unstructured.Unstructured, stage *lifecycle.Stage) (bool, error) {
	next := stage.Next()
	logger := log.FromContext(ctx)
	logger = logger.With(
		"resource", log.KObj(resource),
		"stage", stage.Name(),
	)

	var (
		result *unstructured.Unstructured
		err    error
	)

	if event := next.Event(); event != nil && c.recorder != nil {
		c.recorder.Event(&corev1.ObjectReference{
			APIVersion: resource.GetAPIVersion(),
			Kind:       resource.GetKind(),
			UID:        resource.GetUID(),
			Name:       resource.GetName(),
			Namespace:  resource.GetNamespace(),
		}, event.Type, event.Reason, event.Message)
	}

	patch, err := next.Finalizers(resource.GetFinalizers())
	if err != nil {
		return false, fmt.Errorf("failed to get finalizers for resource %s: %w", resource.GetName(), err)
	}
	if patch != nil {
		result, err = c.patchResource(ctx, resource, patch)
		if err != nil {
			return shouldRetry(err), fmt.Errorf("failed to patch the finalizer of resource %s: %w", resource.GetName(), err)
		}
	}

	if next.Delete() {
		err = c.deleteResource(ctx, resource)
		if err != nil {
			return shouldRetry(err), fmt.Errorf("failed to delete resource %s: %w", resource.GetName(), err)
		}
		result = nil
	} else {
		patches, err := next.Patches(resource.Object, c.renderer)
		if err != nil {
			return false, fmt.Errorf("failed to get patches for resource %s: %w", resource.GetName(), err)
		}
		for _, patch := range patches {
			changed, err := checkNeedPatch(resource.Object, patch.Data, patch.Type, c.schema)
			if err != nil {
				return false, fmt.Errorf("failed to check need patch for resource %s: %w", resource.GetName(), err)
			}

			if !changed {
				logger.Debug("Skip resource",
					"reason", "do not need to modify",
				)
			} else {
				result, err = c.patchResource(ctx, resource, patch)
				if err != nil {
					return shouldRetry(err), fmt.Errorf("failed to patch resource %s: %w", resource.GetName(), err)
				}
			}
		}
	}

	if result != nil && stage.ImmediateNextStage() {
		logger.Debug("Re-push to preprocessChan",
			"reason", "immediateNextStage is true")
		c.preprocessChan <- result
	}
	return false, nil
}

// patchResource patches the resource
func (c *StageController) patchResource(ctx context.Context, resource *unstructured.Unstructured, patch *lifecycle.Patch) (*unstructured.Unstructured, error) {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"resource", log.KObj(resource),
	)

	nri := c.dynamicClient.Resource(c.gvr)
	if patch.Impersonation != nil {
		logger.With(
			"impersonate", patch.Impersonation.Username,
		)

		dc, err := c.impersonatingDynamicClient.Impersonate(rest.ImpersonationConfig{UserName: patch.Impersonation.Username})
		if err != nil {
			logger.Error("error getting impersonating client", err)
			return nil, err
		}
		nri = dc.Resource(c.gvr)
	}
	var cli dynamic.ResourceInterface = nri
	if ns := resource.GetNamespace(); ns != "" {
		cli = nri.Namespace(ns)
	}
	subresource := []string{}
	if patch.Subresource != "" {
		logger = logger.With(
			"subresource", patch.Subresource,
		)
		subresource = []string{patch.Subresource}
	}

	result, err := cli.Patch(ctx, resource.GetName(), patch.Type, patch.Data, metav1.PatchOptions{}, subresource...)
	if err != nil {
		return nil, err
	}
	logger.Info("Patch resource")
	return result, nil
}

func (c *StageController) need(resource *unstructured.Unstructured) bool {
	if c.disregardStatusWithAnnotationSelector != nil &&
		len(resource.GetAnnotations()) != 0 &&
		c.disregardStatusWithAnnotationSelector.Matches(labels.Set(resource.GetAnnotations())) {
		return false
	}

	if c.disregardStatusWithLabelSelector != nil &&
		len(resource.GetLabels()) != 0 &&
		c.disregardStatusWithLabelSelector.Matches(labels.Set(resource.GetLabels())) {
		return false
	}
	return true
}

// watchResources watch resources and send to preprocessChan
func (c *StageController) watchResources(ctx context.Context, events <-chan informer.Event[*unstructured.Unstructured]) {
	logger := log.FromContext(ctx)
loop:
	for {
		select {
		case event, ok := <-events:
			if !ok {
				break loop
			}

			switch event.Type {
			case informer.Added, informer.Modified, informer.Sync:
				resource := event.Object
				if c.need(resource) {
					c.preprocessChan <- resource.DeepCopy()
				} else {
					logger.Debug("Skip resource",
						"reason", "not managed",
						"event", event.Type,
						"resource", log.KObj(resource),
					)
				}

			case informer.Deleted:
				resource := event.Object
				if c.need(resource) {
					// Cancel delay job
					key := log.KObj(resource).String()
					resourceJob, ok := c.delayQueueMapping.LoadAndDelete(key)
					if ok {
						c.delayQueue.Cancel(resourceJob)
					}
				}
			}
		case <-ctx.Done():
			break loop
		}
	}
	logger.Info("Stop watch resources")
}

// addStageJob adds a stage to be applied into the underlying weight delay queue and the associated helper map
func (c *StageController) addStageJob(ctx context.Context, job resourceStageJob[*unstructured.Unstructured], delay time.Duration, weight int) {
	old, loaded := c.delayQueueMapping.Swap(job.Key, job)
	if loaded {
		if !c.delayQueue.Cancel(old) {
			logger := log.FromContext(ctx)
			logger.Debug("Failed to cancel stage",
				"stage", job.Stage.Name(),
			)
		}
	}
	c.delayQueue.AddWeightAfter(job, weight, delay)
}
