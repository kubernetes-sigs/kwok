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

package etcd

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
		err      error
	}{
		{
			name:     "Detect Protobuf Media Type",
			input:    append(protoEncodingPrefix, []byte(`{"apiVersion":"v1","kind":"Pod"}`)...),
			expected: StorageBinaryMediaType,
			err:      nil,
		},
		{
			name:     "Detect JSON Media Type",
			input:    []byte(`{"apiVersion":"v1","kind":"Pod"}`),
			expected: JSONMediaType,
			err:      nil,
		},
		{
			name:     "Detect Unknown Media Type",
			input:    []byte(`apiVersion: v1\nkind: Pod\n`),
			expected: "",
			err:      fmt.Errorf("unable to detect media type"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectMediaType(tt.input)
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("DetectMediaType() error = %v, wantErr %v", err, tt.err)
				return
			}
			if result != tt.expected {
				t.Errorf("DetectMediaType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPrefixFromGVR(t *testing.T) {
	type args struct {
		gvr schema.GroupVersionResource
	}
	tests := []struct {
		name       string
		args       args
		wantPrefix string
		wantErr    bool
	}{
		{
			name: "pod",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
			},
			wantPrefix: "pods",
			wantErr:    false,
		},
		{
			name: "deployment",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "apps",
					Version:  "v1",
					Resource: "deployments",
				},
			},
			wantPrefix: "deployments",
			wantErr:    false,
		},
		{
			name: "service",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "services",
				},
			},
			wantPrefix: "services/specs",
			wantErr:    false,
		},
		{
			name: "ingress",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "networking.k8s.io",
					Version:  "v1",
					Resource: "ingresses",
				},
			},
			wantPrefix: "ingress",
		},
		{
			name: "apiextensions.k8s.io",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "apiextensions.k8s.io",
					Version:  "v1",
					Resource: "customresourcedefinitions",
				},
			},
			wantPrefix: "apiextensions.k8s.io/customresourcedefinitions",
		},
		{
			name: "x-k8s.io",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "kwok.x-k8s.io",
					Version:  "v1",
					Resource: "foo",
				},
			},
			wantPrefix: "kwok.x-k8s.io/foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPrefix, err := PrefixFromGVR(tt.args.gvr)
			if (err != nil) != tt.wantErr {
				t.Errorf("PrefixFromGVR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPrefix != tt.wantPrefix {
				t.Errorf("PrefixFromGVR() gotPrefix = %v, want %v", gotPrefix, tt.wantPrefix)
			}
		})
	}
}

func TestMediaTypeFromGVR(t *testing.T) {
	type args struct {
		gvr schema.GroupVersionResource
	}
	tests := []struct {
		name          string
		args          args
		wantMediaType string
		wantErr       bool
	}{
		{
			name: "pod",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
			},
			wantMediaType: StorageBinaryMediaType,
		},
		{
			name: "deployment",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "apps",
					Version:  "v1",
					Resource: "deployments",
				},
			},
			wantMediaType: StorageBinaryMediaType,
		},
		{
			name: "service",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "services",
				},
			},
			wantMediaType: StorageBinaryMediaType,
		},
		{
			name: "ingress",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "networking.k8s.io",
					Version:  "v1",
					Resource: "ingresses",
				},
			},
			wantMediaType: StorageBinaryMediaType,
		},
		{
			name: "apiextensions.k8s.io",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "apiextensions.k8s.io",
					Version:  "v1",
					Resource: "customresourcedefinitions",
				},
			},
			wantMediaType: JSONMediaType,
		},
		{
			name: "x-k8s.io",
			args: args{
				gvr: schema.GroupVersionResource{
					Group:    "kwok.x-k8s.io",
					Version:  "v1",
					Resource: "foo",
				},
			},
			wantMediaType: JSONMediaType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMediaType, err := MediaTypeFromGVR(tt.args.gvr)
			if (err != nil) != tt.wantErr {
				t.Errorf("MediaTypeFromGVR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotMediaType != tt.wantMediaType {
				t.Errorf("MediaTypeFromGVR() gotMediaType = %v, want %v", gotMediaType, tt.wantMediaType)
			}
		})
	}
}
