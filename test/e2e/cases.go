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

package e2e

import (
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseNode creates a feature that tests node creation and deletion
func CaseNode(nodeName string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder(nodeName).
		Build()
	return features.New("Node can be created and deleted").
		Assess("create node", helper.CreateNode(node)).
		Assess("delete node", helper.DeleteNode(node))
}

// CasePod creates a feature that tests pod creation and deletion
func CasePod(nodeName string, namespace string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder(nodeName).
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		Build()
	pod1 := helper.NewPodBuilder("pod1").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		WithHostNetwork(true).
		Build()
	return features.New("Pod can be created and deleted").
		Setup(helper.CreateNode(node)).
		Teardown(helper.DeleteNode(node)).
		Assess("create pod", helper.CreatePod(pod0)).
		Assess("create pod for host network", helper.CreatePod(pod1)).
		Assess("delete pod", helper.DeletePod(pod0)).
		Assess("delete pod for host network", helper.DeletePod(pod1))
}
