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

	yamlv3 "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// Decoder is a YAML decoder.
type Decoder struct {
	decoder *yamlv3.Decoder
}

// NewDecoder returns a new YAML decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		decoder: yamlv3.NewDecoder(r),
	}
}

// Decode decodes YAML into a list of unstructured objects.
func (d *Decoder) Decode(visitFunc func(obj *unstructured.Unstructured) error) error {
	var tmp map[string]interface{}
	for {
		err := d.decoder.Decode(&tmp)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		data, err := yamlv3.Marshal(tmp)
		if err != nil {
			return err
		}
		data, err = yaml.YAMLToJSON(data)
		if err != nil {
			return err
		}
		obj := &unstructured.Unstructured{}
		err = obj.UnmarshalJSON(data)
		if err != nil {
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
