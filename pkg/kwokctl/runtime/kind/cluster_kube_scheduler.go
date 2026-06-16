/*
Copyright 2026 The Kubernetes Authors.

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

package kind

import (
	"context"
	"slices"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
)

func (c *Cluster) addKubeScheduler(_ context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeScheduler) {
		return nil
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, internalversion.Component{
		Name: consts.ComponentKubeScheduler,
		Metric: &internalversion.ComponentMetric{
			Scheme:             "https",
			Host:               "127.0.0.1:10259",
			Path:               "/metrics",
			CertPath:           "/etc/kubernetes/pki/admin.crt",
			KeyPath:            "/etc/kubernetes/pki/admin.key",
			InsecureSkipVerify: true,
		},
	})
	return nil
}
