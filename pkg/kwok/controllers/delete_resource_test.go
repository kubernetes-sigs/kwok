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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestDeleteResourceIsIdempotent(t *testing.T) {
	ctx := context.Background()

	pod := &corev1.Pod{Name: "pod", Namespace: "default"}
	node := &corev1.Node{Name: "node"}
	typedClient := kubefake.NewClientset(pod, node)

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	pv := &corev1.PersistentVolume{Name: "volume"}
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, pv)
	gvr := corev1.SchemeGroupVersion.WithResource("persistentvolumes")
	stageController := &StageController{dynamicClient: dynamicClient, gvr: gvr}
	stageResource := &unstructured.Unstructured{}
	stageResource.SetAPIVersion("v1")
	stageResource.SetKind("PersistentVolume")
	stageResource.SetName(pv.Name)

	cases := []struct {
		name         string
		delete       func() error
		verifyAbsent func() error
	}{
		{
			name: "pod",
			delete: func() error {
				return (&PodController{typedClient: typedClient}).deleteResource(ctx, pod)
			},
			verifyAbsent: func() error {
				_, err := typedClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("get after delete returned %v, want not found", err)
				}
				return nil
			},
		},
		{
			name: "node",
			delete: func() error {
				return (&NodeController{typedClient: typedClient}).deleteResource(ctx, node)
			},
			verifyAbsent: func() error {
				_, err := typedClient.CoreV1().Nodes().Get(ctx, node.Name, metav1.GetOptions{})
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("get after delete returned %v, want not found", err)
				}
				return nil
			},
		},
		{
			name: "generic stage resource",
			delete: func() error {
				return stageController.deleteResource(ctx, stageResource)
			},
			verifyAbsent: func() error {
				_, err := dynamicClient.Resource(gvr).Get(ctx, pv.Name, metav1.GetOptions{})
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("get after delete returned %v, want not found", err)
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for attempt := 1; attempt <= 2; attempt++ {
				if err := tc.delete(); err != nil {
					t.Fatalf("delete attempt %d: %v", attempt, err)
				}
				if attempt == 1 {
					if err := tc.verifyAbsent(); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}

func TestDeleteResourcePreservesErrorsAndDeletePreconditions(t *testing.T) {
	ctx := context.Background()
	uid := types.UID("expected-uid")
	resourceVersion := "17"
	originalDeleteOptions := deleteOpt
	deleteOpt = metav1.DeleteOptions{
		GracePeriodSeconds: originalDeleteOptions.GracePeriodSeconds,
		Preconditions: &metav1.Preconditions{
			UID:             &uid,
			ResourceVersion: &resourceVersion,
		},
	}
	t.Cleanup(func() {
		deleteOpt = originalDeleteOptions
	})
	wantOptions := deleteOpt

	checkOptions := func(action ktesting.Action) {
		t.Helper()
		deleteAction, ok := action.(ktesting.DeleteAction)
		if !ok {
			t.Errorf("expected delete action, got %T", action)
			return
		}
		if got := deleteAction.GetDeleteOptions(); !reflect.DeepEqual(got, wantOptions) {
			t.Errorf("delete options = %#v, want %#v", got, wantOptions)
		}
	}

	notFound := apierrors.NewNotFound(schema.GroupResource{Resource: "resources"}, "missing")
	forbidden := apierrors.NewForbidden(corev1.Resource("pods"), "pod", errors.New("forbidden"))
	conflict := apierrors.NewConflict(corev1.Resource("nodes"), "node", errors.New("conflict"))
	failures := []struct {
		name        string
		err         error
		wantNoError bool
	}{
		{name: "not-found", err: notFound, wantNoError: true},
		{name: "forbidden", err: forbidden},
		{name: "conflict", err: conflict},
		{name: "timeout", err: context.DeadlineExceeded},
	}

	for _, failure := range failures {
		for _, tc := range newDeleteResourceErrorCases(t, checkOptions) {
			t.Run(tc.name+"/"+failure.name, func(t *testing.T) {
				got := tc.delete(ctx, failure.err)
				if failure.wantNoError {
					if got != nil {
						t.Fatalf("delete error = %v, want nil", got)
					}
					return
				}
				if !errors.Is(got, failure.err) {
					t.Fatalf("delete error = %v, want %v", got, failure.err)
				}
			})
		}
	}
}

type deleteResourceErrorCase struct {
	name   string
	delete func(context.Context, error) error
}

func newDeleteResourceErrorCases(t *testing.T, checkOptions func(ktesting.Action)) []deleteResourceErrorCase {
	t.Helper()
	return []deleteResourceErrorCase{
		{
			name: "pod",
			delete: func(ctx context.Context, deleteErr error) error {
				client := kubefake.NewClientset()
				client.PrependReactor("delete", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
					checkOptions(action)
					return true, nil, deleteErr
				})
				return (&PodController{typedClient: client}).deleteResource(ctx, &corev1.Pod{Name: "pod", Namespace: "default"})
			},
		},
		{
			name: "node",
			delete: func(ctx context.Context, deleteErr error) error {
				client := kubefake.NewClientset()
				client.PrependReactor("delete", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
					checkOptions(action)
					return true, nil, deleteErr
				})
				return (&NodeController{typedClient: client}).deleteResource(ctx, &corev1.Node{Name: "node"})
			},
		},
		{
			name: "generic stage resource",
			delete: func(ctx context.Context, deleteErr error) error {
				gvr := schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumes"}
				client := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
				client.PrependReactor("delete", "persistentvolumes", func(action ktesting.Action) (bool, runtime.Object, error) {
					checkOptions(action)
					return true, nil, deleteErr
				})
				resource := &unstructured.Unstructured{}
				resource.SetAPIVersion("v1")
				resource.SetKind("PersistentVolume")
				resource.SetName("volume")
				return (&StageController{dynamicClient: client, gvr: gvr}).deleteResource(ctx, resource)
			},
		},
	}
}
