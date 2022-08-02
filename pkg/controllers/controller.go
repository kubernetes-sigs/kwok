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
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"sigs.k8s.io/kwok/pkg/logger"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var (
	startTime = time.Now().Format(time.RFC3339)

	funcMap = template.FuncMap{
		"Now": func() string {
			return time.Now().Format(time.RFC3339)
		},
		"StartTime": func() string {
			return startTime
		},
		"YAML": func(s interface{}, indent ...int) (string, error) {
			d, err := yaml.Marshal(s)
			if err != nil {
				return "", err
			}

			data := string(d)
			if len(indent) == 1 && indent[0] > 0 {
				pad := strings.Repeat(" ", indent[0]*2)
				data = strings.Replace("\n"+data, "\n", "\n"+pad, -1)
			}
			return data, nil
		},
	}
)

// Controller is a fake kubelet implementation that can be used to test
type Controller struct {
	nodes *NodeController
	pods  *PodController
}

type Config struct {
	ClientSet                             kubernetes.Interface
	ManageAllNodes                        bool
	ManageNodesWithAnnotationSelector     string
	ManageNodesWithLabelSelector          string
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	CIDR                                  string
	NodeIP                                string
	Logger                                logger.Logger
	PodStatusTemplate                     string
	NodeInitializationTemplate            string
	NodeHeartbeatTemplate                 string
}

// NewController creates a new fake kubelet controller
func NewController(conf Config) (*Controller, error) {
	var nodeSelectorFunc func(node *corev1.Node) bool
	if conf.ManageAllNodes {
		nodeSelectorFunc = func(node *corev1.Node) bool {
			return true
		}
		conf.ManageNodesWithAnnotationSelector = ""
		conf.ManageNodesWithLabelSelector = ""
	} else if conf.ManageNodesWithAnnotationSelector != "" {
		selector, err := labels.Parse(conf.ManageNodesWithAnnotationSelector)
		if err != nil {
			return nil, err
		}
		nodeSelectorFunc = func(node *corev1.Node) bool {
			return selector.Matches(labels.Set(node.Annotations))
		}
	}

	var lockPodsOnNodeFunc func(ctx context.Context, nodeName string) error

	nodes, err := NewNodeController(NodeControllerConfig{
		ClientSet:                             conf.ClientSet,
		NodeIP:                                conf.NodeIP,
		DisregardStatusWithAnnotationSelector: conf.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      conf.DisregardStatusWithLabelSelector,
		ManageNodesWithAnnotationSelector:     conf.ManageNodesWithAnnotationSelector,
		ManageNodesWithLabelSelector:          conf.ManageNodesWithLabelSelector,
		NodeSelectorFunc:                      nodeSelectorFunc,
		LockPodsOnNodeFunc: func(ctx context.Context, nodeName string) error {
			return lockPodsOnNodeFunc(ctx, nodeName)
		},
		NodeStatusTemplate:       conf.NodeInitializationTemplate,
		NodeHeartbeatTemplate:    conf.NodeHeartbeatTemplate,
		NodeHeartbeatInterval:    30 * time.Second,
		NodeHeartbeatParallelism: 16,
		LockNodeParallelism:      16,
		Logger:                   conf.Logger,
		FuncMap:                  funcMap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create nodes controller: %v", err)
	}

	pods, err := NewPodController(PodControllerConfig{
		ClientSet:                             conf.ClientSet,
		NodeIP:                                conf.NodeIP,
		CIDR:                                  conf.CIDR,
		DisregardStatusWithAnnotationSelector: conf.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      conf.DisregardStatusWithLabelSelector,
		PodStatusTemplate:                     conf.PodStatusTemplate,
		LockPodParallelism:                    16,
		DeletePodParallelism:                  16,
		NodeHasFunc:                           nodes.Has, // just handle pods that are on nodes we have
		Logger:                                conf.Logger,
		FuncMap:                               funcMap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pods controller: %v", err)
	}

	lockPodsOnNodeFunc = pods.LockPodsOnNode

	n := &Controller{
		pods:  pods,
		nodes: nodes,
	}

	return n, nil
}

func (c *Controller) Start(ctx context.Context) error {
	err := c.pods.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start pods controller: %v", err)
	}
	err = c.nodes.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start nodes controller: %v", err)
	}
	return nil
}
