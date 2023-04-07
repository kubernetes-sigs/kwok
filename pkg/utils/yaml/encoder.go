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
	"io"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// Encoder is a YAML encoder.
type Encoder struct {
	printCount int64
	w          io.Writer
}

// NewEncoder returns a new YAML printer.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

var separator = []byte("---\n")

// Encode prints the object as YAML.
func (p *Encoder) Encode(obj runtime.Object) error {
	count := atomic.AddInt64(&p.printCount, 1)
	if count > 1 {
		if _, err := p.w.Write(separator); err != nil {
			return err
		}
	}

	output, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = p.w.Write(output)
	if err != nil {
		return err
	}

	return nil
}
