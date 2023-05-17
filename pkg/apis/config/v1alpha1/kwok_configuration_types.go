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
	// KwokConfigurationKind is the kind of the KwokConfiguration.
	KwokConfigurationKind = "KwokConfiguration"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KwokConfiguration provides configuration for the Kwok.
type KwokConfiguration struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Options holds information about the default value.
	Options KwokConfigurationOptions `json:"options,omitempty"`
}

// KwokConfigurationOptions holds information about the options.
type KwokConfigurationOptions struct {
	// The default IP assigned to the Pod on maintained Nodes.
	// is the default value for flag --cidr
	// +default="10.0.0.1/24"
	CIDR string `json:"cidr,omitempty"`

	// The ip of all nodes maintained by the Kwok
	// is the default value for flag --node-ip
	NodeIP string `json:"nodeIP,omitempty"`

	// The name of all nodes maintained by the Kwok
	// is the default value for flag --node-name
	NodeName string `json:"nodeName,omitempty"`

	// The port of all nodes maintained by the Kwok
	// is the default value for flag --node-port
	NodePort int `json:"nodePort,omitempty"`

	// TLSCertFile is the file containing x509 Certificate for HTTPS.
	// If HTTPS serving is enabled, and --tls-cert-file and --tls-private-key-file
	// is the default value for flag --tls-cert-file
	TLSCertFile string `json:"tlsCertFile,omitempty"`

	// TLSPrivateKeyFile is the ile containing x509 private key matching --tls-cert-file.
	// is the default value for flag --tls-private-key-file
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`

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

	// ServerAddress is server address of the Kwok.
	// is the default value for flag --server-address
	ServerAddress string `json:"serverAddress,omitempty"`

	// Experimental support for getting pod ip from CNI, for CNI-related components, Only works with Linux.
	// is the default value for flag --experimental-enable-cni
	// +default=false
	EnableCNI *bool `json:"experimentalEnableCNI,omitempty"`

	// enableDebuggingHandlers enables server endpoints for log collection
	// and local running of containers and commands
	// +default=true
	EnableDebuggingHandlers *bool `json:"enableDebuggingHandlers,omitempty"`

	// enableContentionProfiling enables lock contention profiling, if enableDebuggingHandlers is true.
	// +default=false
	EnableContentionProfiling *bool `json:"enableContentionProfiling,omitempty"`

	// EnableProfiling enables /debug/pprof handler, if enableDebuggingHandlers is true.
	// +default=true
	EnableProfilingHandler *bool `json:"enableProfilingHandler,omitempty"`

	// PodPlayStageParallelism is the number of PodPlayStages that are allowed to run in parallel.
	// +default=4
	PodPlayStageParallelism uint `json:"podPlayStageParallelism,omitempty"`

	// NodePlayStageParallelism is the number of NodePlayStages that are allowed to run in parallel.
	// +default=4
	NodePlayStageParallelism uint `json:"nodePlayStageParallelism,omitempty"`

	// NodeLeaseDurationSeconds is the duration the Kubelet will set on its corresponding Lease.
	NodeLeaseDurationSeconds uint `json:"nodeLeaseDurationSeconds,omitempty"`

	// NodeLeaseParallelism is the number of NodeLeases that are allowed to be processed in parallel.
	// +default=4
	NodeLeaseParallelism uint `json:"nodeLeaseParallelism,omitempty"`
}
