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

package patch

import (
	"fmt"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/openapi"
	"k8s.io/client-go/openapi3"
	"k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"

	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

const (
	patchStrategyOpenapiextensionKey = "x-kubernetes-patch-strategy"
	patchMergeKeyOpenapiextensionKey = "x-kubernetes-patch-merge-key"
)

// PatchMetaFromOpenAPI3 is a PatchMetaFromOpenAPI3 implementation that uses openapi3.
type PatchMetaFromOpenAPI3 struct {
	root      openapi3.Root
	specCache map[schema.GroupVersionResource]*patchMeta
}

// NewPatchMetaFromOpenAPI3 creates a new PatchMetaFromOpenAPI3.
func NewPatchMetaFromOpenAPI3(s rest.Interface) *PatchMetaFromOpenAPI3 {
	openapiClient := openapi.NewClient(s)
	openapi3Root := openapi3.NewRoot(openapiClient)
	return &PatchMetaFromOpenAPI3{
		root:      openapi3Root,
		specCache: map[schema.GroupVersionResource]*patchMeta{},
	}
}

// Lookup returns the patch metadata for the given group-version-resource.
func (p *PatchMetaFromOpenAPI3) Lookup(gvr schema.GroupVersionResource) (strategicpatch.LookupPatchMeta, error) {
	lookmeta := p.specCache[gvr]
	if lookmeta != nil {
		return lookmeta, nil
	}

	gv := gvr.GroupVersion()
	spec, err := p.root.GVSpec(gv)
	if err != nil {
		return nil, fmt.Errorf("failed to get openapi spec: %w", err)
	}

	// Match the suffix like "/nodes/{name}" and exclude the watch path like "/watch/nodes/{name}"
	findResourceWithPath := "/" + gvr.Resource + "/{name}"
	paths := slices.Filter(maps.Keys(spec.Paths.Paths), func(s string) bool {
		return strings.HasSuffix(s, findResourceWithPath) &&
			!strings.Contains(s[:len(s)-len(findResourceWithPath)+1], "/watch/")
	})
	if len(paths) == 0 {
		return nil, fmt.Errorf("failed to find resource: %s", gvr.Resource)
	}
	if len(paths) > 1 {
		return nil, fmt.Errorf("found multiple resources: %s", paths)
	}
	key := paths[0]

	path := spec.Paths.Paths[key]
	if path.Get == nil {
		return nil, fmt.Errorf("failed to find get method: %s", key)
	}

	if path.Get.Responses.StatusCodeResponses == nil {
		return nil, fmt.Errorf("failed to find response: %s", key)
	}

	response := path.Get.Responses.StatusCodeResponses[http.StatusOK]
	if response == nil {
		return nil, fmt.Errorf("failed to find response 200: %s", key)
	}

	if response.Content == nil {
		return nil, fmt.Errorf("failed to find content: %s", key)
	}

	if response.Content["application/json"] == nil {
		return nil, fmt.Errorf("failed to find application/json: %s", key)
	}

	schema := response.Content["application/json"].Schema
	if schema == nil {
		return nil, fmt.Errorf("failed to find schema: %s", key)
	}

	lookmeta = &patchMeta{
		openapi: spec,
		gvr:     gvr,
		schema:  schema,
		name:    "",
	}

	err = lookmeta.forward(5)
	if err != nil {
		return nil, fmt.Errorf("failed to forward schema: %w", err)
	}

	p.specCache[gvr] = lookmeta
	return lookmeta, nil
}

type patchMeta struct {
	openapi *spec3.OpenAPI
	gvr     schema.GroupVersionResource
	schema  *spec.Schema
	name    string
}

func (p *patchMeta) forwardWithArray(ttl int) error {
	if p.schema.Items == nil {
		return fmt.Errorf("schema items not found")
	}

	p.schema = p.schema.Items.Schema
	return p.forward(ttl)
}

func (p *patchMeta) forward(ttl int) error {
	if len(p.schema.Type) != 0 {
		return nil
	}

	if ttl <= 0 {
		return fmt.Errorf("forwarding schema too deep")
	}

	switch {
	case p.schema.Ref.HasFragmentOnly:
		token := p.schema.Ref.GetPointer().DecodedTokens()
		p.schema = p.openapi.Components.Schemas[token[len(token)-1]]
	case len(p.schema.AllOf) != 0:
		p.schema = &p.schema.AllOf[0]
	case len(p.schema.OneOf) != 0:
		p.schema = &p.schema.OneOf[0]
	case len(p.schema.AnyOf) != 0:
		p.schema = &p.schema.AnyOf[0]
	default:
		return fmt.Errorf("failed to forward schema")
	}

	return p.forward(ttl - 1)
}

// LookupPatchMetadataForStruct gets subschema and the patch metadata (e.g. patch strategy and merge key) for map.
func (p *patchMeta) LookupPatchMetadataForStruct(key string) (strategicpatch.LookupPatchMeta, strategicpatch.PatchMeta, error) {
	if p.schema == nil {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("schema not found")
	}

	if p.schema.Properties == nil {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("schema properties not found: %s", p.gvr)
	}

	prop, ok := p.schema.Properties[key]
	if !ok {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("schema property not found: %s: %s", p.gvr, key)
	}

	meta, err := parsePatchMetadata(prop.Extensions)
	if err != nil {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("failed to parse patch metadata: %s: %w", p.gvr, err)
	}

	lookmeta := &patchMeta{
		openapi: p.openapi,
		gvr:     p.gvr,
		schema:  &prop,
		name:    key,
	}

	err = lookmeta.forward(5)
	if err != nil {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("failed to forward schema: %s: %w", p.gvr, err)
	}

	return lookmeta, meta, nil
}

// LookupPatchMetadataForSlice get subschema and the patch metadata for slice.
func (p *patchMeta) LookupPatchMetadataForSlice(key string) (strategicpatch.LookupPatchMeta, strategicpatch.PatchMeta, error) {
	if p.schema == nil {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("schema not found")
	}

	if p.schema.Properties == nil {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("schema properties not found: %s", p.gvr)
	}

	prop, ok := p.schema.Properties[key]
	if !ok {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("schema property not found: %s: %s", p.gvr, key)
	}
	if !prop.Type.Contains("array") {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("schema property is not array: %s: %s", p.gvr, key)
	}

	meta, err := parsePatchMetadata(prop.Extensions)
	if err != nil {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("failed to parse patch metadata: %s: %w", p.gvr, err)
	}

	lookmeta := &patchMeta{
		openapi: p.openapi,
		gvr:     p.gvr,
		schema:  &prop,
		name:    key,
	}

	err = lookmeta.forwardWithArray(5)
	if err != nil {
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("failed to forward schema: %s: %w", p.gvr, err)
	}

	return lookmeta, meta, nil
}

// Name returns the type name of the field
func (p *patchMeta) Name() string {
	return p.name
}

func parsePatchMetadata(extensions map[string]interface{}) (strategicpatch.PatchMeta, error) {
	ps, foundPS := extensions[patchStrategyOpenapiextensionKey]
	var patchStrategies []string
	var mergeKey, patchStrategy string
	var ok bool
	if foundPS {
		patchStrategy, ok = ps.(string)
		if ok {
			patchStrategies = strings.Split(patchStrategy, ",")
		} else {
			return strategicpatch.PatchMeta{}, mergepatch.ErrBadArgType(patchStrategy, ps)
		}
	}
	mk, foundMK := extensions[patchMergeKeyOpenapiextensionKey]
	if foundMK {
		mergeKey, ok = mk.(string)
		if !ok {
			return strategicpatch.PatchMeta{}, mergepatch.ErrBadArgType(mergeKey, mk)
		}
	}
	var meta strategicpatch.PatchMeta
	if len(patchStrategies) != 0 {
		// Avoid duplicate values being ignored, e.g. heartbeat on condition
		patchStrategies = slices.Filter(patchStrategies, func(s string) bool {
			return s != "retainKeys"
		})

		meta.SetPatchStrategies(patchStrategies)
	}
	if mergeKey != "" {
		meta.SetPatchMergeKey(mergeKey)
	}
	return meta, nil
}
