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

package snapshot

import (
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/fake"

	utilsyaml "sigs.k8s.io/kwok/pkg/utils/yaml"
)

func TestLoaderContinuesAfterFilteredResource(t *testing.T) {
	ctx := t.Context()
	groupVersion := schema.GroupVersion{Version: "v1"}
	podGVK := groupVersion.WithKind("Pod")
	podGVR := groupVersion.WithResource("pods")
	namespaceGVR := groupVersion.WithResource("namespaces")

	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{groupVersion})
	restMapper.Add(podGVK, meta.RESTScopeNamespace)
	dynamicClient := fake.NewSimpleDynamicClient(runtime.NewScheme())
	loader := &Loader{
		exist:         make(map[uniqueKey]types.UID),
		pending:       make(map[uniqueKey][]*unstructured.Unstructured),
		restMapper:    restMapper,
		dynamicClient: dynamicClient,
		loadConfig: LoadConfig{
			Filters: []*meta.RESTMapping{
				{
					Resource:         podGVR,
					GroupVersionKind: podGVK,
					Scope:            meta.RESTScopeNamespace,
				},
			},
		},
	}

	decoder := utilsyaml.NewDecoder(strings.NewReader(`
apiVersion: v1
kind: Namespace
metadata:
  name: filtered
---
apiVersion: v1
kind: Pod
metadata:
  name: restored
  namespace: default
`))
	if err := loader.Load(ctx, decoder); err != nil {
		t.Fatalf("failed to load snapshot: %v", err)
	}

	if _, err := dynamicClient.Resource(namespaceGVR).Get(ctx, "filtered", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected filtered namespace to remain absent, got: %v", err)
	}
	_, err := dynamicClient.Resource(podGVR).Namespace("default").Get(ctx, "restored", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected pod after filtered resource to be restored: %v", err)
	}
}
