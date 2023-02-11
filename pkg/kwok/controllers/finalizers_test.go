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

package controllers

import (
	"reflect"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func Test_finalizersAdd(t *testing.T) {
	type args struct {
		metaFinalizers []string
		finalizers     []internalversion.FinalizerItem
	}
	tests := []struct {
		name string
		args args
		want []jsonpathOperation
	}{
		{
			args: args{
				metaFinalizers: nil,
				finalizers: []internalversion.FinalizerItem{
					{
						Value: "a",
					},
				},
			},
			want: []jsonpathOperation{
				{Op: "add", Path: "/metadata/finalizers", Value: []string{"a"}},
			},
		},
		{
			args: args{
				metaFinalizers: []string{"a"},
				finalizers: []internalversion.FinalizerItem{
					{
						Value: "a",
					},
					{
						Value: "b",
					},
				},
			},
			want: []jsonpathOperation{
				{Op: "add", Path: "/metadata/finalizers/-", Value: "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := finalizersAdd(tt.args.metaFinalizers, tt.args.finalizers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("finalizersAdd() got = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_finalizersRemove(t *testing.T) {
	type args struct {
		metaFinalizers []string
		finalizers     []internalversion.FinalizerItem
	}
	tests := []struct {
		name string
		args args
		want []jsonpathOperation
	}{
		{
			args: args{
				metaFinalizers: nil,
				finalizers: []internalversion.FinalizerItem{
					{
						Value: "a",
					},
				},
			},
			want: nil,
		},
		{
			args: args{
				metaFinalizers: []string{"a", "b", "c"},
				finalizers: []internalversion.FinalizerItem{
					{
						Value: "a",
					},
					{
						Value: "c",
					},
				},
			},
			want: []jsonpathOperation{
				{Op: "remove", Path: "/metadata/finalizers/2", Value: nil},
				{Op: "remove", Path: "/metadata/finalizers/0", Value: nil},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := finalizersRemove(tt.args.metaFinalizers, tt.args.finalizers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("finalizersRemove() got = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_finalizersModify(t *testing.T) {
	type args struct {
		metaFinalizers []string
		finalizers     *internalversion.StageFinalizers
	}
	tests := []struct {
		name string
		args args
		want []jsonpathOperation
	}{
		{
			args: args{
				metaFinalizers: nil,
				finalizers: &internalversion.StageFinalizers{
					Empty: true,
				},
			},
			want: nil,
		},
		{
			args: args{
				metaFinalizers: []string{"a"},
				finalizers: &internalversion.StageFinalizers{
					Empty: true,
				},
			},
			want: []jsonpathOperation{
				{Op: "remove", Path: "/metadata/finalizers", Value: nil},
			},
		},
		{
			args: args{
				metaFinalizers: []string{"a"},
				finalizers: &internalversion.StageFinalizers{
					Empty: true,
					Add: []internalversion.FinalizerItem{
						{
							Value: "b",
						},
					},
				},
			},
			want: []jsonpathOperation{
				{Op: "remove", Path: "/metadata/finalizers", Value: nil},
				{Op: "add", Path: "/metadata/finalizers", Value: []string{"b"}},
			},
		},
		{
			args: args{
				metaFinalizers: nil,
				finalizers: &internalversion.StageFinalizers{
					Empty: true,
					Add: []internalversion.FinalizerItem{
						{
							Value: "b",
						},
					},
				},
			},
			want: []jsonpathOperation{
				{Op: "add", Path: "/metadata/finalizers", Value: []string{"b"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := finalizersModify(tt.args.metaFinalizers, tt.args.finalizers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("finalizersModify() = %v, want %v", got, tt.want)
			}
		})
	}
}
