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

package file

import (
	"context"
	"io/fs"
	"os"
	"testing"

	"sigs.k8s.io/kwok/pkg/utils/path"
)

func TestDownloadWithCacheAndExtract(t *testing.T) {
	ctx := context.Background()
	cacheDir := "/tmp/cache"
	src := "http://example.com/file.tar.gz"
	dest := "/tmp/dest/file"
	match := "file"
	mode := fs.FileMode(0644)
	quiet := false
	clean := false

	// Setup: Ensure the destination file exists
	if err := os.MkdirAll(path.Dir(dest), 0750); err != nil {
		t.Fatalf("Failed to close : %v", err)
	}

	f, _ := os.Create(dest)
	if err := f.Close(); err != nil {
		t.Fatalf("Failed to close : %v", err)
	}

	// Call the function
	err := DownloadWithCacheAndExtract(ctx, cacheDir, src, dest, match, mode, quiet, clean)

	// Assert: No error should be returned
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Cleanup
	if err := os.Remove(dest); err != nil {
		t.Fatalf("Failed to close : %v", err)
	}
	if err := os.Remove(path.Dir(dest)); err != nil {
		t.Fatalf("Failed to close : %v", err)
	}
}

func TestDownloadWithCache(t *testing.T) {
	type args struct {
		ctx      context.Context
		cacheDir string
		src      string
		dest     string
		mode     fs.FileMode
		quiet    bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name: "TestingDownloadWithCache",
		args: args{
			ctx:      context.Background(),
			cacheDir: "/tmp/cache",
			src:      "http://example.com/file",
			dest:     "/tmp/dest/file",
			mode:     fs.FileMode(0644),
			quiet:    false,
		},
		wantErr: true,
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DownloadWithCache(tt.args.ctx, tt.args.cacheDir, tt.args.src, tt.args.dest, tt.args.mode, tt.args.quiet); (err != nil) != tt.wantErr {
				t.Errorf("DownloadWithCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCachePath(t *testing.T) {
	type args struct {
		cacheDir string
		src      string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "TestgetCachePath",
			args: args{
				cacheDir: "/local/cache",
				src:      "http://example.com/path/to/resource",
			},
			want:    "/local/cache/http/example.com/path/to/resource",
			wantErr: false,
		},
		{
			name: "TestgetCachePath",
			args: args{
				cacheDir: "/local/cache",
				src:      "http://example.com/path/to/resource?query=value&other=param",
			},
			want:    "/local/cache/http/example.com/path/to/resource",
			wantErr: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCachePath(tt.args.cacheDir, tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCachePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCachePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCacheOrDownload(t *testing.T) {
	type args struct {
		ctx      context.Context
		cacheDir string
		src      string
		mode     fs.FileMode
		quiet    bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "TestingwithNoErr",
			args: args{
				ctx:      context.Background(),
				cacheDir: "/tmp/cache",
				src:      "http://example.com/data.txt",
				mode:     fs.FileMode(0644),
				quiet:    false},
			want:    "/tmp/cache/http/example.com/data.txt",
			wantErr: false,
		},
		{
			name: "TestingwithErr",
			args: args{
				ctx:      context.Background(),
				cacheDir: "/tmp/cache",
				src:      "http://%41:8080/data.txt",
				mode:     fs.FileMode(0644),
				quiet:    false},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.MkdirAll(path.Dir(tt.want), 0750); err != nil {
				t.Fatalf("Failed to close : %v", err)
			}
			file, _ := os.Create(tt.want)
			if err := file.Close(); err != nil {
				t.Fatalf("Failed to close : %v", err)
			}
			got, err := getCacheOrDownload(tt.args.ctx, tt.args.cacheDir, tt.args.src, tt.args.mode, tt.args.quiet)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCacheOrDownload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCacheOrDownload() = %v, want %v", got, tt.want)
			}
		})
	}
}
