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

package net

import (
	"net"
	"reflect"
	"testing"
)

func TestAddIP(t *testing.T) {
	tests := []struct {
		name  string
		ip    net.IP
		index int
		want  net.IP
	}{
		{
			name:  "ipv4 carry",
			ip:    net.ParseIP("10.0.0.255"),
			index: 1,
			want:  net.ParseIP("10.0.1.0"),
		},
		{
			name:  "ipv6 carry",
			ip:    net.ParseIP("2001:db8::ffff:ffff:ffff:ffff"),
			index: 1,
			want:  net.ParseIP("2001:db8:0:1::"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddIP(tt.ip, tt.index)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddIP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddCIDR(t *testing.T) {
	type args struct {
		cidr  string
		index int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "24 bit",
			args: args{
				cidr:  "10.100.0.1/24",
				index: 0,
			},
			want: "10.100.0.1/24",
		},
		{
			name: "24 bit +1",
			args: args{
				cidr:  "10.100.0.1/24",
				index: 1,
			},
			want: "10.100.1.1/24",
		},
		{
			name: "25 bit",
			args: args{
				cidr:  "10.100.0.1/25",
				index: 0,
			},
			want: "10.100.0.1/25",
		},
		{
			name: "25 bit +1",
			args: args{
				cidr:  "10.100.0.1/25",
				index: 1,
			},
			want: "10.100.0.129/25",
		},
		{
			name: "25 bit +2",
			args: args{
				cidr:  "10.100.0.1/25",
				index: 2,
			},
			want: "10.100.1.1/25",
		},
		{
			name: "ipv6 /64 +1",
			args: args{
				cidr:  "2001:db8::1/64",
				index: 1,
			},
			want: "2001:db8:0:1::1/64",
		},
		{
			name: "ipv6 /64 +2",
			args: args{
				cidr:  "2001:db8::1/64",
				index: 2,
			},
			want: "2001:db8:0:2::1/64",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AddCIDRStr(tt.args.cidr, tt.args.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AddCIDR() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
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
		{
			name: "test parseCIDR ipv6 success",
			args: args{
				s: "2001:db8::1/64",
			},
			want: &net.IPNet{
				IP:   net.ParseIP("2001:db8::1"),
				Mask: net.CIDRMask(64, 128),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCIDR(tt.args.s)
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
