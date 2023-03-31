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

package snapshot

import (
	"context"
)

// Runtime is the runtime of cluster
// Sync with runtime.Runtime interface
type Runtime interface {
	KubectlInCluster(ctx context.Context, args ...string) error
}

// Resources is the resources of cluster want to save or restore
var Resources = []string{
	"configmap",
	"endpoints",
	"namespace",
	"node",
	"persistentvolumeclaim",
	"persistentvolume",
	"pod",
	"secret",
	"serviceaccount",
	"service",
	"daemonset.apps",
	"deployment.apps",
	"replicaset.apps",
	"statefulset.apps",
	"cronjob.batch",
	"job.batch",
}
