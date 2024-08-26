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

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	configv1alpha1 "sigs.k8s.io/kwok/pkg/apis/config/v1alpha1"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/patch"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

var (
	errUnsupportedType = errors.New("unsupported type")
)

func loadRawMessages(src []string) ([]json.RawMessage, error) {
	var raws []json.RawMessage

	for _, p := range src {
		if p == "-" {
			r, err := loadRaw(os.Stdin)
			if err != nil {
				return nil, err
			}
			raws = append(raws, r...)
			continue
		}
		p, err := path.Expand(p)
		if err != nil {
			return nil, err
		}
		r, err := loadRawMessage(p)
		if err != nil {
			return nil, err
		}
		raws = append(raws, r...)
	}
	return raws, nil
}

type versiondObject interface {
	GetObjectKind() schema.ObjectKind
	InternalObject
}

// InternalObject is an object that is internal to the kwok project.
type InternalObject interface {
	GetNamespace() string
	GetName() string
}

type configHandler struct {
	Unmarshal        func(raw []byte) (versiondObject, error)
	Marshal          func(obj versiondObject) ([]byte, error)
	MutateToInternal func(objs []versiondObject) ([]InternalObject, error)
	MutateToVersiond func(objs []InternalObject) ([]versiondObject, error)
}

var configHandlers = map[string]configHandler{
	configv1alpha1.KwokConfigurationKind: {
		Unmarshal:        unmarshalConfig[*configv1alpha1.KwokConfiguration],
		Marshal:          marshalConfig,
		MutateToInternal: mergeAndMutateToInternalConfig(convertToInternalKwokConfiguration),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1alpha1KwokConfiguration),
	},
	configv1alpha1.KwokctlConfigurationKind: {
		Unmarshal:        unmarshalConfig[*configv1alpha1.KwokctlConfiguration],
		Marshal:          marshalConfig,
		MutateToInternal: mergeAndMutateToInternalConfig(convertToInternalKwokctlConfiguration),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1alpha1KwokctlConfiguration),
	},
	configv1alpha1.KwokctlResourceKind: {
		Unmarshal:        unmarshalConfig[*configv1alpha1.KwokctlResource],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalKwokctlResource),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1alpha1KwokctlResource),
	},
	configv1alpha1.KwokctlComponentKind: {
		Unmarshal:        unmarshalConfig[*configv1alpha1.KwokctlComponent],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalKwokctlComponent),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1alpha1KwokctlComponent),
	},
	v1alpha1.StageKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.Stage],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(convertToInternalStage),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1alpha1Stage),
	},
	v1alpha1.PortForwardKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.PortForward],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalPortForward),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1PortForward),
	},
	v1alpha1.ClusterPortForwardKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.ClusterPortForward],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalClusterPortForward),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1ClusterPortForward),
	},
	v1alpha1.ExecKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.Exec],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalExec),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1Exec),
	},
	v1alpha1.ClusterExecKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.ClusterExec],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalClusterExec),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1ClusterExec),
	},
	v1alpha1.LogsKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.Logs],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalLogs),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1Logs),
	},
	v1alpha1.ClusterLogsKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.ClusterLogs],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalClusterLogs),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1ClusterLogs),
	},
	v1alpha1.AttachKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.Attach],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalAttach),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1Attach),
	},
	v1alpha1.ClusterAttachKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.ClusterAttach],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalClusterAttach),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1ClusterAttach),
	},
	v1alpha1.ResourceUsageKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.ResourceUsage],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalResourceUsage),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1ResourceUsage),
	},
	v1alpha1.ClusterResourceUsageKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.ClusterResourceUsage],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalClusterResourceUsage),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1ClusterResourceUsage),
	},
	v1alpha1.MetricKind: {
		Unmarshal:        unmarshalConfig[*v1alpha1.Metric],
		Marshal:          marshalConfig,
		MutateToInternal: mutateToInternalConfig(internalversion.ConvertToInternalMetric),
		MutateToVersiond: mutateToVersiondConfig(internalversion.ConvertToV1Alpha1Metric),
	},
}

func unmarshalConfig[T versiondObject](raw []byte) (versiondObject, error) {
	var obj T
	err := json.Unmarshal(raw, &obj)
	return obj, err
}

func marshalConfig(obj versiondObject) ([]byte, error) {
	return json.Marshal(obj)
}

