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

// Package crd contains the default stages for CRD.
package crd

import (
	_ "embed"
)

var (
	// Stage is the custom resource definition for stages.
	//go:embed bases/kwok.x-k8s.io_stages.yaml
	Stage []byte

	// Attach is the custom resource definition for attaches.
	//go:embed bases/kwok.x-k8s.io_attaches.yaml
	Attach []byte

	// ClusterAttach is the custom resource definition for cluster attaches.
	//go:embed bases/kwok.x-k8s.io_clusterattaches.yaml
	ClusterAttach []byte

	// Exec is the custom resource definition for execs.
	//go:embed bases/kwok.x-k8s.io_execs.yaml
	Exec []byte

	// ClusterExec is the custom resource definition for cluster execs.
	//go:embed bases/kwok.x-k8s.io_clusterexecs.yaml
	ClusterExec []byte

	// Logs is the custom resource definition for logs.
	//go:embed bases/kwok.x-k8s.io_logs.yaml
	Logs []byte

	// ClusterLogs is the custom resource definition for cluster logs.
	//go:embed bases/kwok.x-k8s.io_clusterlogs.yaml
	ClusterLogs []byte

	// PortForward is the custom resource definition for port forwards.
	//go:embed bases/kwok.x-k8s.io_portforwards.yaml
	PortForward []byte

	// ClusterPortForward is the custom resource definition for cluster port forwards.
	//go:embed bases/kwok.x-k8s.io_clusterportforwards.yaml
	ClusterPortForward []byte

	// Metric is the custom resource definition for metrics.
	//go:embed bases/kwok.x-k8s.io_metrics.yaml
	Metric []byte
)
