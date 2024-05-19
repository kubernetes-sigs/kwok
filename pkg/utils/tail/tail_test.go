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

package tail

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestFindTailLineStartIndex(t *testing.T) {
	builder := strings.Builder{}
	for i := 0; i < 10; i++ {
		if i == 9 {
			builder.WriteString(fmt.Sprintf("%d", i))
		} else {
			builder.WriteString(fmt.Sprintf("%d\n", i))
		}
	}
	strLastLineNoEol := builder.String()
	strLastLineWithEol := strLastLineNoEol + "\n"
	fNoEol := strings.NewReader(strLastLineNoEol)
	fWithEol := strings.NewReader(strLastLineWithEol)

	type args struct {
		f io.ReadSeeker
		n int64
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "last line without eol, negative input",
			args: args{
				f: fNoEol,
				n: -1,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "last line without eol, last line",
			args: args{
				f: fNoEol,
				n: 1,
			},
			want:    16,
			wantErr: false,
		},
		{
			name: "last line without eol, all",
			args: args{
				f: fNoEol,
				n: 9,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "last line without eol, get first line",
			args: args{
				f: fNoEol,
				n: 8,
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "last line without eol, over 9 lines",
			args: args{
				f: fNoEol,
				n: 10,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "last line with eol, last line",
			args: args{
				f: fWithEol,
				n: 1,
			},
			want:    18,
			wantErr: false,
		},
		{
			name: "last line with eol, all",
			args: args{
				f: fWithEol,
				n: 10,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "last line with eol, get first line",
			args: args{
				f: fWithEol,
				n: 9,
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "last line with eol, over 10 lines",
			args: args{
				f: fWithEol,
				n: 11,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "last line with eol, negative input",
			args: args{
				f: fWithEol,
				n: -1,
			},
			want:    0,
			wantErr: false,
		},
		// Edge cases
		{
			name: "empty file",
			args: args{
				f: strings.NewReader(""),
				n: 1,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "single line file without eol",
			args: args{
				f: strings.NewReader("single line"),
				n: 1,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "file with multiple lines within one block",
			args: args{
				f: strings.NewReader("line1\nline2\nline3\n"),
				n: 2,
			},
			want:    6,
			wantErr: false,
		},
		{
			name: "error on seek",
			args: args{
				f: &faultyReader{data: []byte("line1\nline2\nline3\n"), seekErr: true},
				n: 1,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "error on read",
			args: args{
				f: &faultyReader{data: []byte("line1\nline2\nline3\n"), readErr: true},
				n: 1,
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindTailLineStartIndex(tt.args.f, tt.args.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindTailLineStartIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FindTailLineStartIndex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// faultyReader simulates errors for testing error handling in FindTailLineStartIndex.
type faultyReader struct {
	data    []byte
	seekErr bool
	readErr bool
	pos     int64
}

func (f *faultyReader) Seek(offset int64, whence int) (int64, error) {
	if f.seekErr {
		return 0, fmt.Errorf("seek error")
	}
	switch whence {
	case io.SeekStart:
		f.pos = offset
	case io.SeekCurrent:
		f.pos += offset
	case io.SeekEnd:
		f.pos = int64(len(f.data)) + offset
	}
	return f.pos, nil
}

func (f *faultyReader) Read(p []byte) (n int, err error) {
	if f.readErr {
		return 0, fmt.Errorf("read error")
	}
	copy(p, f.data[f.pos:])
	n = len(p)
	f.pos += int64(n)
	return n, nil
}