func mergeAndMutateToInternalConfig[V versiondObject, I InternalObject](fun func(V) (I, error)) func(objs []versiondObject) ([]InternalObject, error) {
	return func(objs []versiondObject) ([]InternalObject, error) {
		if len(objs) == 0 {
			return []InternalObject{}, nil
		}

		var v V
		if len(objs) == 1 {
			obj, ok := objs[0].(V)
			if !ok {
				return nil, fmt.Errorf("unexpected type %T", objs[0])
			}
			v = obj
		} else {
			obj, ok := objs[0].(V)
			if !ok {
				return nil, fmt.Errorf("unexpected type %T", objs[0])
			}
			for _, o := range objs[1:] {
				a, ok := o.(V)
				if !ok {
					return nil, fmt.Errorf("unexpected type %T", o)
				}
				r, err := patch.StrategicMerge(obj, a)
				if err != nil {
					return nil, err
				}
				obj = r
			}
			v = obj
		}

		i, err := fun(v)
		if err != nil {
			return nil, err
		}
		return []InternalObject{i}, nil
	}
}

func mutateToInternalConfig[V versiondObject, I InternalObject](fun func(V) (I, error)) func(objs []versiondObject) ([]InternalObject, error) {
	return func(objs []versiondObject) ([]InternalObject, error) {
		var internalObjs []InternalObject
		for _, obj := range objs {
			v, ok := obj.(V)
			if !ok {
				return nil, fmt.Errorf("unexpected type %T", obj)
			}
			i, err := fun(v)
			if err != nil {
				return nil, err
			}
			internalObjs = append(internalObjs, i)
		}
		return internalObjs, nil
	}
}

func mutateToVersiondConfig[V versiondObject, I InternalObject](fun func(I) (V, error)) func(objs []InternalObject) ([]versiondObject, error) {
	return func(objs []InternalObject) ([]versiondObject, error) {
		var versiondObjs []versiondObject
		for _, obj := range objs {
			i, ok := obj.(I)
			if !ok {
				return nil, fmt.Errorf("unexpected type %T", obj)
			}
			v, err := fun(i)
			if err != nil {
				return nil, err
			}
			versiondObjs = append(versiondObjs, v)
		}
		return versiondObjs, nil
	}
}

// Load loads the given path into the context.
func Load(ctx context.Context, src ...string) ([]InternalObject, error) {
	raws, err := loadRawMessages(src)
	if err != nil {
		return nil, err
	}

	result := map[string][]versiondObject{}

	logger := log.FromContext(ctx)
	meta := metav1.TypeMeta{}
	for _, raw := range raws {
		err := json.Unmarshal(raw, &meta)
		if err != nil {
			logger.Error("Unsupported config", err,
				"src", src,
			)
			continue
		}

		gvk := meta.GroupVersionKind()

		handler, ok := configHandlers[gvk.Kind]
		if !ok {
			logger.Warn("Unsupported type",
				"apiVersion", meta.APIVersion,
				"kind", meta.Kind,
				"src", src,
			)
			continue
		}

		vobj, err := handler.Unmarshal(raw)
		if err != nil {
			return nil, err
		}
		result[gvk.Kind] = append(result[gvk.Kind], vobj)
	}

	kinds := maps.Keys(result)
	sort.Strings(kinds)
	objs := []InternalObject{}
	for _, kind := range kinds {
		handler, ok := configHandlers[kind]
		if !ok {
			logger.Warn("Unsupported type",
				"kind", kind,
				"src", src,
			)
			continue
		}
		versiondObjs := result[kind]
		internalObjs, err := handler.MutateToInternal(versiondObjs)
		if err != nil {
			return nil, err
		}
		objs = append(objs, internalObjs...)
	}

	return objs, nil
}

// LoadUnstructured loads the given path into the context.
func LoadUnstructured(src ...string) ([]InternalObject, error) {
	raws, err := loadRawMessages(src)
	if err != nil {
		return nil, err
	}

	objs := []InternalObject{}
	for _, raw := range raws {
		obj := unstructured.Unstructured{}
		err := json.Unmarshal(raw, &obj)
		if err != nil {
			return nil, err
		}

		objs = append(objs, &obj)
	}

	return objs, nil
}

// Save saves the given objects to the given path.
func Save(ctx context.Context, dist string, objs []InternalObject) error {
	dist = path.Clean(dist)
	err := file.MkdirAll(path.Dir(dist))
	if err != nil {
		return err
	}

	f, err := file.Open(dist)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		if err != nil {
			_ = file.Remove(dist)
		}
	}()
	return SaveTo(ctx, f, objs)
}

