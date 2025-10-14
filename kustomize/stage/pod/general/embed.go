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

// Package general contains the general pod stages for kwok.
package general

import (
	_ "embed"
)

var (
	// DefaultPodCreate is the default pod create yaml.
	//go:embed pod-create.yaml
	DefaultPodCreate string

	// DefaultPodInitContainerRunning is the default pod init container running yaml.
	//go:embed pod-init-container-running.yaml
	DefaultPodInitContainerRunning string

	// DefaultPodInitContainerCompleted is the default pod init container completed yaml.
	//go:embed pod-init-container-completed.yaml
	DefaultPodInitContainerCompleted string

	// DefaultPodInitContainerFailed is the default pod init container failed yaml.
	//go:embed pod-init-container-failed.yaml
	DefaultPodInitContainerFailed string

	// DefaultPodInitContainerRestart is the default pod init container restart yaml.
	//go:embed pod-init-container-restart.yaml
	DefaultPodInitContainerRestart string

	// DefaultPodReady is the default pod ready yaml.
	//go:embed pod-ready.yaml
	DefaultPodReady string

	// DefaultPodContainerFailed is the default pod container failed yaml.
	//go:embed pod-container-failed.yaml
	DefaultPodContainerFailed string

	// DefaultPodContainerRestart is the default pod container restart yaml.
	//go:embed pod-container-restart.yaml
	DefaultPodContainerRestart string

	// DefaultPodComplete is the default pod complete yaml.
	//go:embed pod-complete.yaml
	DefaultPodComplete string

	// DefaultPodDelete is the default pod delete yaml.
	//go:embed pod-delete.yaml
	DefaultPodDelete string
)
