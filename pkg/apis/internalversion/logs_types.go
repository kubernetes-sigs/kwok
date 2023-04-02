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

// Logs provides log configuration for a single pod.
type Logs struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Spec holds spec for logs.
	Spec LogsSpec
}

// LogsSpec holds spec for logs.
type LogsSpec struct {
	// Logs is a list of logs to configure.
	Logs []Log
}

// Log holds information how to forward logs.
type Log struct {
	// Containers is list of container names.
	Containers []string
	// LogsFile is the file from which the log forward starts
	LogsFile string
	// Follow up if true
	Follow bool
}
