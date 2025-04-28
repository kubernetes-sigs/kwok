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

package etcd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

const (
	// StorageBinaryMediaType is the media type for Kubernetes storage binary format
	StorageBinaryMediaType = "application/vnd.kubernetes.storagebinary"
	// ProtobufMediaType is the media type for Protocol Buffers format
	ProtobufMediaType = "application/vnd.kubernetes.protobuf"
	// YAMLMediaType is the media type for YAML format
	YAMLMediaType = "application/yaml"
	// JSONMediaType is the media type for JSON format
	JSONMediaType = "application/json"
)

// Convert converts the given data from one media type to another.
func Convert(inMediaType, outMediaType string, in []byte) (*runtime.TypeMeta, []byte, error) {
	if inMediaType == ProtobufMediaType && outMediaType == StorageBinaryMediaType {
		return nil, nil, fmt.Errorf("cannot convert from %s to %s", inMediaType, outMediaType)
	}

	if inMediaType == StorageBinaryMediaType && outMediaType == ProtobufMediaType {
		d, err := decodeUnknown(in)
		if err != nil {
			return nil, nil, fmt.Errorf("error decoding from %s: %w", inMediaType, err)
		}
		return &d.TypeMeta, d.Raw, nil
	}

	typeMeta, err := decodeTypeMeta(inMediaType, in)
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding from %s: %w", inMediaType, err)
	}

	if inMediaType == outMediaType {
		return typeMeta, in, nil
	}

	if inMediaType == JSONMediaType && outMediaType == YAMLMediaType {
		d, err := yaml.JSONToYAML(in)
		if err != nil {
			return nil, nil, fmt.Errorf("error decoding from %s: %w", inMediaType, err)
		}
		return typeMeta, d, nil
	}

	if inMediaType == YAMLMediaType && outMediaType == JSONMediaType {
		d, err := yaml.YAMLToJSON(in)
		if err != nil {
			return nil, nil, fmt.Errorf("error decoding from %s: %w", inMediaType, err)
		}
		return typeMeta, d, nil
	}

	gv, err := schema.ParseGroupVersion(typeMeta.APIVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse group version %s: %w", typeMeta.APIVersion, err)
	}

	inCodec, err := newCodec(gv, inMediaType)
	if err != nil {
		return nil, nil, err
	}

	outCodec, err := newCodec(gv, outMediaType)
	if err != nil {
		return nil, nil, err
	}

	obj, err := runtime.Decode(inCodec, in)
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding from %s: %w", inMediaType, err)
	}

	out, err := runtime.Encode(outCodec, obj)
	if err != nil {
		return nil, nil, fmt.Errorf("error encoding to %s: %w", outMediaType, err)
	}

	return typeMeta, out, nil
}

// DetectMediaType detects the media type of the given data.
func DetectMediaType(in []byte) (string, error) {
	if ok := isProto(in); ok {
		return StorageBinaryMediaType, nil
	}
	if ok := isJSONObject(in); ok {
		return JSONMediaType, nil
	}
	return "", fmt.Errorf("unable to detect media type")
}

func isProto(in []byte) bool {
	return bytes.HasPrefix(in, protoEncodingPrefix)
}

func isJSONObject(in []byte) bool {
	if len(in) == 0 {
		return false
	}
	return in[0] == '{'
}

func newCodec(gv schema.GroupVersion, mediaType string) (runtime.Codec, error) {
	if mediaType == StorageBinaryMediaType {
		mediaType = ProtobufMediaType
	}

	mediaTypes := Codecs.SupportedMediaTypes()
	info, ok := runtime.SerializerInfoForMediaType(mediaTypes, mediaType)
	if !ok {
		if len(mediaTypes) == 0 {
			return nil, fmt.Errorf("no serializers registered for %v", mediaTypes)
		}
		info = mediaTypes[0]
	}

	encoder := Codecs.EncoderForVersion(info.Serializer, gv)
	decoder := Codecs.DecoderToVersion(info.Serializer, gv)
	codec := Codecs.CodecForVersions(encoder, decoder, gv, gv)
	return codec, nil
}

