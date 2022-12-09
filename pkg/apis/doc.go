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

package apis

//go:generate go run k8s.io/code-generator/cmd/deepcopy-gen@latest -i ./v1alpha1/ --trim-path-prefix sigs.k8s.io/kwok/pkg/apis -o ./ -O zz_generated.deepcopy --go-header-file ../../hack/tools/boilerplate.go.txt
//go:generate go run k8s.io/code-generator/cmd/defaulter-gen@latest -i ./v1alpha1/ --trim-path-prefix sigs.k8s.io/kwok/pkg/apis -o ./ -O zz_generated.defaults --go-header-file ../../hack/tools/boilerplate.go.txt
//go:generate go run k8s.io/code-generator/cmd/deepcopy-gen@latest -i ./internalversion/ --trim-path-prefix sigs.k8s.io/kwok/pkg/apis -o ./ -O zz_generated.deepcopy --go-header-file ../../hack/tools/boilerplate.go.txt
//go:generate go run k8s.io/code-generator/cmd/conversion-gen@latest -i ./internalversion/ --trim-path-prefix sigs.k8s.io/kwok/pkg/apis -o ./ -O zz_generated.conversion --go-header-file ../../hack/tools/boilerplate.go.txt
