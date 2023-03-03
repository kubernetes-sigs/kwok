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
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	config := filepath.Join(t.TempDir(), "config.yaml")
	data := []metav1.Object{
		&internalversion.KwokConfiguration{},
		&internalversion.KwokctlConfiguration{},
		&internalversion.Stage{},
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

func Test_loadRawConfig(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    []json.RawMessage
		wantErr bool
	}{
		{
			name: "single document",
			data: []byte(`apiVersion: kwok.io/v1alpha1
kind: KwokConfiguration
`),
			want: []json.RawMessage{
				[]byte(`{"apiVersion":"kwok.io/v1alpha1","kind":"KwokConfiguration"}`),
			},
		},
		{
			name: "multiple documents",
			data: []byte(`apiVersion: kwok.io/v1alpha1
kind: KwokConfiguration
---
apiVersion: kwok.io/v1alpha1
kind: KwokctlConfiguration
`),
			want: []json.RawMessage{
				[]byte(`{"apiVersion":"kwok.io/v1alpha1","kind":"KwokConfiguration"}`),
				[]byte(`{"apiVersion":"kwok.io/v1alpha1","kind":"KwokctlConfiguration"}`),
			},
		},
		{
			name: "empty document at start",
			data: []byte(`# Some comment
---
apiVersion: kwok.io/v1alpha1
kind: KwokConfiguration
`),
			want: []json.RawMessage{
				[]byte(`{"apiVersion":"kwok.io/v1alpha1","kind":"KwokConfiguration"}`),
			},
		},
		{
			name: "empty document at end",
			data: []byte(`apiVersion: kwok.io/v1alpha1
kind: KwokConfiguration
---
# Some comment
`),
			want: []json.RawMessage{
				[]byte(`{"apiVersion":"kwok.io/v1alpha1","kind":"KwokConfiguration"}`),
			},
		},
		{
			name: "empty document in middle",
			data: []byte(`apiVersion: kwok.io/v1alpha1
kind: KwokConfiguration
---
# Some comment
---
apiVersion: kwok.io/v1alpha1
kind: KwokctlConfiguration
`),
			want: []json.RawMessage{
				[]byte(`{"apiVersion":"kwok.io/v1alpha1","kind":"KwokConfiguration"}`),
				[]byte(`{"apiVersion":"kwok.io/v1alpha1","kind":"KwokctlConfiguration"}`),
			},
		},
		{
			name:    "invalid document",
			data:    []byte(`{"invalid"}"`),
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty",
			data: []byte(``),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "config.yaml")
			err := os.WriteFile(p, tt.data, 0o640)
			if err != nil {
				t.Fatal(err)
			}

			got, err := loadRawConfig(p)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadRawConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadRawConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
