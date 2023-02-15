/*
Copyright 2023 The Kubernetes Authors.

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

package config

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestContext(t *testing.T) {
	ctx := context.Background()
	ctx = setupContext(ctx, []metav1.Object{
		&metav1.ObjectMeta{
			Name: "first",
		},
	})

	addToContext(ctx, &metav1.ObjectMeta{
		Name: "second",
	})

	objs := getFromContext(ctx)
	if len(objs) != 2 {
		t.Errorf("expected 2 objects, got %d", len(objs))
	}

	want := []metav1.Object{
		&metav1.ObjectMeta{
			Name: "first",
		},
		&metav1.ObjectMeta{
			Name: "second",
		},
	}

	if diff := cmp.Diff(want, objs); diff != "" {
		t.Errorf("unexpected objects (-want +got):\n%s", diff)
	}
}
