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

package runtime

import (
	"context"
	"reflect"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
)

func TestApplyComponentArgsOverride(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		patch    internalversion.ExtraArgs
		wantArgs []string
	}{
		{
			name: "Override the value",
			args: []string{
				"--foo=1",
				"--bar=2",
			},
			patch: internalversion.ExtraArgs{
				Key:      "foo",
				Value:    new("10"),
				Override: true,
			},
			wantArgs: []string{
				"--foo=10",
				"--bar=2",
			},
		},
		{
			name: "Unmatched flag",
			args: []string{
				"--foo=1",
			},
			patch: internalversion.ExtraArgs{
				Key:      "bar",
				Value:    new("2"),
				Override: true,
			},
			wantArgs: []string{
				"--foo=1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyComponentArgsOverride(context.TODO(), tt.args, internalversion.ExtraArgs{
				Key:      tt.patch.Key,
				Value:    tt.patch.Value,
				Override: tt.patch.Override,
			})

			if !reflect.DeepEqual(tt.wantArgs, tt.args) {
				t.Errorf("Exist is not expact! key:%s want args:%s, got args:%s", tt.patch.Key, tt.wantArgs, tt.args)
			}
		})
	}
}

func TestApplyComponentPatch(t *testing.T) {
	tests := []struct {
		name      string
		component internalversion.Component
		patch     internalversion.ComponentPatches
		wantArgs  []string
	}{
		{
			name: "Override the value",
			component: internalversion.Component{
				Name: "test",
				Args: []string{"--etcd-servers=http://localhost:2379", "--etcd-prefix=/registry"},
			},
			patch: internalversion.ComponentPatches{
				Name: "test",
				ExtraArgs: []internalversion.ExtraArgs{
					{
						Key:      "etcd-servers",
						Value:    new("http://127.0.0.1:2379"),
						Override: true,
					},
				},
			},
			wantArgs: []string{"--etcd-servers=http://127.0.0.1:2379", "--etcd-prefix=/registry"},
		},
		{
			name: "Do not override the value",
			component: internalversion.Component{
				Name: "test",
				Args: []string{"--etcd-servers=http://localhost:2379", "--etcd-prefix=/registry"},
			},
			patch: internalversion.ComponentPatches{
				Name: "test",
				ExtraArgs: []internalversion.ExtraArgs{
					{
						Key:      "etcd-servers",
						Value:    new("http://127.0.0.1:2379"),
						Override: false,
					},
				},
			},
			wantArgs: []string{"--etcd-servers=http://localhost:2379", "--etcd-prefix=/registry", "--etcd-servers=http://127.0.0.1:2379"},
		},
		{
			name: "Append bare flag when value is nil",
			component: internalversion.Component{
				Name: "test",
				Args: []string{"--config.file=/etc/prometheus/prometheus.yaml"},
			},
			patch: internalversion.ComponentPatches{
				Name: "test",
				ExtraArgs: []internalversion.ExtraArgs{
					{
						Key:      "web.enable-lifecycle",
						Value:    nil,
						Override: false,
					},
				},
			},
			wantArgs: []string{"--config.file=/etc/prometheus/prometheus.yaml", "--web.enable-lifecycle"},
		},
		{
			name: "Append flag with explicit empty value",
			component: internalversion.Component{
				Name: "test",
				Args: []string{"--server=https://127.0.0.1:6443", "--namespace=default"},
			},
			patch: internalversion.ComponentPatches{
				Name: "test",
				ExtraArgs: []internalversion.ExtraArgs{
					{
						Key:      "cluster",
						Value:    new(""),
						Override: false,
					},
				},
			},
			wantArgs: []string{"--server=https://127.0.0.1:6443", "--namespace=default", "--cluster="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyComponentPatch(context.TODO(), &tt.component, tt.patch)
			if !reflect.DeepEqual(tt.wantArgs, tt.component.Args) {
				t.Errorf("Exist is not expact! key:%s want args:%s, got args:%s", tt.patch.ExtraArgs[0].Key, tt.wantArgs, tt.component.Args)
			}
		})
	}
}

func TestApplyComponentPatchesReadinessProbe(t *testing.T) {
	newComponent := func() internalversion.Component {
		return internalversion.Component{
			Name: consts.ComponentKubeApiserver,
			Args: []string{"--bind-address=0.0.0.0", "--secure-port=6443"},
			ReadinessProbe: &internalversion.ComponentProbe{
				HTTPGet: &internalversion.ComponentHTTPGetAction{
					Path:   "/readyz",
					Port:   6443,
					Scheme: internalversion.URISchemeHTTPS,
				},
			},
		}
	}
	value := func(s string) *string { return &s }

	tests := []struct {
		name     string
		patches  []internalversion.ComponentPatches
		wantPort uint32
		// wantConflict is true if the patched component must stay not ready.
		wantConflict bool
	}{
		{
			name:     "no patches",
			wantPort: 6443,
		},
		{
			name: "patches of another component",
			patches: []internalversion.ComponentPatches{{
				Name:      consts.ComponentKubeScheduler,
				ExtraArgs: []internalversion.ExtraArgs{{Key: "secure-port", Value: value("0")}},
			}},
			wantPort: 6443,
		},
		{
			// An override replaces the argument in place, which must still be noticed.
			name: "port overridden",
			patches: []internalversion.ComponentPatches{{
				Name:      consts.ComponentKubeApiserver,
				ExtraArgs: []internalversion.ExtraArgs{{Key: "secure-port", Value: value("7443"), Override: true}},
			}},
			wantPort: 7443,
		},
		{
			name: "port appended",
			patches: []internalversion.ComponentPatches{{
				Name:      consts.ComponentKubeApiserver,
				ExtraArgs: []internalversion.ExtraArgs{{Key: "secure-port", Value: value("7443")}},
			}},
			wantPort: 7443,
		},
		{
			name: "anonymous authentication disabled",
			patches: []internalversion.ComponentPatches{{
				Name:      consts.ComponentKubeApiserver,
				ExtraArgs: []internalversion.ExtraArgs{{Key: "anonymous-auth", Value: value("false")}},
			}},
			wantPort:     6443,
			wantConflict: true,
		},
		{
			name: "anonymous authentication overridden",
			patches: []internalversion.ComponentPatches{{
				Name:      consts.ComponentKubeApiserver,
				ExtraArgs: []internalversion.ExtraArgs{{Key: "anonymous-auth", Value: value("false"), Override: true}},
			}},
			// The override has no argument to replace, so the probe keeps working.
			wantPort: 6443,
		},
		{
			name: "secure port overridden to zero",
			patches: []internalversion.ComponentPatches{{
				Name:      consts.ComponentKubeApiserver,
				ExtraArgs: []internalversion.ExtraArgs{{Key: "secure-port", Value: value("0"), Override: true}},
			}},
			wantPort:     6443,
			wantConflict: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := newComponent()
			ApplyComponentPatches(context.Background(), &component, tt.patches)
			// The probe is never removed, so that the component is never reported ready unprobed.
			if component.ReadinessProbe == nil || component.ReadinessProbe.HTTPGet.Port != tt.wantPort {
				t.Fatalf("ReadinessProbe = %#v, want port %d", component.ReadinessProbe, tt.wantPort)
			}
			err := components.ReadinessProbeConflict(component)
			if (err != nil) != tt.wantConflict {
				t.Fatalf("ReadinessProbeConflict() = %v, want a conflict %v", err, tt.wantConflict)
			}
		})
	}
}
