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

package kubeconfig

import (
	"errors"
	"testing"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGetCurrentCluster(t *testing.T) {
	tests := []struct {
		name               string
		kubeConfig         *clientcmdapi.Config
		wantCurrentContext string
		wantClusterName    string
		wantErr            error
	}{
		{
			name: "valid current cluster",
			kubeConfig: &clientcmdapi.Config{
				CurrentContext: "ctx",
				Contexts: map[string]*clientcmdapi.Context{
					"ctx": {
						Cluster: "cluster",
					},
				},
				Clusters: map[string]*clientcmdapi.Cluster{
					"cluster": {
						Server: "https://127.0.0.1:6443",
					},
				},
			},
			wantCurrentContext: "ctx",
			wantClusterName:    "cluster",
		},
		{
			name: "missing current cluster name",
			kubeConfig: &clientcmdapi.Config{
				CurrentContext: "ctx",
				Contexts: map[string]*clientcmdapi.Context{
					"ctx": {},
				},
				Clusters: map[string]*clientcmdapi.Cluster{},
			},
			wantErr: errCurrentClusterNotFound,
		},
		{
			name: "missing current cluster entry",
			kubeConfig: &clientcmdapi.Config{
				CurrentContext: "ctx",
				Contexts: map[string]*clientcmdapi.Context{
					"ctx": {
						Cluster: "cluster",
					},
				},
				Clusters: map[string]*clientcmdapi.Cluster{},
			},
			wantErr: errCurrentClusterNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCurrentContext, gotClusterName, err := getCurrentCluster(tt.kubeConfig)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("getCurrentCluster() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotCurrentContext != tt.wantCurrentContext {
				t.Fatalf("getCurrentCluster() currentContext = %q, want %q", gotCurrentContext, tt.wantCurrentContext)
			}
			if gotClusterName != tt.wantClusterName {
				t.Fatalf("getCurrentCluster() clusterName = %q, want %q", gotClusterName, tt.wantClusterName)
			}
		})
	}
}

func TestGetCurrentClusterAfterMinifyConfig(t *testing.T) {
	kubeConfig := &clientcmdapi.Config{
		CurrentContext: "ctx",
		Contexts: map[string]*clientcmdapi.Context{
			"ctx": {},
		},
		Clusters: map[string]*clientcmdapi.Cluster{},
	}

	if err := clientcmdapi.MinifyConfig(kubeConfig); err != nil {
		t.Fatalf("MinifyConfig() error = %v", err)
	}

	_, _, err := getCurrentCluster(kubeConfig)
	if !errors.Is(err, errCurrentClusterNotFound) {
		t.Fatalf("getCurrentCluster() error = %v, wantErr %v", err, errCurrentClusterNotFound)
	}
}

func TestModifyAddress(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		address string
		want    string
	}{
		{
			name:    "hostname without port keeps original port",
			origin:  "https://127.0.0.1:6443",
			address: "localhost",
			want:    "https://localhost:6443",
		},
		{
			name:    "ipv6 without port keeps original port",
			origin:  "https://127.0.0.1:6443",
			address: "::1",
			want:    "https://[::1]:6443",
		},
		{
			name:    "bracketed ipv6 without port keeps original port",
			origin:  "https://127.0.0.1:6443",
			address: "[::1]",
			want:    "https://[::1]:6443",
		},
		{
			name:    "address with explicit port overrides original port",
			origin:  "https://127.0.0.1:6443",
			address: "[::1]:8443",
			want:    "https://[::1]:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := modifyAddress(tt.origin, tt.address)
			if err != nil {
				t.Fatalf("modifyAddress() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("modifyAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}
