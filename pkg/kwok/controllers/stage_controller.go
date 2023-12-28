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
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/queue"
)

// StageController is a fake resources implementation that can be used to test
type StageController struct {
	clock                                 clock.Clock
	dynamicClient                         dynamic.Interface
	schema                                strategicpatch.LookupPatchMeta
	gvr                                   schema.GroupVersionResource
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	renderer                              gotpl.Renderer
	preprocessChan                        chan *unstructured.Unstructured
	playStageParallelism                  uint
	lifecycle                             resources.Getter[Lifecycle]
	delayQueue                            queue.DelayingQueue[resourceStageJob[*unstructured.Unstructured]]
	delayQueueMapping                     maps.SyncMap[string, resourceStageJob[*unstructured.Unstructured]]
	recorder                              record.EventRecorder
}

// StageControllerConfig is the configuration for the StageController
type StageControllerConfig struct {
	Clock                                 clock.Clock
	DynamicClient                         dynamic.Interface
	Schema                                strategicpatch.LookupPatchMeta
	GVR                                   schema.GroupVersionResource
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	Lifecycle                             resources.Getter[Lifecycle]
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
		schema:                                conf.Schema,
		gvr:                                   conf.GVR,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		delayQueue:                            queue.NewDelayingQueue[resourceStageJob[*unstructured.Unstructured]](conf.Clock),
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

// finalizersModify modify the finalizers of the resource
func (c *StageController) finalizersModify(ctx context.Context, resource *unstructured.Unstructured, finalizers *internalversion.StageFinalizers) (*unstructured.Unstructured, error) {
	ops := finalizersModify(resource.GetFinalizers(), finalizers)
	if len(ops) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(ops)
	if err != nil {
		return nil, err
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"resource", log.KObj(resource),
	)

	nri := c.dynamicClient.Resource(c.gvr)
	var cli dynamic.ResourceInterface = nri
	if ns := resource.GetNamespace(); ns != "" {
		cli = nri.Namespace(ns)
	}
	result, err := cli.Patch(ctx, resource.GetName(), types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch resource finalizers",
				"err", err,
			)
			return nil, nil
		}
		return nil, err
	}
	logger.Info("Patch resource finalizers")
	return result, nil
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
		if apierrors.IsNotFound(err) {
			logger.Warn("Delete resource",
				"err", err,
			)
			return nil
		}
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
				logger.Error("Failed to preprocess node", err,
					"resource", log.KObj(resource),
				)
			}
		}
	}
}

// preprocess the resource and send it to the playStageWorker
func (c *StageController) preprocess(ctx context.Context, resource *unstructured.Unstructured) error {
	key := log.KObj(resource).String()

	resourceJob, ok := c.delayQueueMapping.Load(key)
	if ok && resourceJob.Resource.GetResourceVersion() == resource.GetResourceVersion() {
		return nil
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"resource", key,
	)

	data, err := expression.ToJSONStandard(resource)
	if err != nil {
		return err
	}

	lifecycle := c.lifecycle.Get()
	stage, err := lifecycle.Match(resource.GetLabels(), resource.GetAnnotations(), data)
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
		Resource: resource,
		Stage:    stage,
		Key:      key,
	}
	ok = c.delayQueue.AddAfter(item, delay)
	if !ok {
		logger.Debug("Skip resource",
			"reason", "delayed",
		)
	} else {
		c.delayQueueMapping.Store(key, item)
	}

	return nil
}

// playStageWorker receives the resource from the playStageChan and play the stage
func (c *StageController) playStageWorker(ctx context.Context) {
	for ctx.Err() == nil {
		resource := c.delayQueue.GetOrWait()
		c.delayQueueMapping.Delete(resource.Key)
		c.playStage(ctx, resource.Resource, resource.Stage)
	}
}

// playStage plays the stage
func (c *StageController) playStage(ctx context.Context, resource *unstructured.Unstructured, stage *LifecycleStage) {
	next := stage.Next()
	logger := log.FromContext(ctx)
	logger = logger.With(
		"resource", log.KObj(resource),
		"stage", stage.Name(),
	)

	var (
		result *unstructured.Unstructured
		patch  []byte
		err    error
	)

	if next.Event != nil && c.recorder != nil {
		c.recorder.Event(&corev1.ObjectReference{
			Kind:      "Stage",
			UID:       resource.GetUID(),
			Name:      resource.GetName(),
			Namespace: resource.GetNamespace(),
		}, next.Event.Type, next.Event.Reason, next.Event.Message)
	}

	if next.Finalizers != nil {
		result, err = c.finalizersModify(ctx, resource, next.Finalizers)
		if err != nil {
			logger.Error("Failed to finalizers", err)
			return
		}
	}

	if next.Delete {
		err = c.deleteResource(ctx, resource)
		if err != nil {
			logger.Error("Failed to delete resource", err)
			return
		}
	} else if next.StatusTemplate != "" {
		patch, err = c.configureResource(resource, next.StatusTemplate)
		if err != nil {
			logger.Error("Failed to configure resource", err)
			return
		}
		if patch == nil {
			logger.Debug("Skip resource",
				"reason", "do not need to modify",
			)
		} else {
			result, err = c.patchResource(ctx, resource, patch)
			if err != nil {
				logger.Error("Failed to patch resource", err)
				return
			}
		}
	}

	if result != nil && stage.ImmediateNextStage() {
		logger.Debug("Re-push to preprocessChan",
			"reason", "immediateNextStage is true")
		c.preprocessChan <- result
	}
}

// patchResource patches the resource
func (c *StageController) patchResource(ctx context.Context, resource *unstructured.Unstructured, patch []byte) (*unstructured.Unstructured, error) {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"resource", log.KObj(resource),
	)

	nri := c.dynamicClient.Resource(c.gvr)
	var cli dynamic.ResourceInterface = nri
	if ns := resource.GetNamespace(); ns != "" {
		cli = nri.Namespace(ns)
	}
	result, err := cli.Patch(ctx, resource.GetName(), types.MergePatchType, patch, metav1.PatchOptions{}, "status")
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch resource",
				"err", err,
			)
			return nil, nil
		}
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

func (c *StageController) configureResource(resource *unstructured.Unstructured, template string) ([]byte, error) {
	patch, err := c.computePatch(resource, template)
	if err != nil {
		return nil, err
	}
	if patch == nil {
		return nil, nil
	}

	return json.Marshal(map[string]json.RawMessage{
		"status": patch,
	})
}

func (c *StageController) computePatch(resource *unstructured.Unstructured, tpl string) ([]byte, error) {
	patchData, err := c.renderer.ToJSON(tpl, resource.Object)
	if err != nil {
		return nil, err
	}

	if c.schema != nil {
		status, _, err := unstructured.NestedFieldNoCopy(resource.Object, "status")
		if err != nil {
			return nil, err
		}

		original, err := json.Marshal(status)
		if err != nil {
			return nil, err
		}

		statusPatchMeta, _, err := c.schema.LookupPatchMetadataForStruct("status")
		if err != nil {
			return nil, err
		}
		sum, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(original, patchData, statusPatchMeta)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(sum, original) {
			return nil, nil
		}
	}
	return patchData, nil
}
