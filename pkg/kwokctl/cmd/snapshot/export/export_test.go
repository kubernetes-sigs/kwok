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

package export

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

func TestExportPreservesTimestamps(t *testing.T) {
	const timestamp = "2026-01-01T00:00:00Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1":
			_, _ = fmt.Fprint(w, `{
				"kind": "APIResourceList", "apiVersion": "v1", "groupVersion": "v1",
				"resources": [{
					"name": "configmaps", "singularName": "configmap", "namespaced": true,
					"kind": "ConfigMap", "verbs": ["get", "list"]
				}]
			}`)
		case "/api/v1/configmaps":
			_, _ = fmt.Fprintf(w, `{
				"kind": "ConfigMapList", "apiVersion": "v1", "metadata": {"resourceVersion": "1"},
				"items": [{
					"kind": "ConfigMap", "apiVersion": "v1",
					"metadata": {"name": "example", "namespace": "default", "creationTimestamp": %q},
					"data": {"timestamp": %q}
				}]
			}`, timestamp, timestamp)
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	kubeconfigPath := filepath.Join(t.TempDir(), "kubeconfig")
	config := kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
		ProjectName: "test",
		Address:     server.URL,
	})
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"snapshot.yaml", "snapshot.yaml.gz"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			cmd := NewCommand(t.Context())
			cmd.SetArgs([]string{"--kubeconfig", kubeconfigPath, "--path", path, "--filter", "configmaps"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}

			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			r, err := file.Decompress(path, f)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()

			var cm corev1.ConfigMap
			if err := yaml.NewDecoder(r).Decode(&cm); err != nil {
				t.Fatalf("exported snapshot is not a valid ConfigMap: %v", err)
			}
			if want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC); !cm.CreationTimestamp.Time.Equal(want) {
				t.Errorf("creationTimestamp = %v, want %v", cm.CreationTimestamp, want)
			}
			if got := cm.Data["timestamp"]; got != timestamp {
				t.Errorf("data timestamp = %q, want %q", got, timestamp)
			}
		})
	}
}
