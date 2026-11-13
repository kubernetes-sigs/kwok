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
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	config := filepath.Join(t.TempDir(), "config.yaml")
	data := []InternalObject{
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

func TestLoadKwokConfigurationAppliesDefaults(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.yaml")
	data := `apiVersion: kwok.io/v1alpha1
kind: KwokConfiguration
options:
  nodeLeaseParallelism: 8
`
	if err := os.WriteFile(config, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	objs, err := Load(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 1 {
		t.Fatalf("got %d objects, want 1", len(objs))
	}
	conf, ok := objs[0].(*internalversion.KwokConfiguration)
	if !ok {
		t.Fatalf("unexpected type %T", objs[0])
	}

	// A field the document does not write keeps its default.
	if conf.Options.CIDR != "10.0.0.0/24" {
		t.Errorf("CIDR = %q, want the default 10.0.0.0/24", conf.Options.CIDR)
	}
	if conf.Options.PodPlayStageParallelism != 4 {
		t.Errorf("PodPlayStageParallelism = %d, want the default 4", conf.Options.PodPlayStageParallelism)
	}
	// A field the document does write keeps the document's value.
	if conf.Options.NodeLeaseParallelism != 8 {
		t.Errorf("NodeLeaseParallelism = %d, want 8", conf.Options.NodeLeaseParallelism)
	}
}

func TestLoadKwokConfigurationDefaultsAfterMerge(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.yaml")
	// The second document does not write cidr, so the first one's value must
	// survive the merge instead of being replaced by the default.
	data := `apiVersion: kwok.io/v1alpha1
kind: KwokConfiguration
options:
  cidr: 10.1.0.0/24
---
apiVersion: kwok.io/v1alpha1
kind: KwokConfiguration
options:
  nodeLeaseParallelism: 8
`
	if err := os.WriteFile(config, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	objs, err := Load(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 1 {
		t.Fatalf("got %d objects, want 1", len(objs))
	}
	conf, ok := objs[0].(*internalversion.KwokConfiguration)
	if !ok {
		t.Fatalf("unexpected type %T", objs[0])
	}

	if conf.Options.CIDR != "10.1.0.0/24" {
		t.Errorf("CIDR = %q, want the value from the first document", conf.Options.CIDR)
	}
	if conf.Options.NodeLeaseParallelism != 8 {
		t.Errorf("NodeLeaseParallelism = %d, want 8", conf.Options.NodeLeaseParallelism)
	}
	if conf.Options.PodPlayStageParallelism != 4 {
		t.Errorf("PodPlayStageParallelism = %d, want the default 4", conf.Options.PodPlayStageParallelism)
	}
}

func Test_loadRaw(t *testing.T) {
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
			got, err := loadRaw(bytes.NewBuffer(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("loadRaw() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadRaw() got = %v, want %v", got, tt.want)
			}
		})
	}
}
