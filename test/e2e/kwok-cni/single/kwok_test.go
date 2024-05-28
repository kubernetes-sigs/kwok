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

package single_test

import (
	"context"
	"testing"

	"fmt"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/kwok/test/e2e"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func CaseE(nodeName string) *features.FeatureBuilder {
	

	return features.New("allah").Assess("allaahh",
	func(ctx context.Context, t *testing.T, _ *envconf.Config )context.Context  {
		config, err := rest.InClusterConfig()
		if err != nil {
			fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
				fmt.Errorf("failed to create clientset: %w", err)
		}
	
		for i := 0; i < 300; i++ {
			node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
			if err != nil {
				 fmt.Errorf("failed to get node: %w", err)
			}
	
			if isNodeReady(node) {
				 	fmt.Println("Node is ready")
			}
	
			fmt.Println("Waiting for node to be ready...")
			time.Sleep(1 * time.Second)
		}
	
		  fmt.Errorf("node %s is not ready after 300 seconds", nodeName)
return ctx
		})
}


func isNodeReady(node *v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false

}


func TestNode(t *testing.T) {
	f0 := e2e.CaseNode("node").
		Feature()
	testEnv.Test(t, f0)
}

func TestPod(t *testing.T) {
	f0 := e2e.CasePod("node", namespace).
		Feature()
	testEnv.Test(t, f0)
}
func Test(t *testing.T){
	f0 := CaseE("fake-node").
		Feature()
		testEnv.Test(t, f0)
}