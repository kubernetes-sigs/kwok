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
				Value:    "10",
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
				Value:    "2",
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
						Value:    "http://127.0.0.1:2379",
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
						Value:    "http://127.0.0.1:2379",
						Override: false,
					},
				},
			},
			wantArgs: []string{"--etcd-servers=http://localhost:2379", "--etcd-prefix=/registry", "--etcd-servers=http://127.0.0.1:2379"},
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
