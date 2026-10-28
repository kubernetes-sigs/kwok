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

package components

import (
	"os"
	"reflect"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

func TestBuildJobSetComponentReadinessProbe(t *testing.T) {
	// The webhook registered last is probed over the TLS of the webhook server.
	want := &internalversion.ComponentProbe{
		HTTPGet: &internalversion.ComponentHTTPGetAction{
			Path:   "/validate--v1-pod",
			Port:   9443,
			Scheme: internalversion.URISchemeHTTPS,
		},
		TimeoutSeconds: 5,
	}

	tests := []struct {
		name      string
		manifests []string
	}{
		{
			name:      "manifests",
			manifests: []string{"apiVersion: v1\nkind: Service\nmetadata:\n  name: jobset-webhook-service\n  namespace: jobset-system\nspec:\n  ports:\n  - port: 443\n"},
		},
		{
			// As in a dry run, where no manifest is read.
			name: "no manifests",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caCertPath := utilspath.Join(t.TempDir(), "ca.crt")
			if err := os.WriteFile(caCertPath, []byte("ca"), 0o600); err != nil {
				t.Fatal(err)
			}
			component, err := BuildJobSetComponent(BuildJobSetComponentConfig{
				Runtime:      consts.RuntimeTypeDocker,
				ProjectName:  "kwok-test",
				Image:        "registry.k8s.io/jobset/jobset:v0.12.0",
				CaCertPath:   caCertPath,
				RawManifests: tt.manifests,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(component.ReadinessProbe, want) {
				t.Errorf("ReadinessProbe = %#v, want %#v", component.ReadinessProbe, want)
			}
			if err := ReadinessProbeConflict(component); err != nil {
				t.Errorf("ReadinessProbeConflict() = %v, want none", err)
			}
		})
	}
}
