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
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestFilterDuplicatedExtraArgs(t *testing.T) {
	got := filterDuplicatedExtraArgs(context.Background(),
		[]internalversion.ExtraArgs{
			{Key: "zeta", Value: "old-zeta"},
			{Key: "alpha", Value: "old-alpha"},
		},
		[]internalversion.ExtraArgs{
			{Key: "beta", Value: "new-beta"},
			{Key: "alpha", Value: "new-alpha"},
		},
	)

	want := []internalversion.ExtraArgs{
		{Key: "alpha", Value: "new-alpha"},
		{Key: "beta", Value: "new-beta"},
		{Key: "zeta", Value: "old-zeta"},
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected length: got=%d want=%d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected arg at %d: got=%+v want=%+v", i, got[i], want[i])
		}
	}
}
