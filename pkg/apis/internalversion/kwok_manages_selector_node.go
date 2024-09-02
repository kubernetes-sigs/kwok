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

package internalversion

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
)

// ManageNodeSelector is a struct that holds how to manage nodes.
type ManageNodeSelector struct {
	// ManageSingleNode is the option to manage a single node name
	ManageSingleNode string

	// Default option to manage (i.e., maintain heartbeat/liveness of) all Nodes or not.
	ManageAllNodes bool

	// Default annotations specified on Nodes to demand manage.
	ManageNodesWithAnnotationSelector string

	// Default labels specified on Nodes to demand manage.
	ManageNodesWithLabelSelector string
}

// IsEmpty means that no node needs to be managed.
func (n ManageNodeSelector) IsEmpty() bool {
	return !n.ManageAllNodes &&
		n.ManageSingleNode == "" &&
		n.ManageNodesWithAnnotationSelector == "" &&
		n.ManageNodesWithLabelSelector == ""
}

// NodeSelector returns the selector of nodes
func (s ManagesSelectors) NodeSelector() (ManageNodeSelector, error) {
	var n *ManageNodeSelector
	for _, sel := range s {
		// TODO: Node, Lease, Pod can be maintained separately by different controllers.
		if sel.Kind == "Pod" &&
			sel.Group == "" &&
			(sel.Version == "" || sel.Version == "v1") {
			return ManageNodeSelector{}, fmt.Errorf("unsupported pod selector type")
		}
		if sel.Kind == "Lease" &&
			sel.Namespace == "kube-node-lease" &&
			sel.Group == "coordination.k8s.io" &&
			(sel.Version == "" || sel.Version == "v1") {
			return ManageNodeSelector{}, fmt.Errorf("unsupported leases.coordination.k8s.io on kube-node-lease selector type")
		}

		if sel.Kind != "Node" || sel.Group != "" || !(sel.Version == "" || sel.Version == "v1") {
			continue
		}

		// TODO: Support multiple nodes selector
		if n != nil {
			return ManageNodeSelector{}, fmt.Errorf("duplicate node selector: %v", sel)
		}

		if sel.Namespace != "" {
			return ManageNodeSelector{}, fmt.Errorf("invalid node selector with namespace %q", sel.Namespace)
		}

		n = &ManageNodeSelector{}

		if sel.Name != "" {
			n.ManageSingleNode = sel.Name
		}
		if len(sel.Labels) != 0 {
			n.ManageNodesWithLabelSelector = labels.Set(sel.Labels).String()
		}
		if len(sel.Annotations) != 0 {
			n.ManageNodesWithAnnotationSelector = labels.Set(sel.Annotations).String()
		}
	}

	if n == nil {
		return ManageNodeSelector{}, nil
	}

	return *n, nil
}
