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

package e2e

import (
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseEnsurePodScheduledConditions is a very error-prone case where we've mishandled the PodScheduled
// https://github.com/kubernetes-sigs/kwok/issues/1243
func CaseEnsurePodScheduledConditions() *features.FeatureBuilder {
	node := helper.NewNodeBuilder("node0").
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		Build()
	pod1 := helper.NewPodBuilder("pod1").
		WithNodeName("node0").
		Build()
	return features.New("Ensure PodScheduled conditions").
		Setup(helper.CreateNode(node)).
		Teardown(helper.DeleteNode(node)).
		Assess("create pod0", helper.CreatePod(pod0)).
		Assess("delete pod0", helper.DeletePod(pod0)).
		Assess("create pod1", helper.CreatePod(pod1)).
		Assess("delete pod1", helper.DeletePod(pod1))
}
