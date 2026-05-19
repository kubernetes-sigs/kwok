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

package components

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiserverv1alpha1 "k8s.io/apiserver/pkg/apis/apiserver/v1alpha1"
	tracingapi "k8s.io/component-base/tracing/api/v1"

	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// BuildKubeApiserverTracing builds a apiserverTracingConfig file from the given parameters.
func BuildKubeApiserverTracing(conf BuildKubeApiserverTracingConfig) (string, error) {
	c := &apiserverv1alpha1.TracingConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TracingConfiguration",
			APIVersion: apiserverv1alpha1.ConfigSchemeGroupVersion.String(),
		},
		TracingConfiguration: tracingapi.TracingConfiguration{
			Endpoint:               &conf.Endpoint,
			SamplingRatePerMillion: new(int32(1000000)),
		},
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// BuildKubeApiserverTracingConfig is the configuration for BuildKubeApiserverTracingConfig.
type BuildKubeApiserverTracingConfig struct {
	Endpoint string
}
