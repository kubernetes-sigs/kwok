/*
Copyright 2024 The Kubernetes Authors.

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

package recording

import (
	"testing"
)

func Test_SpeedUpDown(t *testing.T) {
	for i := Speed(0); i <= 1000; {
		next := i.Up()
		if n := next.Down(); n != i {
			t.Errorf("%v up-> %v down-> %v", i, next, n)
		}
		i = next
	}
}

func Test_digitCount(t *testing.T) {
	tests := []struct {
		name string
		i    int64
		want int64
	}{

		{
			name: "0",
			i:    0,
			want: 0,
		},
		{
			name: "1",
			i:    1,
			want: 1,
		},
		{
			name: "10",
			i:    10,
			want: 2,
		},
		{
			name: "100",
			i:    100,
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := digitCount(tt.i); got != tt.want {
				t.Errorf("digitCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
