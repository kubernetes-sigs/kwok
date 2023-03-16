/*
Copyright 2023 The Kubernetes Authors.

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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	yamlv3 "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// Load loads the resources to cluster from the reader
// This is a wrapper around `kubectl apply`, which will handle the ownerReference
// so that the resources remain relative to each other
func Load(ctx context.Context, rt Runtime, r io.Reader, filters []string) error {
	objs, err := decodeObjects(r)
	if err != nil {
		return err
	}

	// Filter out the resources that are not in the filters
	filterMap := map[string]struct{}{}
	for _, filter := range filters {
		filterMap[filter] = struct{}{}
	}

	objs = slices.Filter(objs, func(obj *unstructured.Unstructured) bool {
		gvk := obj.GetObjectKind().GroupVersionKind()
		// These are built-in resources that do not need to be created
		//nolint:goconst
		switch gvk.Kind {
		case "Namespace":
			if obj.GetName() == "kube-public" ||
				obj.GetName() == "kube-node-lease" ||
				obj.GetName() == "kube-system" ||
				obj.GetName() == "default" {
				return false
			}
		case "ServiceAccount":
			if obj.GetName() == "default" {
				return false
			}
		case "Service", "Endpoints":
			if obj.GetName() == "kubernetes" &&
				obj.GetNamespace() == "default" {
				return false
			}
		case "Secret":
			if obj.GetName() == "default-token" {
				return false
			}
		case "ConfigMap":
			if obj.GetName() == "kube-root-ca.crt" {
				return false
			}
		}

		_, ok := filterMap[strings.ToLower(gvk.GroupKind().String())]
		return ok
	})

	inputRaw := bytes.NewBuffer(nil)
	outputRaw := bytes.NewBuffer(nil)
	otherResource, err := load(objs, func(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
		inputRaw.Reset()
		outputRaw.Reset()

		encoder := json.NewEncoder(inputRaw)
		for _, obj := range objs {
			err = encoder.Encode(obj)
			if err != nil {
				return nil, err
			}
		}

		err = rt.KubectlInCluster(exec.WithIOStreams(ctx, exec.IOStreams{
			In:     inputRaw,
			Out:    outputRaw,
			ErrOut: os.Stderr,
		}), "apply", "--validate=false", "-o=json", "-f", "-")
		if err != nil {
			for _, obj := range objs {
				fmt.Fprintf(os.Stderr, "%s/%s failed\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
			}
		}
		newObj, err := decodeObjects(outputRaw)
		if err != nil {
			return nil, err
		}
		for _, obj := range newObj {
			fmt.Fprintf(os.Stderr, "%s/%s succeed\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
		}
		return newObj, nil
	})
	if err != nil {
		return err
	}
	for _, obj := range otherResource {
		fmt.Fprintf(os.Stderr, "%s/%s skipped\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
	}
	return nil
}

func decodeObjects(data io.Reader) ([]*unstructured.Unstructured, error) {
	out := []*unstructured.Unstructured{}
	tmp := map[string]interface{}{}
	decoder := yamlv3.NewDecoder(data)
	for {
		err := decoder.Decode(&tmp)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		data, err := yamlv3.Marshal(tmp)
		if err != nil {
			return nil, err
		}
		data, err = yaml.YAMLToJSON(data)
		if err != nil {
			return nil, err
		}
		obj := &unstructured.Unstructured{}
		err = obj.UnmarshalJSON(data)
		if err != nil {
			return nil, err
		}

		if obj.IsList() {
			err = obj.EachListItem(func(object runtime.Object) error {
				out = append(out, object.(*unstructured.Unstructured))
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			out = append(out, obj)
		}
	}
	return out, nil
}

func load(input []*unstructured.Unstructured, apply func([]*unstructured.Unstructured) ([]*unstructured.Unstructured, error)) ([]*unstructured.Unstructured, error) {
	applyResource := []*unstructured.Unstructured{}
	otherResource := []*unstructured.Unstructured{}

	for _, obj := range input {
		refs := obj.GetOwnerReferences()
		if len(refs) != 0 && refs[0].Controller != nil && *refs[0].Controller {
			otherResource = append(otherResource, obj)
		} else {
			applyResource = append(applyResource, obj)
		}
	}

	for len(applyResource) != 0 {
		nextApplyResource := []*unstructured.Unstructured{}
		newResource, err := apply(applyResource)
		if err != nil {
			return nil, err
		}
		if len(otherResource) == 0 {
			break
		}
		for i, newObj := range newResource {
			oldUID := applyResource[i].GetUID()
			newUID := newObj.GetUID()

			remove := map[*unstructured.Unstructured]struct{}{}
			nextResource := slices.Filter(otherResource, func(otherObj *unstructured.Unstructured) bool {
				otherRefs := otherObj.GetOwnerReferences()
				otherRef := &otherRefs[0]
				if otherRef.UID != oldUID {
					return false
				}
				otherRef.UID = newUID
				otherObj.SetOwnerReferences(otherRefs)
				remove[otherObj] = struct{}{}
				return true
			})
			if len(remove) != 0 {
				otherResource = slices.Filter(otherResource, func(otherObj *unstructured.Unstructured) bool {
					_, ok := remove[otherObj]
					return !ok
				})
				nextApplyResource = append(nextApplyResource, nextResource...)
			}
		}
		applyResource = nextApplyResource
	}
	return otherResource, nil
}
