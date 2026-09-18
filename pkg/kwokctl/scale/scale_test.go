/*
Copyright The Kubernetes Authors.

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

package scale

import (
	"context"
	"errors"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"sigs.k8s.io/kwok/pkg/utils/client"
)

func TestScaleReturnsDeleteErrorsAndContinues(t *testing.T) {
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "nodes"}

	nodes := []runtime.Object{
		newScaledNode("test-000000"),
		newScaledNode("test-000001"),
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "NodeList"},
		nodes...,
	)

	var attempted []string
	dynamicClient.PrependReactor("delete", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		name := action.(k8stesting.DeleteAction).GetName()
		attempted = append(attempted, name)
		if name == "test-000000" {
			return true, nil, apierrors.NewForbidden(gvr.GroupResource(), name, errors.New("deletion denied"))
		}
		return false, nil, nil
	})

	err := Scale(context.Background(), &testClientset{
		dynamicClient: dynamicClient,
		mapper:        &testRESTMapper{gvr: gvr},
	}, Config{
		Template:     "apiVersion: v1\nkind: Node\nmetadata: {}\n",
		Name:         "test",
		Replicas:     0,
		SerialLength: 6,
	})
	if err == nil || !strings.Contains(err.Error(), "deletion denied") {
		t.Fatalf("Scale() error = %v, want deletion error", err)
	}
	if len(attempted) != 2 {
		t.Fatalf("attempted deletes = %v, want both nodes", attempted)
	}
	if _, err := dynamicClient.Resource(gvr).Get(context.Background(), "test-000000", metav1.GetOptions{}); err != nil {
		t.Fatalf("denied node was removed: %v", err)
	}
	if _, err := dynamicClient.Resource(gvr).Get(context.Background(), "test-000001", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("allowed node still exists: %v", err)
	}
}

func newScaledNode(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata": map[string]any{
			"name": name,
			"labels": map[string]any{
				labelNameKey: "test",
			},
		},
	}}
}

type testClientset struct {
	client.Clientset
	dynamicClient dynamic.Interface
	mapper        meta.RESTMapper
}

type testRESTMapper struct {
	meta.RESTMapper
	gvr schema.GroupVersionResource
}

func (m *testRESTMapper) ResourceFor(schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return m.gvr, nil
}

func (c *testClientset) ToDynamicClient() (dynamic.Interface, error) {
	return c.dynamicClient, nil
}

func (c *testClientset) ToRESTMapper() (meta.RESTMapper, error) {
	return c.mapper, nil
}
