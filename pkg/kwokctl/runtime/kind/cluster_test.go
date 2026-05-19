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

import "testing"

func TestImageSaveArchiveArgs(t *testing.T) {
	t.Parallel()

	got := imageSaveArchiveArgs("registry.k8s.io/kwok:v0.0.0", "/tmp/image.tar")
	want := []string{"save", "--format", "oci-archive", "registry.k8s.io/kwok:v0.0.0", "-o", "/tmp/image.tar"}

	if len(got) != len(want) {
		t.Fatalf("unexpected args length: got %d want %d, got args: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected arg at index %d: got %q want %q (all args: %v)", i, got[i], want[i], got)
		}
	}
}
