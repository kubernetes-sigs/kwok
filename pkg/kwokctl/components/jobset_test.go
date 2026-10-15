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
	"reflect"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
)

func TestBuildJobSetComponentReadinessProbe(t *testing.T) {
	component, err := BuildJobSetComponent(BuildJobSetComponentConfig{
		Runtime: consts.RuntimeTypeDocker,
		Image:   "registry.k8s.io/jobset/jobset:v0.12.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := &internalversion.ComponentProbe{
		HTTPGet: &internalversion.ComponentHTTPGetAction{
			Path: "/readyz",
			Port: 8081,
		},
		TimeoutSeconds: 5,
	}
	if !reflect.DeepEqual(component.ReadinessProbe, want) {
		t.Errorf("ReadinessProbe = %#v, want %#v", component.ReadinessProbe, want)
	}
}
