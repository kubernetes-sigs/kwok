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
	"testing"
)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AddCIDR(tt.args.cidr, tt.args.index)
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
