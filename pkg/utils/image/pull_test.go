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

package image

import (
	"archive/tar"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/random"
)

func TestSaveOCIArchive(t *testing.T) {
	img, err := random.Image(256, 1)
	if err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "test.tar")
	ref := "docker.io/library/test:latest"

	err = saveOCIArchive(img, ref, dest)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the archive is a valid tar
	f, err := os.Open(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	tr := tar.NewReader(f)
	files := map[string]bool{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		files[header.Name] = true
	}

	// Verify OCI layout files exist
	if !files["oci-layout"] {
		t.Error("missing oci-layout file")
	}
	if !files["index.json"] {
		t.Error("missing index.json file")
	}

	// Verify blobs directory exists
	hasBlobsDir := false
	for name := range files {
		if len(name) > 6 && name[:6] == "blobs/" {
			hasBlobsDir = true
			break
		}
	}
	if !hasBlobsDir {
		t.Error("missing blobs directory")
	}
}

func TestSaveOCIArchiveAnnotation(t *testing.T) {
	img, err := random.Image(256, 1)
	if err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "test.tar")
	ref := "docker.io/library/test:latest"

	err = saveOCIArchive(img, ref, dest)
	if err != nil {
		t.Fatal(err)
	}

	// Read index.json from the archive and check annotation
	f, err := os.Open(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	tr := tar.NewReader(f)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Name == "index.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			var index struct {
				Manifests []struct {
					Annotations map[string]string `json:"annotations"`
				} `json:"manifests"`
			}
			if err := json.Unmarshal(data, &index); err != nil {
				t.Fatal(err)
			}
			if len(index.Manifests) == 0 {
				t.Fatal("no manifests in index.json")
			}
			gotRef := index.Manifests[0].Annotations["org.opencontainers.image.ref.name"]
			if gotRef != ref {
				t.Errorf("expected ref annotation %q, got %q", ref, gotRef)
			}
			return
		}
	}
	t.Error("index.json not found in archive")
}

func TestTarDir(t *testing.T) {
	srcDir := t.TempDir()

	// Create some test files
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("world"), 0600); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "test.tar")
	err := tarDir(srcDir, dest)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the tar contents
	f, err := os.Open(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	tr := tar.NewReader(f)
	files := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			files[header.Name] = string(data)
		}
	}

	if files["file1.txt"] != "hello" {
		t.Errorf("expected file1.txt content 'hello', got %q", files["file1.txt"])
	}
	if files["subdir/file2.txt"] != "world" {
		t.Errorf("expected subdir/file2.txt content 'world', got %q", files["subdir/file2.txt"])
	}
}
