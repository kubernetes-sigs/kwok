/*
Copyright The Kubernetes Authors.

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

package cluster

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestMutationComponentPatches(t *testing.T) {
	flags := &flagpole{
		ExtraArgs: []string{
			"kube-apiserver=feature-gates=A=true",
			"kube-apiserver=audit-log-maxage=7",
			"kube-scheduler=profiling=false",
			"kube-scheduler=v=2",
		},
		KwokctlConfiguration: &internalversion.KwokctlConfiguration{},
	}
	want := []internalversion.ComponentPatches{
		{
			Name: "kube-apiserver",
			ExtraArgs: []internalversion.ExtraArgs{
				{Key: "feature-gates", Value: new("A=true")},
				{Key: "audit-log-maxage", Value: new("7")},
			},
		},
		{
			Name: "kube-scheduler",
			ExtraArgs: []internalversion.ExtraArgs{
				{Key: "profiling", Value: new("false")},
				{Key: "v", Value: new("2")},
			},
		},
	}

	mutationComponentPatches(flags)

	if diff := cmp.Diff(want, flags.ComponentsPatches); diff != "" {
		t.Errorf("mutationComponentPatches() mismatch (-want +got):\n%s", diff)
	}
}
