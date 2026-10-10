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
	"fmt"
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
	clienttesting "k8s.io/client-go/testing"

	utilsyaml "sigs.k8s.io/kwok/pkg/utils/yaml"
)

func TestLoaderRestoresOwnerChain(t *testing.T) {
	for _, tc := range []struct {
		name   string
		names  []string
		owners map[string][]string
	}{
		{
			name:   "ordered",
			names:  []string{"parent", "child", "grandchild"},
			owners: map[string][]string{"child": {"parent"}, "grandchild": {"child"}},
		},
		{
			name:   "reversed",
			names:  []string{"grandchild", "child", "parent"},
			owners: map[string][]string{"child": {"parent"}, "grandchild": {"child"}},
		},
		{
			name:   "mixed",
			names:  []string{"child", "grandchild", "parent"},
			owners: map[string][]string{"child": {"parent"}, "grandchild": {"child"}},
		},
		{
			name:   "shared-owner",
			names:  []string{"child", "grandchild", "parent"},
			owners: map[string][]string{"child": {"parent"}, "grandchild": {"child", "parent"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			gv := schema.GroupVersion{Version: "v1"}
			gvr := gv.WithResource("configmaps")
			mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gv})
			mapper.Add(gv.WithKind("ConfigMap"), meta.RESTScopeNamespace)
			dynamicClient := fake.NewSimpleDynamicClient(runtime.NewScheme())
			createCalls := make(map[string]int)
			dynamicClient.PrependReactor("create", "configmaps", func(action clienttesting.Action) (bool, runtime.Object, error) {
				obj := action.(clienttesting.CreateAction).GetObject().(*unstructured.Unstructured)
				createCalls[obj.GetName()]++
				created := obj.DeepCopy()
				created.SetUID(types.UID("restored-" + obj.GetName()))
				if err := dynamicClient.Tracker().Create(gvr, created, "default"); err != nil {
					return true, nil, err
				}
				return true, created, nil
			})
			loader := &Loader{
				exist: make(map[uniqueKey]types.UID), pending: make(map[uniqueKey][]*unstructured.Unstructured),
				restMapper: mapper, dynamicClient: dynamicClient, loadConfig: LoadConfig{NoFilers: true},
			}
			var input strings.Builder
			for _, name := range tc.names {
				fmt.Fprintf(&input, "---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: %s\n  namespace: default\n  uid: original-%s\n", name, name)
				if len(tc.owners[name]) != 0 {
					input.WriteString("  ownerReferences:\n")
				}
				for _, owner := range tc.owners[name] {
					fmt.Fprintf(&input, "  - apiVersion: v1\n    kind: ConfigMap\n    name: %s\n    uid: original-%s\n", owner, owner)
				}
			}
			if err := loader.Load(ctx, utilsyaml.NewDecoder(strings.NewReader(input.String()))); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"parent", "child", "grandchild"} {
				obj, err := dynamicClient.Resource(gvr).Namespace("default").Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("%s was not restored: %v", name, err)
				}
				refs := obj.GetOwnerReferences()
				if len(refs) != len(tc.owners[name]) {
					t.Fatalf("%s owner references = %v, want owners %v", name, refs, tc.owners[name])
				}
				for i, owner := range tc.owners[name] {
					if refs[i].UID != types.UID("restored-"+owner) {
						t.Fatalf("%s owner reference = %v, want UID restored-%s", name, refs[i], owner)
					}
				}
				if createCalls[name] != 1 {
					t.Errorf("%s was created %d times, want once", name, createCalls[name])
				}
			}
		})
	}
}

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
