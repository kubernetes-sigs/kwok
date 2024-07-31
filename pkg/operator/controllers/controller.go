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

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/kwok/pkg/client/clientset/versioned"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/sets"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

type Controller struct {
	typedClient     kubernetes.Interface
	typedKwokClient versioned.Interface
	sourceNamespace string
	sourceName      string

	podTemplate            *corev1.PodTemplateSpec
	controllersInformer    *informer.Informer[*v1alpha1.Controller, *v1alpha1.ControllerList]
	controllersCacheGetter informer.Getter[*v1alpha1.Controller]
	controllersChan        chan informer.Event[*v1alpha1.Controller]
}

// ControllerConfig is the configuration for the controller
type ControllerConfig struct {
	TypedClient     kubernetes.Interface
	TypedKwokClient versioned.Interface

	SourceNamespace string
	SourceName      string
}

func NewController(conf ControllerConfig) (*Controller, error) {
	c := &Controller{
		typedClient:     conf.TypedClient,
		typedKwokClient: conf.TypedKwokClient,
		sourceNamespace: conf.SourceNamespace,
		sourceName:      conf.SourceName,
	}

	return c, nil
}

func (c *Controller) Start(ctx context.Context) error {
	c.controllersInformer = informer.NewInformer[*v1alpha1.Controller, *v1alpha1.ControllerList](c.typedKwokClient.OperatorV1alpha1().Controllers(""))
	c.controllersChan = make(chan informer.Event[*v1alpha1.Controller], 1)
	controllersCacheGetter, err := c.controllersInformer.WatchWithCache(ctx, informer.Option{}, c.controllersChan)
	if err != nil {
		return err
	}
	c.controllersCacheGetter = controllersCacheGetter

	go c.syncWorker(ctx)
	return nil
}

func (c *Controller) scaleDeployment(ctx context.Context, replicas int32) error {
	deploy, err := c.typedClient.AppsV1().
		Deployments(c.sourceNamespace).
		Get(ctx, c.sourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting deployment: %w", err)
	}
	c.podTemplate = deploy.Spec.Template.DeepCopy()

	if *deploy.Spec.Replicas == replicas {
		return nil
	}

	_, err = c.typedClient.AppsV1().
		Deployments(c.sourceNamespace).
		UpdateScale(ctx, c.sourceName, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name: c.sourceName,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: replicas,
			},
		}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating deployment scale: %w", err)
	}
	return nil
}

func (c *Controller) syncWorker(ctx context.Context) {
	s := sets.Sets[log.ObjectRef]{}

	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop controller sync worker")
			return
		case event := <-c.controllersChan:
			cc := event.Object

			p := c.typedClient.CoreV1().
				Pods(cc.Namespace)

			if event.Type == informer.Deleted {
				err := p.Delete(ctx, cc.GetName(), metav1.DeleteOptions{})
				if err != nil {
					logger.Error("can't delete the origin pod", err)
				}
				s.Delete(log.KObj(cc))
				if s.Len() == 0 {
					err := c.scaleDeployment(ctx, 1)
					if err != nil {
						logger.Error("can't sync deployment", err)
						continue
					}
				}
				continue
			}
			if s.Len() == 0 {
				err := c.scaleDeployment(ctx, 0)
				if err != nil {
					logger.Error("can't sync deployment", err)
					continue
				}
			}
			s.Insert(log.KObj(cc))

			out := internalversion.Controller{}

			err := internalversion.Convert_v1alpha1_Controller_To_internalversion_Controller(cc, &out, nil)
			if err != nil {
				logger.Error("can't convert controller to internal version", err)
				continue
			}

			pod := controllerPod(&out, c.podTemplate.Spec)

			oriPod, err := p.Get(ctx, cc.GetName(), metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					logger.Error("can't get the origin pod", err)
					continue
				}
			} else {
				if reflect.DeepEqual(pod.Spec, oriPod.Spec) {
					continue
				}

				err = p.Delete(ctx, cc.GetName(), metav1.DeleteOptions{
					GracePeriodSeconds: format.Ptr[int64](0),
				})
				if err != nil {
					logger.Error("can't delete the origin pod", err)
					continue
				}
			}

			_, err = p.Create(ctx, pod, metav1.CreateOptions{})
			if err != nil {
				logger.Error("can't create the pod", err)
				continue
			}
		}
	}
}

func controllerPod(c *internalversion.Controller, ps corev1.PodSpec) *corev1.Pod {
	p := ps.DeepCopy()
	if len(c.Spec.Manages) != 0 {
		for i, container := range p.Containers {
			if c.Name != "kwok-controller" {
				continue
			}
			args := container.Args
			args = slices.Filter(args, func(s string) bool {
				return !strings.HasPrefix(s, "--manage")
			})
			for _, m := range c.Spec.Manages {
				args = append(args, "--manage="+m.String())
			}

			p.Containers[i].Args = args
			break
		}
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.Name,
			Namespace:   c.Namespace,
			Labels:      c.Labels,
			Annotations: c.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         v1alpha1.GroupVersion.String(),
					Kind:               v1alpha1.ControllerKind,
					Name:               c.Name,
					UID:                c.ObjectMeta.UID,
					Controller:         format.Ptr(true),
					BlockOwnerDeletion: format.Ptr(true),
				},
			},
		},
		Spec: *p,
	}
	return &pod
}
