/*
Copyright 2022 The Kubernetes Authors.

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
	"strconv"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

type jsonpathOperation struct {
	Op    string      `json:"op,omitempty"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

func finalizersAdd(metaFinalizers []string, finalizers []internalversion.FinalizerItem) []jsonpathOperation {
	var ops []jsonpathOperation

	finalizersValue := make([]string, 0, len(finalizers))
	for _, finalizer := range finalizers {
		finalizersValue = append(finalizersValue, finalizer.Value)
	}
	if len(metaFinalizers) != 0 {
		for _, finalizer := range finalizersValue {
			if slices.Contains(metaFinalizers, finalizer) {
				continue
			}
			ops = append(ops, jsonpathOperation{
				Op:    "add",
				Path:  "/metadata/finalizers/-",
				Value: finalizer,
			})
		}
	} else {
		ops = append(ops, jsonpathOperation{
			Op:    "add",
			Path:  "/metadata/finalizers",
			Value: finalizersValue,
		})
	}

	return ops
}

func finalizersRemove(metaFinalizers []string, finalizers []internalversion.FinalizerItem) []jsonpathOperation {
	var ops []jsonpathOperation

	finalizersValue := make([]string, 0, len(finalizers))
	for _, finalizer := range finalizers {
		finalizersValue = append(finalizersValue, finalizer.Value)
	}

	for i := len(metaFinalizers) - 1; i >= 0; i-- {
		metaFinalizer := metaFinalizers[i]
		if !slices.Contains(finalizersValue, metaFinalizer) {
			continue
		}
		ops = append(ops, jsonpathOperation{
			Op:   "remove",
			Path: "/metadata/finalizers/" + strconv.FormatInt(int64(i), 10),
		})
	}

	return ops
}

func finalizersModify(metaFinalizers []string, finalizers *internalversion.StageFinalizers) []jsonpathOperation {
	isEmpty := false
	var ops []jsonpathOperation
	if finalizers.Empty {
		isEmpty = true
	} else if len(finalizers.Remove) != 0 {
		removed := finalizersRemove(metaFinalizers, finalizers.Remove)
		if len(removed) == len(metaFinalizers) {
			isEmpty = true
		} else {
			ops = append(ops, removed...)
		}
	}

	if !isEmpty {
		if len(finalizers.Add) != 0 {
			ops = append(ops, finalizersAdd(metaFinalizers, finalizers.Add)...)
		}
	} else {
		if len(metaFinalizers) != 0 {
			ops = append(ops, finalizersEmpty)
		}

		if len(finalizers.Add) != 0 {
			ops = append(ops, finalizersAdd(nil, finalizers.Add)...)
		}
	}
	return ops
}

var finalizersEmpty = jsonpathOperation{
	Op:   "remove",
	Path: "/metadata/finalizers",
}
