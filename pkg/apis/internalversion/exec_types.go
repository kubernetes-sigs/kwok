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

package internalversion

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Exec provides exec configuration for a single pod.
type Exec struct {
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Spec holds spec for exec
	Spec ExecSpec
}

// ExecSpec holds spec for exec
type ExecSpec struct {
	// Execs is a list of execs to configure.
	Execs []ExecTarget
}

// ExecTarget holds information how to exec.
type ExecTarget struct {
	// Containers is a list of containers to exec.
	// if not set, all containers will be execed.
	Containers []string
	// Local holds information how to exec to a local target.
	Local *ExecTargetLocal
}

// ExecTargetLocal holds information how to exec to a local target.
type ExecTargetLocal struct {
	// WorkDir is the working directory to exec with.
	WorkDir string
	// Envs is a list of environment variables to exec with.
	Envs []EnvVar
	// SecurityContext is the user context to exec.
	SecurityContext *SecurityContext
}

// EnvVar represents an environment variable present in a Container.
type EnvVar struct {
	// Name of the environment variable.
	Name string
	// Value of the environment variable.
	Value string
}

// SecurityContext specifies the existing uid and gid to run exec command in container process.
type SecurityContext struct {
	// RunAsUser is the existing uid to run exec command in container process.
	RunAsUser *int64
	// RunAsGroup is the existing gid to run exec command in container process.
	RunAsGroup *int64
}