// SaveTo saves the given objects to the given writer.
func SaveTo(ctx context.Context, w io.Writer, objs []InternalObject) error {
	logger := log.FromContext(ctx)
	for i, obj := range objs {
		if i != 0 {
			_, err := w.Write([]byte("\n---\n"))
			if err != nil {
				return err
			}
		}

		data, err := Marshal(obj)
		if err != nil {
			if errors.Is(err, errUnsupportedType) {
				logger.Warn("Unsupported type", err,
					"obj", obj,
				)
				continue
			}
			return err
		}

		_, err = w.Write(data)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalWithType unmarshals the given raw message into the internal object.
func UnmarshalWithType[T InternalObject, D string | []byte](raw D) (t T, err error) {
	obj, err := Unmarshal(raw)
	if err != nil {
		return t, err
	}
	t, ok := obj.(T)
	if !ok {
		return t, fmt.Errorf("unexpected type %T %s", obj, log.KObj(obj))
	}
	return t, nil
}

// Unmarshal unmarshals the given raw message into the internal object.
func Unmarshal[D string | []byte](d D) (InternalObject, error) {
	meta := metav1.TypeMeta{}

	raw := []byte(d)

	raw, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, &meta)
	if err != nil {
		return nil, err
	}

	gvk := meta.GroupVersionKind()

	handler, ok := configHandlers[gvk.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errUnsupportedType, gvk.Kind)
	}

	vobj, err := handler.Unmarshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", gvk.Kind, err)
	}

	iobj, err := handler.MutateToInternal([]versiondObject{vobj})
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s: %w", gvk.Kind, err)
	}

	if len(iobj) == 0 {
		return nil, fmt.Errorf("failed to convert %s: no object", gvk.Kind)
	}

	if len(iobj) > 1 {
		return nil, fmt.Errorf("failed to convert %s: too many objects", gvk.Kind)
	}

	return iobj[0], nil
}

// Marshal marshals the given internal object into a raw message.
func Marshal(obj InternalObject) ([]byte, error) {
	typ := reflect.TypeOf(obj)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	kind := typ.Name()
	handler, ok := configHandlers[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errUnsupportedType, kind)
	}

	versiondObj, err := handler.MutateToVersiond([]InternalObject{obj})
	if err != nil {
		return nil, err
	}
	if len(versiondObj) != 1 {
		return nil, fmt.Errorf("unexpected length of versiond object: %d", len(versiondObj))
	}

	data, err := handler.Marshal(versiondObj[0])
	if err != nil {
		return nil, err
	}

	data, err = yaml.JSONToYAML(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// FilterWithType returns a list of objects with the given type.
func FilterWithType[T InternalObject](objs []InternalObject) (out []T) {
	for _, obj := range objs {
		o, ok := obj.(T)
		if ok {
			out = append(out, o)
		}
	}
	return out
}

// FilterWithoutType filters out objects of the given type.
func FilterWithoutType[T InternalObject](objs []InternalObject) (out []InternalObject) {
	for _, obj := range objs {
		_, ok := obj.(T)
		if !ok {
			out = append(out, obj)
		}
	}
	return out
}

// FilterWithTypeFromContext returns all objects of the given type from the context.
func FilterWithTypeFromContext[T metav1.Object](ctx context.Context) (out []T) {
	objs := GetFromContext(ctx)
	if len(objs) == 0 {
		return nil
	}
	return FilterWithType[T](objs)
}

// FilterWithoutTypeFromContext returns all objects from the context that are not of the given type.
func FilterWithoutTypeFromContext[T InternalObject](ctx context.Context) (out []InternalObject) {
	objs := GetFromContext(ctx)
	if len(objs) == 0 {
		return nil
	}
	return FilterWithoutType[T](objs)
}

func loadRawMessage(uri string) ([]json.RawMessage, error) {
	stat, err := os.Stat(uri)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return loadRawFromKustomize(uri)
	}
	return loadRawFromFile(uri)
}

func loadRawFromFile(p string) ([]json.RawMessage, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	return loadRaw(file)
}

func loadRawFromKustomize(dir string) ([]json.RawMessage, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	fs := filesys.MakeFsOnDisk()

	objs, err := k.Run(fs, dir)
	if err != nil {
		return nil, err
	}

	config, err := objs.AsYaml()
	if err != nil {
		return nil, err
	}

	return loadRaw(bytes.NewReader(config))
}

func loadRaw(r io.Reader) ([]json.RawMessage, error) {
	var raws []json.RawMessage
	decoder := yaml.NewDecoder(r)
	for {
		var raw json.RawMessage
		err := decoder.Decode(&raw)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to decode %q: %w", raw, err)
		}
		if len(raw) == 0 {
			// Ignore empty documents
			continue
		}
		raws = append(raws, raw)
	}
	return raws, nil
}
