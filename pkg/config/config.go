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
	"io"
	"os"
	"path/filepath"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/config/compatibility"
	"sigs.k8s.io/kwok/pkg/log"
)

func Load(ctx context.Context, path string) ([]metav1.Object, error) {
	raws, err := loadRawConfig(path)
	if err != nil {
		return nil, err
	}

	logger := log.FromContext(ctx)
	meta := metav1.TypeMeta{}
	objs := make([]metav1.Object, 0, len(raws))
	for _, raw := range raws {
		err = json.Unmarshal(raw, &meta)
		if err != nil {
			logger.Error("Unsupported config", err,
				"path", path,
			)
			continue
		}

		gvk := meta.GroupVersionKind()

		// Converting old configurations to the latest
		if gvk.Version == "" && gvk.Group == "" && gvk.Kind == "" {
			conf := compatibility.Config{}
			err = json.Unmarshal(raw, &conf)
			if err != nil {
				logger.Error("Unsupported config", err,
					"path", path,
				)
				continue
			}
			obj, ok := compatibility.Convert_Config_To_v1alpha1_KwokctlConfiguration(&conf)
			if ok {
				logger.Debug("Convert old config",
					"path", path,
				)
				objs = append(objs, obj)
				continue
			}
		}

		if gvk.Version != v1alpha1.GroupVersion.Version ||
			gvk.Group != v1alpha1.GroupVersion.Group {
			logger.Warn("Unsupported type",
				"apiVersion", meta.APIVersion,
				"kind", meta.Kind,
				"path", path,
			)
			continue
		}
		switch gvk.Kind {
		default:
			logger.Warn("Unsupported type",
				"apiVersion", meta.APIVersion,
				"kind", meta.Kind,
				"path", path,
			)
		case v1alpha1.KwokConfigurationKind:
			obj := &v1alpha1.KwokConfiguration{}
			err = json.Unmarshal(raw, &obj)
			if err != nil {
				return nil, err
			}
			obj = setKwokConfigurationDefaults(obj)
			out, err := internalversion.ConvertToInternalVersionKwokConfiguration(obj)
			if err != nil {
				return nil, err
			}
			objs = append(objs, out)
		case v1alpha1.KwokctlConfigurationKind:
			obj := &v1alpha1.KwokctlConfiguration{}
			err = json.Unmarshal(raw, &obj)
			if err != nil {
				return nil, err
			}
			obj = setKwokctlConfigurationDefaults(obj)
			out, err := internalversion.ConvertToInternalVersionKwokctlConfiguration(obj)
			if err != nil {
				return nil, err
			}
			objs = append(objs, out)
		}
	}
	return objs, nil
}

func Save(ctx context.Context, path string, objs []metav1.Object) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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

		switch o := obj.(type) {
		default:
			logger.Warn("Unsupported type",
				"type", reflect.TypeOf(obj).String(),
			)
			continue
		case *internalversion.KwokConfiguration:
			obj, err = internalversion.ConvertToV1alpha1KwokConfiguration(o)
			if err != nil {
				return err
			}
		case *internalversion.KwokctlConfiguration:
			obj, err = internalversion.ConvertToV1alpha1KwokctlConfiguration(o)
			if err != nil {
				return err
			}
		}
		data, err := yaml.Marshal(obj)
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

func FilterWithType[T metav1.Object](objs []metav1.Object) (out []T) {
	for _, obj := range objs {
		o, ok := obj.(T)
		if ok {
			out = append(out, o)
		}
	}
	return out
}

func FilterWithoutType[T metav1.Object](objs []metav1.Object) (out []metav1.Object) {
	for _, obj := range objs {
		_, ok := obj.(T)
		if !ok {
			out = append(out, obj)
		}
	}
	return out
}

func FilterWithTypeFromContext[T metav1.Object](ctx context.Context) (out []T) {
	v := ctx.Value(configCtx(0))
	objs, ok := v.([]metav1.Object)
	if !ok {
		return nil
	}
	return FilterWithType[T](objs)
}

func FilterWithoutTypeFromContext[T metav1.Object](ctx context.Context) (out []metav1.Object) {
	v := ctx.Value(configCtx(0))
	objs, ok := v.([]metav1.Object)
	if !ok {
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
		raws = append(raws, raw)
	}
	return raws, nil
}
