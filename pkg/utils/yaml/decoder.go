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

	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// Decoder is a YAML decoder.
type Decoder struct {
	decoder      *yaml.YAMLToJSONDecoder
	errorHandler func(error) error
	buf          []*unstructured.Unstructured
}

// WithErrorHandler sets the error handler for the decoder.
func (d *Decoder) WithErrorHandler(handler func(error) error) *Decoder {
	d.errorHandler = handler
	return d
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

// UndecodedUnstructured put a decoded unstructured object back to the decoder.
func (d *Decoder) UndecodedUnstructured(obj *unstructured.Unstructured) {
	d.buf = append(d.buf, obj)
}

// DecodeUnstructured decodes YAML into an unstructured object.
func (d *Decoder) DecodeUnstructured() (*unstructured.Unstructured, error) {
	if len(d.buf) != 0 {
		last := len(d.buf) - 1
		obj := d.buf[last]
		d.buf = d.buf[:last]
		return obj, nil
	}

	obj := &unstructured.Unstructured{}
	err := d.Decode(obj)
	if err != nil {
		return nil, err
	}

	if obj.IsList() {
		_ = obj.EachListItem(func(object runtime.Object) error {
			obj := object.(*unstructured.Unstructured)
			if len(obj.Object) == 0 {
				return nil
			}
			d.buf = append(d.buf, object.(*unstructured.Unstructured))
			return nil
		})
		// Reverse the slice to keep the order.
		slices.Reverse(d.buf)
		return d.DecodeUnstructured()
	}

	// If the object is empty, we should continue to decode the next object.
	if len(obj.Object) == 0 {
		return d.DecodeUnstructured()
	}

	return obj, nil
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
			if d.errorHandler != nil {
				err = d.errorHandler(err)
				if err == nil {
					continue
				}
			}
			return err
		}

		if obj.IsList() {
			err = obj.EachListItem(func(object runtime.Object) error {
				obj := object.(*unstructured.Unstructured)
				if len(obj.Object) == 0 {
					return nil
				}
				return visitFunc(object.(*unstructured.Unstructured))
			})
			if err != nil {
				return err
			}
		} else {
			if len(obj.Object) == 0 {
				continue
			}
			err = visitFunc(obj)
			if err != nil {
				return err
			}
		}
	}
}
