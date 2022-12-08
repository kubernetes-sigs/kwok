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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KwokConfigurationKind = "KwokConfiguration"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KwokConfiguration provides configuration for the Kwok.
type KwokConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Options holds information about the default value.
	Options KwokConfigurationOptions `json:"options,omitempty"`
}

type KwokConfigurationOptions struct {
	// The default IP assigned to the Pod on maintained Nodes.
	// is the default value for flag --cidr
	// +default="10.0.0.1/24"
	CIDR string `json:"cidr,omitempty"`

	// The ip of all nodes maintained by the Kwok
	// is the default value for flag --node-ip
	// +default="196.168.0.1"
	NodeIP string `json:"nodeIP,omitempty"`

	// Default option to manage (i.e., maintain heartbeat/liveness of) all Nodes or not.
	// is the default value for flag --manage-all-nodes
	// +default=false
	ManageAllNodes *bool `json:"manageAllNodes,omitempty"`

	// Default annotations specified on Nodes to demand manage.
	// Note: when `all-node-manage` is specified as true, this is a no-op.
	// is the default value for flag --manage-nodes-with-annotation-selector
	ManageNodesWithAnnotationSelector string `json:"manageNodesWithAnnotationSelector,omitempty"`

	// Default labels specified on Nodes to demand manage.
	// Note: when `all-node-manage` is specified as true, this is a no-op.
	// is the default value for flag --manage-nodes-with-label-selector
	ManageNodesWithLabelSelector string `json:"manageNodesWithLabelSelector,omitempty"`

	// If a Node/Pod is on a managed Node and has this annotation status will not be modified
	// is the default value for flag --disregard-status-with-annotation-selector
	DisregardStatusWithAnnotationSelector string `json:"disregardStatusWithAnnotationSelector,omitempty"`

	// If a Node/Pod is on a managed Node and has this label status will not be modified
	// is the default value for flag --disregard-status-with-label-selector
	DisregardStatusWithLabelSelector string `json:"disregardStatusWithLabelSelector,omitempty"`

	// is the default value for flag --server-address
	ServerAddress string `json:"serverAddress,omitempty"`

	// Experimental support for getting pod ip from CNI, for CNI-related components
	// is the default value for flag --experimental-enable-cni
	// +default=false
	EnableCNI *bool `json:"experimentalEnableCNI,omitempty"`
}
