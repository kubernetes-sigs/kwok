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
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestApplyComponentPatch(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		override bool
	}{
		{
			name:     "Override the value",
			key:      "etcd-servers",
			value:    "http://192.168.66.2:3379",
			override: true,
		},
		{
			name:     "Do not override the value",
			key:      "etcd-prefix",
			value:    "/test",
			override: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newDefault()
			p := internalversion.ComponentPatches{
				Name: "test",
				ExtraArgs: []internalversion.ExtraArgs{
					{
						Key:      tt.key,
						Value:    tt.value,
						Override: tt.override,
					},
				},
			}
			applyComponentPatch(c, p)
			kk := ""
			vv := []string{}
			for _, a := range c.Args {
				k, v := getKeyValueFromArg(a)
				if k == tt.key {
					kk = k
					vv = append(vv, v)
				}
			}

			if kk == "" {
				t.Errorf("Not exist the patch extra args! key:%s value:%s", tt.key, tt.value)
			}

			if tt.override && (len(vv) != 1 || vv[0] != tt.value) {
				t.Errorf("Extra arg should override the old value! key:%s value:%s, now value:%s", tt.key, tt.value, vv)
			}

			if !tt.override {
				existExtraArg := false
				for _, v := range vv {
					if v == tt.value {
						existExtraArg = true
					}
				}
				if len(vv) != 2 || !existExtraArg {
					t.Errorf("Extra arg should exist the mult value when not override the extra args! key:%s value:%s, now value:%s", tt.key, tt.value, vv)
				}
			}
		})
	}
}

func newDefault() *internalversion.Component {
	return &internalversion.Component{
		Name: "test",
		Args: []string{"--etcd-servers=http://localhost:2379", "--etcd-prefix=/registry"},
	}
}
