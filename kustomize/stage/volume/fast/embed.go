/*
Copyright 2026 The Kubernetes Authors.

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

// Package fast contains the fast volume for kwok.
package fast

import (
	_ "embed"
)

var (
	// DefaultPVBind is the default persistent volume bind yaml.
	//go:embed pv-bind.yaml
	DefaultPVBind string

	// DefaultPVDelete is the default persistent volume delete yaml.
	//go:embed pv-delete.yaml
	DefaultPVDelete string

	// DefaultPVCDelete is the default persistent volume claim delete yaml.
	//go:embed pvc-delete.yaml
	DefaultPVCDelete string

	// DefaultPVCProvision is the default persistent volume claim provision yaml.
	//go:embed pvc-provision.yaml
	DefaultPVCProvision string
)
