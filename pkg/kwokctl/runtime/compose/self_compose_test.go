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

package compose

import (
	"testing"
)

func Test_checkInspect(t *testing.T) {
	type args struct {
		raw []byte
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			args: args{
				[]byte(`{"State":{"Running":true}}`),
			},
			want:    true,
			wantErr: false,
		},
		{
			args: args{
				[]byte(`{"State":{"Running":false}}`),
			},
			want:    false,
			wantErr: false,
		},
		{
			args: args{
				[]byte(`[{"State":{"Running":true}}]`),
			},
			want:    true,
			wantErr: false,
		},
		{
			args: args{
				[]byte(`[{"State":{"Running":false}}]`),
			},
			want:    false,
			wantErr: false,
		},
		{
			args: args{
				[]byte(`{}`),
			},
			want:    false,
			wantErr: false,
		},
		{
			args: args{
				[]byte(`[{}]`),
			},
			want:    false,
			wantErr: false,
		},
		{
			args: args{
				[]byte(`[]`),
			},
			want:    false,
			wantErr: false,
		},
		{
			args: args{
				[]byte(` []`),
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkInspect(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInspect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkInspect() got = %v, want %v", got, tt.want)
			}
		})
	}
}