func decodeTypeMeta(mediaType string, in []byte) (*runtime.TypeMeta, error) {
	switch mediaType {
	case JSONMediaType:
		return decodeTypeMetaFromJson(in)
	case StorageBinaryMediaType:
		return decodeTypeMetaFromBinaryStorage(in)
	case YAMLMediaType:
		return decodeTypeMetaFromYaml(in)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

func decodeTypeMetaFromJson(in []byte) (*runtime.TypeMeta, error) {
	var meta runtime.TypeMeta
	err := json.Unmarshal(in, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func decodeTypeMetaFromYaml(in []byte) (*runtime.TypeMeta, error) {
	var meta runtime.TypeMeta
	err := yaml.Unmarshal(in, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func decodeTypeMetaFromBinaryStorage(in []byte) (*runtime.TypeMeta, error) {
	unknown, err := decodeUnknown(in)
	if err != nil {
		return nil, err
	}
	return &unknown.TypeMeta, nil
}

// see k8s.io/apimachinery/pkg/runtime/generated.proto for details of the runtime.Unknown message.
var protoEncodingPrefix = []byte("k8s\x00") // []byte{0x6b, 0x38, 0x73, 0x00}

func decodeUnknown(in []byte) (*runtime.Unknown, error) {
	if !bytes.HasPrefix(in, protoEncodingPrefix) {
		return nil, fmt.Errorf("invalid storage data")
	}
	unknown := &runtime.Unknown{}
	if err := unknown.Unmarshal(in[4:]); err != nil {
		return nil, fmt.Errorf("error decoding to %s: %w", StorageBinaryMediaType, err)
	}
	return unknown, nil
}

// specialDefaultResourcePrefixes are prefixes compiled into Kubernetes.
// see k8s.io/kubernetes/pkg/kubeapiserver/default_storage_factory_builder.go
var specialDefaultResourcePrefixes = map[schema.GroupResource]string{
	{Group: "", Resource: "replicationcontrollers"}:     "controllers",
	{Group: "", Resource: "endpoints"}:                  "services/endpoints",
	{Group: "", Resource: "services"}:                   "services/specs",
	{Group: "", Resource: "nodes"}:                      "minions",
	{Group: "extensions", Resource: "ingresses"}:        "ingress",
	{Group: "networking.k8s.io", Resource: "ingresses"}: "ingress",
}

var specialDefaultMediaTypes = map[string]struct{}{
	"apiextensions.k8s.io":   {},
	"apiregistration.k8s.io": {},
}

// PrefixFromGVR returns the prefix of the given GroupVersionResource.
func PrefixFromGVR(gvr schema.GroupVersionResource) (prefix string, err error) {
	groupPrefix := false

	if _, ok := specialDefaultMediaTypes[gvr.Group]; ok {
		groupPrefix = true
	} else if !strings.Contains(gvr.Group, ".") || strings.HasSuffix(gvr.Group, ".k8s.io") {
		groupPrefix = false
	} else {
		groupPrefix = true
	}

	if prefix, ok := specialDefaultResourcePrefixes[gvr.GroupResource()]; ok {
		return prefix, nil
	}

	if groupPrefix {
		return gvr.Group + "/" + gvr.Resource, nil
	}

	return gvr.Resource, nil
}

// MediaTypeFromGVR returns the media type of the given GroupVersionResource.
func MediaTypeFromGVR(gvr schema.GroupVersionResource) (mediaType string, err error) {
	mediaType = JSONMediaType

	if _, ok := specialDefaultMediaTypes[gvr.Group]; ok {
		return mediaType, nil
	}

	if !strings.Contains(gvr.Group, ".") || strings.HasSuffix(gvr.Group, ".k8s.io") {
		if _, err := newCodec(gvr.GroupVersion(), StorageBinaryMediaType); err == nil {
			return StorageBinaryMediaType, nil
		}
	}

	return mediaType, nil
}
