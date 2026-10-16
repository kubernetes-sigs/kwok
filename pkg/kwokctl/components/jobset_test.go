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
	"strings"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

// jobsetWebhookManifest returns admission webhook configurations, trimmed to what kwokctl reads,
// with webhooks on the given paths, in the shape of the upstream jobset manifest.
func jobsetWebhookManifest(mutatingPaths, validatingPaths []string) string {
	configuration := func(kind, name string, paths []string) string {
		var b strings.Builder
		b.WriteString("apiVersion: admissionregistration.k8s.io/v1\nkind: " + kind + "\nmetadata:\n  name: " + name + "\nwebhooks:\n")
		for i, path := range paths {
			b.WriteString("- name: webhook" + string(rune('a'+i)) + ".kb.io\n" +
				"  admissionReviewVersions: [v1]\n" +
				"  sideEffects: None\n" +
				"  clientConfig:\n" +
				"    service:\n" +
				"      name: jobset-webhook-service\n" +
				"      namespace: jobset-system\n" +
				"      path: " + path + "\n")
		}
		return b.String()
	}
	return strings.Join([]string{
		"apiVersion: v1\nkind: Service\nmetadata:\n  name: jobset-webhook-service\n  namespace: jobset-system\nspec:\n  ports:\n  - port: 443\n    targetPort: 9443\n",
		configuration("MutatingWebhookConfiguration", "jobset-mutating-webhook-configuration", mutatingPaths),
		configuration("ValidatingWebhookConfiguration", "jobset-validating-webhook-configuration", validatingPaths),
	}, "---\n")
}

func buildJobSetWithManifests(t *testing.T, manifests ...string) (internalversion.Component, error) {
	t.Helper()
	caCertPath := utilspath.Join(t.TempDir(), "ca.crt")
	if err := os.WriteFile(caCertPath, []byte("ca"), 0o600); err != nil {
		t.Fatal(err)
	}
	return BuildJobSetComponent(BuildJobSetComponentConfig{
		Runtime:      consts.RuntimeTypeDocker,
		ProjectName:  "kwok-test",
		Image:        "registry.k8s.io/jobset/jobset:v0.12.0",
		CaCertPath:   caCertPath,
		RawManifests: manifests,
	})
}

var jobsetReadyzProbe = &internalversion.ComponentProbe{
	HTTPGet: &internalversion.ComponentHTTPGetAction{
		Path: "/readyz",
		Port: 8081,
	},
	TimeoutSeconds: 5,
}

func TestBuildJobSetComponentReadinessProbe(t *testing.T) {
	upstreamMutatingPaths := []string{"/mutate-jobset-x-k8s-io-v1alpha2-jobset", "/mutate--v1-pod"}
	upstreamValidatingPaths := []string{"/validate-jobset-x-k8s-io-v1alpha2-jobset", "/validate--v1-pod"}

	tests := []struct {
		name      string
		manifests []string
		want      *internalversion.ComponentProbe
		wantErr   bool
	}{
		{
			// The webhook registered last is probed over the TLS of the webhook server.
			name:      "admission webhooks",
			manifests: []string{jobsetWebhookManifest(upstreamMutatingPaths, upstreamValidatingPaths)},
			want: &internalversion.ComponentProbe{
				HTTPGet: &internalversion.ComponentHTTPGetAction{
					Path:   "/validate--v1-pod",
					Port:   9443,
					Scheme: internalversion.URISchemeHTTPS,
				},
				TimeoutSeconds: 5,
			},
		},
		{
			name: "admission webhooks in separate manifests",
			manifests: []string{
				jobsetWebhookManifest(upstreamMutatingPaths, nil),
				jobsetWebhookManifest(nil, upstreamValidatingPaths),
			},
			want: &internalversion.ComponentProbe{
				HTTPGet: &internalversion.ComponentHTTPGetAction{
					Path:   "/validate--v1-pod",
					Port:   9443,
					Scheme: internalversion.URISchemeHTTPS,
				},
				TimeoutSeconds: 5,
			},
		},
		{
			// Readiness cannot be checked on a webhook that is not configured, so this is not silently /readyz.
			name:      "admission webhooks without the probed path",
			manifests: []string{jobsetWebhookManifest(upstreamMutatingPaths, []string{"/validate-jobset-x-k8s-io-v1alpha2-jobset"})},
			wantErr:   true,
		},
		{
			name:      "manifests without admission webhooks",
			manifests: []string{"apiVersion: v1\nkind: Service\nmetadata:\n  name: jobset-webhook-service\n  namespace: jobset-system\nspec:\n  ports:\n  - port: 443\n"},
			want:      jobsetReadyzProbe,
		},
		{
			name: "no manifests",
			want: jobsetReadyzProbe,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, err := buildJobSetWithManifests(t, tt.manifests...)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "/validate--v1-pod") {
					t.Fatalf("BuildJobSetComponent() error = %v, want an error naming the missing path", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(component.ReadinessProbe, tt.want) {
				t.Errorf("ReadinessProbe = %#v, want %#v", component.ReadinessProbe, tt.want)
			}
			if err := ReadinessProbeConflict(component); err != nil {
				t.Errorf("ReadinessProbeConflict() = %v, want none", err)
			}
		})
	}
}

func TestWebhookServicePaths(t *testing.T) {
	manifest := jobsetWebhookManifest([]string{"/m"}, []string{"/v1", "/v2"}) +
		"---\napiVersion: admissionregistration.k8s.io/v1beta1\nkind: ValidatingWebhookConfiguration\nmetadata:\n  name: old\nwebhooks:\n- name: old.kb.io\n  clientConfig:\n    service:\n      path: /old\n"
	got, err := webhookServicePaths([]string{manifest})
	if err != nil {
		t.Fatal(err)
	}
	// Only admissionregistration.k8s.io/v1, which kwokctl transforms, is read.
	if want := []string{"/m", "/v1", "/v2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("webhookServicePaths() = %q, want %q", got, want)
	}

	if _, err := webhookServicePaths([]string{"kind: [\n"}); err == nil {
		t.Error("webhookServicePaths() of an invalid manifest succeeded, want an error")
	}
}
