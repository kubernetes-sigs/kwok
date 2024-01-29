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

package recording

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"sigs.k8s.io/kwok/pkg/apis/action/v1alpha1"
	"sigs.k8s.io/kwok/pkg/log"
)

// ResourcePatchType is the type of the ResourcePatch.
var ResourcePatchType = metav1.TypeMeta{
	Kind:       v1alpha1.ResourcePatchKind,
	APIVersion: v1alpha1.GroupVersion.String(),
}

// ResourcePatch is the patch of the resource.
type ResourcePatch v1alpha1.ResourcePatch

// DeepCopyObject implements the runtime.Object interface.
func (r *ResourcePatch) DeepCopyObject() runtime.Object {
	if r == nil {
		return nil
	}

	p := (*v1alpha1.ResourcePatch)(r)
	return (*ResourcePatch)(p.DeepCopy())
}

// SetDelete sets the delete of the ResourcePatch.
func (r *ResourcePatch) SetDelete(obj metav1.Object, track map[log.ObjectRef]json.RawMessage) {
	r.Method = v1alpha1.PatchMethodDelete
	r.Template = nil
	key := log.KObj(obj)
	delete(track, key)
}

// SetContent sets the content of the ResourcePatch.
func (r *ResourcePatch) SetContent(obj metav1.Object, track map[log.ObjectRef]json.RawMessage, patchMeta strategicpatch.LookupPatchMeta) error {
	key := log.KObj(obj)

	obj.SetResourceVersion("")
	modified, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	original, ok := track[key]
	if !ok {
		track[key] = modified
		r.Method = v1alpha1.PatchMethodCreate
		r.Template = modified
		return nil
	}

	track[key] = modified
	patch, err := strategicpatch.CreateTwoWayMergePatchUsingLookupPatchMeta(original, modified, patchMeta)
	if err != nil {
		return err
	}

	r.Method = v1alpha1.PatchMethodPatch
	r.Template = patch
	return nil
}

// GetTargetGroupVersionResource returns the target group version resource of the ResourcePatch.
func (r *ResourcePatch) GetTargetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    r.Resource.Group,
		Version:  r.Resource.Version,
		Resource: r.Resource.Resource,
	}
}

// SetTargetGroupVersionResource sets the target group version resource of the ResourcePatch.
func (r *ResourcePatch) SetTargetGroupVersionResource(gvr schema.GroupVersionResource) {
	r.Resource.Group = gvr.Group
	r.Resource.Version = gvr.Version
	r.Resource.Resource = gvr.Resource
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
