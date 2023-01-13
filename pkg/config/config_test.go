/*
Copyright 2022 The Kubernetes Authors.

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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	config := filepath.Join(t.TempDir(), "config.yaml")
	data := []metav1.Object{
		&internalversion.KwokConfiguration{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1alpha1.KwokConfigurationKind,
				APIVersion: v1alpha1.GroupVersion.String(),
			},
		},
		&internalversion.KwokctlConfiguration{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1alpha1.KwokctlConfigurationKind,
				APIVersion: v1alpha1.GroupVersion.String(),
			},
		},
		&internalversion.Stage{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1alpha1.StageKind,
				APIVersion: v1alpha1.GroupVersion.String(),
			},
		},
	}
	err := Save(ctx, config, data)
	if err != nil {
		t.Fatal(err)
	}

	want, err := Load(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	err = Save(ctx, config, want)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Load(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}
