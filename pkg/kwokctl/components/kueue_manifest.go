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

package components

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// BuildManifestConfig is the config for BuildKueueManifest.
type BuildManifestConfig struct {
	Port           uint32
	ExternalName   string
	VisibilityPort uint32
	CABundle       string
	RawManifest    string
}

// BuildKueueManifest transforms raw kueue manifest data with the provided
// configuration values.
func BuildKueueManifest(conf BuildManifestConfig) ([]string, error) {
	if len(conf.RawManifest) == 0 {
		return nil, fmt.Errorf("raw kueue manifest is empty")
	}

	transformers := []resourceTransformer{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Match: func(obj *unstructured.Unstructured) bool {
				strategy, _, _ := unstructured.NestedString(obj.Object, "spec", "conversion", "strategy")
				return strategy == "Webhook"
			},
			Transform: func(obj *unstructured.Unstructured) error {
				return transformCRDConversionWebhook(obj.Object, int64(conf.Port), conf.CABundle)
			},
		},
		{
			Kind:       "Service",
			APIVersion: "v1",
			Transform: func(obj *unstructured.Unstructured) error {
				return transformServiceToExternalName(obj.Object, conf.ExternalName)
			},
		},
		{
			Kind:       "APIService",
			APIVersion: "apiregistration.k8s.io/v1",
			Match: func(obj *unstructured.Unstructured) bool {
				group, _, _ := unstructured.NestedString(obj.Object, "spec", "group")
				return group == "visibility.kueue.x-k8s.io"
			},
			Transform: func(obj *unstructured.Unstructured) error {
				return transformAPIService(obj.Object, int64(conf.VisibilityPort))
			},
		},
		{
			Kind:       "APIService",
			APIVersion: "apiregistration.k8s.io/v1",
			Transform: func(obj *unstructured.Unstructured) error {
				return transformAPIService(obj.Object, int64(conf.Port))
			},
		},
		{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
			Transform: func(obj *unstructured.Unstructured) error {
				return transformWebhookClientConfigs(obj.Object, int64(conf.Port), conf.CABundle)
			},
		},
		{
			Kind:       "ValidatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
			Transform: func(obj *unstructured.Unstructured) error {
				return transformWebhookClientConfigs(obj.Object, int64(conf.Port), conf.CABundle)
			},
		},
		{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
			Delete:     true,
		},
	}

	result, err := rewriteManifest(conf.RawManifest, transformers)
	if err != nil {
		return nil, fmt.Errorf("failed to transform kueue manifest: %w", err)
	}
	return result, nil
}
