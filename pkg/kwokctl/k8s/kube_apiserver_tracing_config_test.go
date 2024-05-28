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

package k8s

import (
	"reflect"
	"testing"
)

func TestBuildKubeApiserverTracingConfig(t *testing.T) {
	tests := []struct {
		name          string
		conf          BuildKubeApiserverTracingConfigParam
		expected      string
		expectedError bool
	}{
		{
			name: "Valid endpoint",
			conf: BuildKubeApiserverTracingConfigParam{
				Endpoint: "http://example.com/tracing",
			},
			expected: `apiVersion: apiserver.config.k8s.io/v1alpha1
kind: TracingConfiguration
endpoint: http://example.com/tracing
samplingRatePerMillion: 1000000`,
			expectedError: false,
		},
		{
			name: "Empty endpoint",
			conf: BuildKubeApiserverTracingConfigParam{
				Endpoint: "",
			},
			expected: `apiVersion: apiserver.config.k8s.io/v1alpha1
kind: TracingConfiguration
endpoint: 
samplingRatePerMillion: 1000000`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildKubeApiserverTracingConfig(tt.conf)

			if tt.expectedError && err == nil {
				t.Errorf("expected an error, but got nil")
			} else if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if reflect.DeepEqual(result, tt.expected) {
				t.Errorf("unexpected output:\nexpected:\n%s\nactual:\n%s", tt.expected, result)
			}
		})
	}
}
