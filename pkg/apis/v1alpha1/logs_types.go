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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LogsKind is the kind of the Logs.
	LogsKind = "Logs"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Logs provides port forward configuration for a single pod.
type Logs struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for port forward.
	Spec LogsSpec `json:"spec"`
}

// LogsSpec holds spec for port forward.
type LogsSpec struct {
	// Logs is a list of logs to configure.
	Logs []Log `json:"logs"`
}

// Log holds information how to forward logs.
type Log struct {
	// Containers is list of container names.
	Containers []string `json:"containers"`
	// LogsFile is the file from which the log forward starts
	LogsFile string `json:"logsFile"`
	// Follow up if true
	Follow bool `json:"follow"`
}
