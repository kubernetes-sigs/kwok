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

package client

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// MappingFor returns the RESTMapping for the given resource or kind argument.
func MappingFor(restMapper meta.RESTMapper, resourceOrKindArg string) (*meta.RESTMapping, error) {
	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(resourceOrKindArg)
	gvk := schema.GroupVersionKind{}

	if fullySpecifiedGVR != nil {
		gvk, _ = restMapper.KindFor(*fullySpecifiedGVR)
	}
	if gvk.Empty() {
		gvk, _ = restMapper.KindFor(groupResource.WithVersion(""))
	}
	if !gvk.Empty() {
		return restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	}

	fullySpecifiedGVK, groupKind := schema.ParseKindArg(resourceOrKindArg)
	if fullySpecifiedGVK == nil {
		gvk := groupKind.WithVersion("")
		fullySpecifiedGVK = &gvk
	}

	if !fullySpecifiedGVK.Empty() {
		if mapping, err := restMapper.RESTMapping(fullySpecifiedGVK.GroupKind(), fullySpecifiedGVK.Version); err == nil {
			return mapping, nil
		}
	}

	mapping, err := restMapper.RESTMapping(groupKind, gvk.Version)
	if err != nil {
		// if we error out here, it is because we could not match a resource or a kind
		// for the given argument. To maintain consistency with previous behavior,
		// announce that a resource type could not be found.
		// if the error is _not_ a *meta.NoKindMatchError, then we had trouble doing discovery,
		// so we should return the original error since it may help a user diagnose what is actually wrong
		if meta.IsNoMatchError(err) {
			return nil, fmt.Errorf("the server doesn't have a resource type %q", groupResource.Resource)
		}
		return nil, err
	}

	return mapping, nil
}

// MatchShortResourceName returns the GroupVersionResource for the given resource name.
func MatchShortResourceName(arl []*metav1.APIResourceList, name string) (gvr schema.GroupVersionResource, resource *metav1.APIResource, err error) {
	name = strings.ToLower(name)

	for _, r := range arl {
		ar, ok := slices.Find(r.APIResources, func(ar metav1.APIResource) bool {
			return ar.Name == name ||
				ar.SingularName == name ||
				slices.Contains(ar.ShortNames, name)
		})
		if ok {
			gv, err := schema.ParseGroupVersion(r.GroupVersion)
			if err != nil {
				return gvr, nil, err
			}
			gvr = schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: ar.Name,
			}
			return gvr, &ar, nil
		}
	}

	return gvr, nil, fmt.Errorf("no resource found for %q", name)
}

// MatchGVK returns the GroupVersionKind for the given resource name.
func MatchGVK(arl []*metav1.APIResourceList, gvk schema.GroupVersionKind) (gvr schema.GroupVersionResource, resource *metav1.APIResource, err error) {
	gvStr := gvk.GroupVersion().String()
	for _, r := range arl {
		if gvStr != r.GroupVersion {
			continue
		}
		ar, ok := slices.Find(r.APIResources, func(ar metav1.APIResource) bool {
			return ar.Kind == gvk.Kind
		})
		if ok {
			gvr = schema.GroupVersionResource{
				Group:    gvk.Group,
				Version:  gvk.Version,
				Resource: ar.Name,
			}
			return gvr, &ar, nil
		}
	}

	return gvr, nil, fmt.Errorf("no resource found for %q", gvk)
}

// MatchGK returns the GroupVersionKind for the given resource name.
func MatchGK(arl []*metav1.APIResourceList, gvk schema.GroupKind) (gvr schema.GroupVersionResource, resource *metav1.APIResource, err error) {
	for _, r := range arl {
		gv, err := schema.ParseGroupVersion(r.GroupVersion)
		if err != nil {
			return gvr, nil, err
		}
		if gv.Group != gvk.Group {
			continue
		}
		ar, ok := slices.Find(r.APIResources, func(ar metav1.APIResource) bool {
			return ar.Kind == gvk.Kind
		})
		if ok {
			gvr = schema.GroupVersionResource{
				Group:    gvk.Group,
				Version:  gv.Version,
				Resource: ar.Name,
			}
			return gvr, &ar, nil
		}
	}

	return gvr, nil, fmt.Errorf("no resource found for %q", gvk)
}

// MappingForResources is a wrapper of MappingFor.
func MappingForResources(restMapper meta.RESTMapper, filters []string) ([]*meta.RESTMapping, []error) {
	mappings := make([]*meta.RESTMapping, 0, len(filters))
	errs := make([]error, 0, len(filters))
	for _, filter := range filters {
		mapping, err := MappingFor(restMapper, filter)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		mappings = append(mappings, mapping)
	}
	return mappings, errs
}
