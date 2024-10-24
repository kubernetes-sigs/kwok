/*
Copyright 2024 The Kubernetes Authors.

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
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
)

// Next represents the next step in the lifecycle
type Next struct {
	next *internalversion.StageNext
}

// newNext creates a new Next from the stage
func newNext(next *internalversion.StageNext) *Next {
	return &Next{
		next: next,
	}
}

// Finalizers returns the finalizers patch
func (n *Next) Finalizers(metaFinalizers []string) (*Patch, error) {
	if n.next.Finalizers == nil {
		return nil, nil
	}
	ops := finalizersModify(metaFinalizers, n.next.Finalizers)
	if len(ops) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(ops)
	if err != nil {
		return nil, err
	}

	return &Patch{
		Data: data,
		Type: types.JSONPatchType,
	}, nil
}

// Event returns the event for the resource
func (n *Next) Event() *internalversion.StageEvent {
	return n.next.Event
}

// Delete returns whether the resource should be deleted
func (n *Next) Delete() bool {
	return n.next.Delete
}

// Patches returns the patches for the resource
func (n *Next) Patches(resource any, renderer gotpl.Renderer) ([]*Patch, error) {
	patches := make([]*Patch, 0, len(n.next.Patches))
	for _, patch := range n.next.Patches {
		patchData, patchType, err := computePatch(renderer, resource, patch)
		if err != nil {
			return nil, err
		}
		patches = append(patches, &Patch{
			Data:          patchData,
			Type:          patchType,
			Subresource:   patch.Subresource,
			Impersonation: patch.Impersonation,
		})
	}
	return patches, nil
}

// Patch represents a patch for the resource
type Patch struct {
	Data          []byte
	Type          types.PatchType
	Subresource   string
	Impersonation *internalversion.ImpersonationConfig
}

func computePatch(renderer gotpl.Renderer, resource any, patch internalversion.StagePatch) ([]byte, types.PatchType, error) {
	switch format.ElemOrDefault(patch.Type) {
	case internalversion.StagePatchTypeJSONPatch:
		patchData, err := computeJSONPatch(renderer, resource, patch.Root, patch.Template)
		if err != nil {
			return nil, "", err
		}
		return patchData, types.JSONPatchType, nil
	case internalversion.StagePatchTypeStrategicMergePatch:
		patchData, err := computeMergePatch(renderer, resource, patch.Root, patch.Template)
		if err != nil {
			return nil, "", err
		}
		return patchData, types.StrategicMergePatchType, nil
	case internalversion.StagePatchTypeMergePatch:
		patchData, err := computeMergePatch(renderer, resource, patch.Root, patch.Template)
		if err != nil {
			return nil, "", err
		}
		return patchData, types.MergePatchType, nil
	case "":
		switch resource.(type) {
		case *corev1.Node, *corev1.Pod:
			patchData, err := computeMergePatch(renderer, resource, patch.Root, patch.Template)
			if err != nil {
				return nil, "", err
			}
			return patchData, types.StrategicMergePatchType, nil
		}
		patchData, err := computeMergePatch(renderer, resource, patch.Root, patch.Template)
		if err != nil {
			return nil, "", err
		}
		return patchData, types.MergePatchType, nil
	}

	return nil, "", fmt.Errorf("unknown patch type %s", *patch.Type)
}

func computeMergePatch(renderer gotpl.Renderer, resource any, root, tpl string) ([]byte, error) {
	patchData, err := renderer.ToJSON(tpl, resource)
	if err != nil {
		return nil, err
	}

	return wrapMergePatchData(root, patchData)
}

func computeJSONPatch(renderer gotpl.Renderer, resource any, root, tpl string) ([]byte, error) {
	patchData, err := renderer.ToJSON(tpl, resource)
	if err != nil {
		return nil, err
	}

	patchData, err = wrapJSONPatchData(root, patchData)
	if err != nil {
		return nil, err
	}

	return patchData, nil
}

// wrapMergePatchData wraps the patch data with the parent key
func wrapMergePatchData(parent string, patchData []byte) ([]byte, error) {
	if parent == "" {
		return patchData, nil
	}
	return json.Marshal(map[string]json.RawMessage{
		parent: patchData,
	})
}

// wrapJSONPatchData wraps the patch data with the parent key
func wrapJSONPatchData(parent string, patchData []byte) ([]byte, error) {
	if parent == "" {
		return patchData, nil
	}

	var data []jsonpatchData
	err := json.Unmarshal(patchData, &data)
	if err != nil {
		return nil, err
	}

	for i := range data {
		data[i].Path = "/" + parent + data[i].Path
	}

	return json.Marshal(data)
}

type jsonpatchData struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
	From  string          `json:"from,omitempty"`
}
