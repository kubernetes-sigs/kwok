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

package recording

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"sigs.k8s.io/kwok/pkg/log"
)

// ResourcePatchType is the type of the ResourcePatch.
var ResourcePatchType = metav1.TypeMeta{
	Kind:       "ResourcePatch",
	APIVersion: "kwok.x-k8s.io/internal",
}

// ResourcePatch is the patch of the resource.
type ResourcePatch struct {
	metav1.TypeMeta    `json:",inline"`
	Target             ResourcePatchTarget `json:"target"`
	DurationNanosecond time.Duration       `json:"durationNanosecond"`
	Create             bool                `json:"create,omitempty"`
	Delete             bool                `json:"delete,omitempty"`
	Patch              json.RawMessage     `json:"patch,omitempty"`
}

// DeepCopyObject implements the runtime.Object interface.
func (r *ResourcePatch) DeepCopyObject() runtime.Object {
	if r == nil {
		return nil
	}
	t := *r
	return &t
}

// SetDelete sets the delete of the ResourcePatch.
func (r *ResourcePatch) SetDelete(obj metav1.Object, track map[log.ObjectRef]json.RawMessage) {
	r.Delete = true
	r.Patch = nil
	r.Create = false
	key := log.KObj(obj)
	delete(track, key)
}

// SetContent sets the content of the ResourcePatch.
func (r *ResourcePatch) SetContent(obj metav1.Object, track map[log.ObjectRef]json.RawMessage, patchMeta strategicpatch.LookupPatchMeta) error {
	clearUnstructured(obj)
	key := log.KObj(obj)

	modified, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	r.Delete = false

	original, ok := track[key]
	if !ok {
		track[key] = modified
		r.Create = true
		r.Patch = modified
		return nil
	}

	track[key] = modified
	patch, err := strategicpatch.CreateTwoWayMergePatchUsingLookupPatchMeta(original, modified, patchMeta)
	if err != nil {
		return err
	}

	r.Create = false
	r.Patch = patch
	return nil
}

// GetTargetGroupVersionResource returns the target group version resource of the ResourcePatch.
func (r *ResourcePatch) GetTargetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    r.Target.Type.Group,
		Version:  r.Target.Type.Version,
		Resource: r.Target.Type.Resource,
	}
}

// SetTargetGroupVersionResource sets the target group version resource of the ResourcePatch.
func (r *ResourcePatch) SetTargetGroupVersionResource(gvr schema.GroupVersionResource) {
	r.Target.Type = GroupVersionResource{
		Group:    gvr.Group,
		Version:  gvr.Version,
		Resource: gvr.Resource,
	}
}

// GetTargetName returns the target name of the ResourcePatch.
func (r *ResourcePatch) GetTargetName() (string, string) {
	return r.Target.Name, r.Target.Namespace
}

// SetTargetName sets the target name of the ResourcePatch.
func (r *ResourcePatch) SetTargetName(name, namespace string) {
	r.Target.Name = name
	r.Target.Namespace = namespace
}

// SetDuration sets the duration of the ResourcePatch.
func (r *ResourcePatch) SetDuration(dur time.Duration) {
	r.DurationNanosecond = dur
}

// GetDuration returns the duration of the ResourcePatch.
func (r *ResourcePatch) GetDuration() time.Duration {
	return r.DurationNanosecond
}

// ResourcePatchTarget is the target of the ResourcePatch.
type ResourcePatchTarget struct {
	Type      GroupVersionResource `json:"type"`
	Name      string               `json:"name"`
	Namespace string               `json:"namespace,omitempty"`
}

// GroupVersionResource is the group version resource.
type GroupVersionResource struct {
	Group    string `json:"group,omitempty"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
}

func clearUnstructured(obj metav1.Object) {
	obj.SetResourceVersion("")
	obj.SetSelfLink("")
	obj.SetManagedFields(nil)
}
