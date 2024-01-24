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
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"syscall"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Test_parseCIDR(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    *net.IPNet
		wantErr bool
	}{
		{
			name: "test parseCIDR error",
			args: args{
				s: "10.12.12.1.12",
			},
			wantErr: true,
		},
		{
			name: "test parseCIDR success",
			args: args{
				s: "172.30.40.1/24",
			},
			want: &net.IPNet{
				IP:   net.ParseIP("172.30.40.1"),
				Mask: net.IPMask{255, 255, 255, 0},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCIDR(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCIDR() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addIP(t *testing.T) {
	type args struct {
		ip  net.IP
		add uint64
	}
	tests := []struct {
		name string
		args args
		want net.IP
	}{
		{
			name: "test ip length less than 8",
			args: args{
				ip:  net.IP{1, 1, 1, 1},
				add: 1,
			},
			want: net.IP{1, 1, 1, 1},
		},
		{
			name: "test add ip success",
			args: args{
				ip:  net.ParseIP("172.30.40.1"),
				add: 1,
			},
			want: net.ParseIP("172.30.40.2"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addIP(tt.args.ip, tt.args.add); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ipPool_new(t *testing.T) {
	testCIDR := "172.30.40.1/24"
	netCIDR, _ := parseCIDR(testCIDR)
	type fields struct {
		cidr  *net.IPNet
		index uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "test ipPool get new ip",
			fields: fields{
				cidr:  netCIDR,
				index: 0,
			},
			want: "172.30.40.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newIPPool(tt.fields.cidr)
			if got := pool.new(); got != tt.want {
				t.Errorf("new() = %v, want %v", got, tt.want)
			}
		})
	}
}

func genResource(resource string) schema.GroupResource {
	return schema.GroupResource{Group: "", Resource: resource}
}

func Test_shouldRetry(t *testing.T) {
	var testCases = []struct {
		name     string
		input    error
		expected bool
	}{
		{
			name:     "connection refused error",
			input:    syscall.ECONNREFUSED,
			expected: true,
		},
		{
			name:     "too many request error",
			input:    apierrors.NewTooManyRequests("foo", 1),
			expected: true,
		},
		{
			name:     "resource not found error",
			input:    apierrors.NewNotFound(genResource("foo"), "bar"),
			expected: false,
		},
		{
			name:     "resource conflict error",
			input:    apierrors.NewConflict(genResource("foo"), "bar", errors.New("message")),
			expected: false,
		},
		{
			name:     "marshal error",
			input:    &json.UnsupportedTypeError{},
			expected: false,
		},
		{
			name:     "unknown error",
			input:    errors.New("an weird error"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if shouldRetry(tc.input) != tc.expected {
				t.Errorf("expected: %t", tc.expected)
			}
		})
	}
}
