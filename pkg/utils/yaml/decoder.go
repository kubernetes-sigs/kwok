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

package yaml

import (
	"errors"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml" //nolint:depguard
)

// Decoder is a YAML decoder.
type Decoder struct {
	decoder *yaml.YAMLToJSONDecoder
}

// NewDecoder returns a new YAML decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		decoder: yaml.NewYAMLToJSONDecoder(r),
	}
}

// Decode decodes YAML into an object.
func (d *Decoder) Decode(obj any) error {
	return d.decoder.Decode(obj)
}

// DecodeToUnstructured decodes YAML into a list of unstructured objects.
func (d *Decoder) DecodeToUnstructured(visitFunc func(obj *unstructured.Unstructured) error) error {
	for {
		obj := &unstructured.Unstructured{}
		err := d.Decode(obj)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if obj.IsList() {
			err = obj.EachListItem(func(object runtime.Object) error {
				return visitFunc(object.(*unstructured.Unstructured))
			})
			if err != nil {
				return err
			}
		} else {
			err = visitFunc(obj)
			if err != nil {
				return err
			}
		}
	}
}
