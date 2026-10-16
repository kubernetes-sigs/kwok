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

package internalversion

import (
	"encoding/json"
	"reflect"
	"testing"

	configv1alpha1 "sigs.k8s.io/kwok/pkg/apis/config/v1alpha1"
)

func TestComponentReadinessProbeConversion(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want *ComponentProbe
	}{
		{
			name: "not configured",
			raw:  `{"name":"jobset"}`,
			want: nil,
		},
		{
			name: "http get",
			raw:  `{"name":"jobset","readinessProbe":{"httpGet":{"path":"/readyz","port":8081},"timeoutSeconds":5}}`,
			want: &ComponentProbe{
				HTTPGet: &ComponentHTTPGetAction{
					Path: "/readyz",
					Port: 8081,
				},
				TimeoutSeconds: 5,
			},
		},
		{
			name: "https get",
			raw:  `{"name":"kube-apiserver","readinessProbe":{"httpGet":{"path":"/readyz","port":6443,"scheme":"HTTPS"},"timeoutSeconds":5}}`,
			want: &ComponentProbe{
				HTTPGet: &ComponentHTTPGetAction{
					Path:   "/readyz",
					Port:   6443,
					Scheme: URISchemeHTTPS,
				},
				TimeoutSeconds: 5,
			},
		},
		{
			// Rejected when the probe runs, rather than when the configuration is loaded.
			name: "unsupported scheme",
			raw:  `{"name":"kube-apiserver","readinessProbe":{"httpGet":{"path":"/readyz","port":6443,"scheme":"TCP"}}}`,
			want: &ComponentProbe{
				HTTPGet: &ComponentHTTPGetAction{
					Path:   "/readyz",
					Port:   6443,
					Scheme: "TCP",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var in configv1alpha1.Component
			if err := json.Unmarshal([]byte(tt.raw), &in); err != nil {
				t.Fatal(err)
			}

			var internal Component
			if err := Convert_v1alpha1_Component_To_internalversion_Component(&in, &internal, nil); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(internal.ReadinessProbe, tt.want) {
				t.Fatalf("ReadinessProbe = %#v, want %#v", internal.ReadinessProbe, tt.want)
			}

			var out configv1alpha1.Component
			if err := Convert_internalversion_Component_To_v1alpha1_Component(&internal, &out, nil); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(out)
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != tt.raw {
				t.Fatalf("round trip = %s, want %s", raw, tt.raw)
			}
		})
	}
}

func TestComponentReadinessProbeDeepCopy(t *testing.T) {
	in := &Component{
		Name: "jobset",
		ReadinessProbe: &ComponentProbe{
			HTTPGet: &ComponentHTTPGetAction{
				Path: "/readyz",
				Port: 8081,
			},
			TimeoutSeconds: 5,
		},
	}
	out := in.DeepCopy()
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("DeepCopy() = %#v, want %#v", out, in)
	}

	out.ReadinessProbe.TimeoutSeconds = 1
	out.ReadinessProbe.HTTPGet.Port = 9090
	if in.ReadinessProbe.TimeoutSeconds != 5 || in.ReadinessProbe.HTTPGet.Port != 8081 {
		t.Fatal("DeepCopy() shares the readiness probe with the original")
	}

	v1 := &configv1alpha1.Component{
		ReadinessProbe: &configv1alpha1.ComponentProbe{
			HTTPGet: &configv1alpha1.ComponentHTTPGetAction{Port: 8081},
		},
	}
	v1Copy := v1.DeepCopy()
	v1Copy.ReadinessProbe.HTTPGet.Port = 9090
	if v1.ReadinessProbe.HTTPGet.Port != 8081 {
		t.Fatal("v1alpha1 DeepCopy() shares the readiness probe with the original")
	}
}
