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

package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"sigs.k8s.io/kwok/pkg/consts"
)

// ApplyResource applies the resource rendered from the template output.
func ApplyResource(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	restMapper meta.RESTMapper,
	sourceNamespace string,
	apply *Apply,
) (*unstructured.Unstructured, error) {
	if dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client is required for create step")
	}

	obj := &unstructured.Unstructured{}
	err := json.Unmarshal(apply.Data, &obj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to decode create template: %w", err)
	}

	gvk := obj.GroupVersionKind()
	if gvk.Empty() {
		return nil, fmt.Errorf("template must include apiVersion and kind")
	}

	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to map %s: %w", gvk.String(), err)
	}

	nri := dynamicClient.Resource(mapping.Resource)
	var cli dynamic.ResourceInterface = nri
	if mapping.Scope != nil && mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = sourceNamespace
			obj.SetNamespace(ns)
		}
		cli = nri.Namespace(ns)
	}

	patchOpts := metav1.PatchOptions{}
	if apply.Type == types.ApplyPatchType {
		patchOpts = metav1.PatchOptions{
			FieldManager: consts.ProjectName,
			Force:        new(true),
		}
	}

	var subresources []string
	if apply.Subresource != "" {
		subresources = append(subresources, apply.Subresource)
	}
	result, err := createOrPatchResource(ctx, cli, obj, apply.Type, patchOpts, subresources...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func createOrPatchResource(
	ctx context.Context,
	ri dynamic.ResourceInterface,
	obj *unstructured.Unstructured,
	patchType types.PatchType,
	patchOpts metav1.PatchOptions,
	subresources ...string,
) (*unstructured.Unstructured, error) {
	if ri == nil {
		return nil, fmt.Errorf("resource interface is required")
	}
	if obj == nil {
		return nil, fmt.Errorf("object is required")
	}

	// Subresource patch requires the parent resource to exist already.
	if len(subresources) > 0 && subresources[0] != "" {
		data, err := obj.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal object for patch: %w", err)
		}
		return ri.Patch(ctx, obj.GetName(), patchType, data, patchOpts, subresources...)
	}

	result, err := ri.Create(ctx, obj, metav1.CreateOptions{})
	if err == nil {
		return result, nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	if obj.GetName() == "" {
		return nil, err
	}

	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object for patch: %w", err)
	}

	result, err = ri.Patch(ctx, obj.GetName(), patchType, data, patchOpts)
	if err != nil {
		return nil, err
	}
	return result, nil
}
