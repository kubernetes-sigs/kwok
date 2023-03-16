//go:build !linux

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

package cni

import (
	"context"
	"fmt"
)

// Setup is stubbed out on non-Linux platforms.
func Setup(ctx context.Context, id, name, namespace string) (ip []string, err error) {
	return nil, fmt.Errorf("unsupported")
}

// Remove is stubbed out on non-Linux platforms.
func Remove(ctx context.Context, id, name, namespace string) (err error) {
	return fmt.Errorf("unsupported")
}
