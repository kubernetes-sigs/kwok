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
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/stages"
)

func TestController(t *testing.T) {
	nodes := []runtime.Object{
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-0",
				Labels: map[string]string{
					"manage-by-kwok": "true",
				},
			},
			Status: corev1.NodeStatus{
				Phase: corev1.NodePending,
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
				Annotations: map[string]string{
					"manage-by-kwok": "true",
				},
			},
			Status: corev1.NodeStatus{
				Phase: corev1.NodePending,
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
			},
			Status: corev1.NodeStatus{
				Phase: corev1.NodePending,
			},
		},
	}

	nodeStages, _ := NewStagesFromYaml([]byte(stages.DefaultNodeStages))
	podStages, _ := NewStagesFromYaml([]byte(stages.DefaultPodStages))

	tests := []struct {
		name          string
		conf          Config
		wantNodePhase map[string]corev1.NodePhase
		wantErr       bool
	}{
		{
			name: "node controller test: manage all nodes",
			conf: Config{
				ClientSet:      fake.NewSimpleClientset(nodes...),
				ManageAllNodes: true,
				NodeStages:     nodeStages,
				PodStages:      podStages,
				CIDR:           "10.0.0.1/24",
			},
			wantNodePhase: map[string]corev1.NodePhase{
				"node-0": corev1.NodeRunning,
				"node-1": corev1.NodeRunning,
				"node-2": corev1.NodeRunning,
			},
			wantErr: false,
		},
		{
			name: "node controller test: manage nodes with label selector `manage-by-kwok=true`",
			conf: Config{
				ClientSet:                    fake.NewSimpleClientset(nodes...),
				ManageNodesWithLabelSelector: "manage-by-kwok",
				NodeStages:                   nodeStages,
				PodStages:                    podStages,
				CIDR:                         "10.0.0.1/24",
			},
			wantNodePhase: map[string]corev1.NodePhase{
				"node-0": corev1.NodeRunning,
				"node-1": corev1.NodePending,
				"node-2": corev1.NodePending,
			},
			wantErr: false,
		},
		{
			name: "node controller test: manage nodes with annotation selector `manage-by-kwok=true`",
			conf: Config{
				ClientSet:                         fake.NewSimpleClientset(nodes...),
				ManageNodesWithAnnotationSelector: "manage-by-kwok=true",
				NodeStages:                        nodeStages,
				PodStages:                         podStages,
				CIDR:                              "10.0.0.1/24",
			},
			wantNodePhase: map[string]corev1.NodePhase{
				"node-0": corev1.NodePending,
				"node-1": corev1.NodeRunning,
				"node-2": corev1.NodePending,
			},
			wantErr: false,
		},
	}

	ctx := context.Background()
	ctx = log.NewContext(ctx, log.NewLogger(os.Stderr, log.DebugLevel))
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctr, err := NewController(tt.conf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewController() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := ctr.Start(ctx); err != nil {
				t.Fatalf("failed to start nodes controller: %v", err)
			}

			// wait for nodes to be right phase indicated by `tt.wantNodePhase`
			err = wait.PollUntilWithContext(ctx, time.Second, func(ctx context.Context) (done bool, err error) {
				list, err := ctr.clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				if err != nil {
					t.Log(fmt.Errorf("failed to list nodes, err: %w", err))
					return false, nil
				}

				for _, node := range list.Items {
					wantNodePhase := tt.wantNodePhase[node.Name]
					if node.Status.Phase != wantNodePhase {
						t.Log(fmt.Errorf("node %s phase is %s, want %s", node.Name, node.Status.Phase, wantNodePhase))
						return false, nil
					}
				}
				return true, nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
