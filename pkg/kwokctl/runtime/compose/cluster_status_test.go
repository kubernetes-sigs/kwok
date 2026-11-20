/*
Copyright 2026 The Kubernetes Authors.

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

func Test_checkAppleInspect(t *testing.T) {
	type args struct {
		raw []byte
	}
	tests := []struct {
		name        string
		args        args
		wantRunning bool
		wantExist   bool
		wantErr     bool
	}{
		// container CLI 1.6.x reports status as an object, 1.0.x as a plain string.
		{
			name: "status object running",
			args: args{
				[]byte(`[{"configuration":{"image":{"reference":"x"}},"id":"buildkit","status":{"networks":[{"hostname":"buildkit","ipv4Address":"192.168.65.3/24","ipv4Gateway":"192.168.65.1","ipv6Address":"fd97:984a:4e3e:dc54:f8c1:e2ff:fe5c:5885/64","macAddress":"fa:c1:e2:5c:58:85","network":"default","variant":"reserved"}],"startedDate":"2026-10-27T08:46:13Z","state":"running"}}]`),
			},
			wantRunning: true,
			wantExist:   true,
			wantErr:     false,
		},
		{
			name: "status object stopped never started",
			args: args{
				[]byte(`[{"configuration":{"image":{"reference":"x"}},"id":"buildkit","status":{"networks":[],"state":"stopped"}}]`),
			},
			wantRunning: false,
			wantExist:   true,
			wantErr:     false,
		},
		{
			name: "status object stopped after start",
			args: args{
				[]byte(`[{"configuration":{"image":{"reference":"x"}},"id":"buildkit","status":{"networks":[],"startedDate":"2026-10-27T08:46:13Z","state":"stopped"}}]`),
			},
			wantRunning: false,
			wantExist:   true,
			wantErr:     false,
		},
		{
			name: "status string running",
			args: args{
				[]byte(`[{"status":"running"}]`),
			},
			wantRunning: true,
			wantExist:   true,
			wantErr:     false,
		},
		{
			name: "status string stopped",
			args: args{
				[]byte(`[{"status":"stopped"}]`),
			},
			wantRunning: false,
			wantExist:   true,
			wantErr:     false,
		},
		{
			name: "empty array",
			args: args{
				[]byte(`[]`),
			},
			wantRunning: false,
			wantExist:   false,
			wantErr:     false,
		},
		{
			name: "not json",
			args: args{
				[]byte(`garbage`),
			},
			wantRunning: false,
			wantExist:   false,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRunning, gotExist, err := checkAppleInspect(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkAppleInspect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRunning != tt.wantRunning {
				t.Errorf("checkAppleInspect() gotRunning = %v, want %v", gotRunning, tt.wantRunning)
			}
			if gotExist != tt.wantExist {
				t.Errorf("checkAppleInspect() gotExist = %v, want %v", gotExist, tt.wantExist)
			}
		})
	}
}
