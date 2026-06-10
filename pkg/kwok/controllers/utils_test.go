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
	"syscall"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func Test_ipPool_new(t *testing.T) {
	testCIDR := "172.30.40.0/24"
	netCIDR, _ := utilsnet.ParseCIDR(testCIDR)
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
		{
			name: "test ipPool get new ipv6 ip",
			fields: fields{
				cidr: func() *net.IPNet {
					cidr, _ := utilsnet.ParseCIDR("2001:db8::/64")
					return cidr
				}(),
				index: 0,
			},
			want: "2001:db8::1",
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
