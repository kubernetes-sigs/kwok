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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	configv1alpha1 "sigs.k8s.io/kwok/pkg/apis/config/v1alpha1"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/config/compatibility"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/patch"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

func loadRawMessages(src []string) ([]json.RawMessage, error) {
	var raws []json.RawMessage

	for _, p := range src {
		p, err := path.Expand(p)
		if err != nil {
			return nil, err
		}
		r, err := loadRawConfig(p)
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

		// Converting old configurations to the latest
		// TODO: Remove this in the future
		if gvk.Version == "" && gvk.Group == "" && gvk.Kind == "" {
			conf := compatibility.Config{}
			err = json.Unmarshal(raw, &conf)
			if err != nil {
				logger.Error("Unsupported config", err,
					"src", src,
				)
				continue
			}
			obj, ok := compatibility.Convert_Config_To_internalversion_KwokctlConfiguration(&conf)
			if ok {
				logger.Debug("Convert old config",
					"src", src,
				)
				return []InternalObject{obj}, nil
			}
		}

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

// Save saves the given objects to the given path.
func Save(ctx context.Context, path string, objs []InternalObject) error {
	err := os.MkdirAll(filepath.Dir(path), 0750)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = os.Remove(path)
		}
	}()

	logger := log.FromContext(ctx)
	for i, obj := range objs {
		if i != 0 {
			_, err = file.WriteString("\n---\n")
			if err != nil {
				return err
			}
		}

		typ := reflect.TypeOf(obj)
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		kind := typ.Name()
		handler, ok := configHandlers[kind]
		if !ok {
			logger.Warn("Unsupported type",
				"type", kind,
			)
			continue
		}

		versiondObj, err := handler.MutateToVersiond([]InternalObject{obj})
		if err != nil {
			return err
		}
		if len(versiondObj) != 1 {
			return fmt.Errorf("unexpected length of versiond object: %d", len(versiondObj))
		}

		data, err := handler.Marshal(versiondObj[0])
		if err != nil {
			return err
		}

		data, err = yaml.JSONToYAML(data)
		if err != nil {
			return err
		}

		_, err = file.Write(data)
		if err != nil {
			return err
		}
	}
	return nil
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
	objs := getFromContext(ctx)
	if len(objs) == 0 {
		return nil
	}
	return FilterWithType[T](objs)
}

// FilterWithoutTypeFromContext returns all objects from the context that are not of the given type.
func FilterWithoutTypeFromContext[T InternalObject](ctx context.Context) (out []InternalObject) {
	objs := getFromContext(ctx)
	if len(objs) == 0 {
		return nil
	}
	return FilterWithoutType[T](objs)
}

func loadRawConfig(p string) ([]json.RawMessage, error) {
	var raws []json.RawMessage
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	decoder := utilyaml.NewYAMLToJSONDecoder(file)
	for {
		var raw json.RawMessage
		err = decoder.Decode(&raw)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if len(raw) == 0 {
			// Ignore empty documents
			continue
		}
		raws = append(raws, raw)
	}
	return raws, nil
}
