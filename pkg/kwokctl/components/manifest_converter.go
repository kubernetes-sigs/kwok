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
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	utilyaml "sigs.k8s.io/kwok/pkg/utils/yaml"
)

// resourceTransformer describes how to identify and transform a specific
// Kubernetes resource within a multi-document manifest.
type resourceTransformer struct {
	// Kind is the resource kind to match (e.g. "Service").
	Kind string
	// APIVersion is the resource apiVersion to match (e.g. "apps/v1").
	// When empty, any apiVersion is accepted.
	APIVersion string
	// Match returns true if this transformer should be applied to the given
	// parsed resource object. It is called only when Kind already matches.
	Match func(obj *unstructured.Unstructured) bool
	// Transform modifies obj in-place to apply the desired substitutions.
	// Ignored when Delete is true.
	Transform func(obj *unstructured.Unstructured) error
	// Delete, when true, causes the matched document to be omitted from output.
	Delete bool
}

// rewriteManifest rewrites a plain Kubernetes multi-document YAML manifest by
// applying transformers to matching resources and leaving all other documents
// untouched.
func rewriteManifest(rawManifest string, transformers []resourceTransformer) ([]string, error) {
	reader := utilyaml.NewDecoder(bufio.NewReader(strings.NewReader(rawManifest)))

	var result []string
	for {
		obj, err := reader.DecodeUnstructured()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read yaml document: %w", err)
		}

		matchedIndex := matchTransformer(obj, transformers)
		if matchedIndex == -1 {
			return nil, fmt.Errorf("no transformer found for resource %s", obj.GroupVersionKind())
		}

		matched := transformers[matchedIndex]
		if matched.Delete {
			continue
		}

		if matched.Transform != nil {
			if err := matched.Transform(obj); err != nil {
				return nil, fmt.Errorf("%s transform: %w", obj.GroupVersionKind(), err)
			}
		}

		doc, err := utilyaml.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal yaml document: %w", err)
		}
		result = append(result, string(doc))
	}

	return result, nil
}

func matchTransformer(obj *unstructured.Unstructured, transformers []resourceTransformer) int {
	kind := obj.GetKind()
	apiVersion := obj.GetAPIVersion()

	for i, t := range transformers {
		if t.Kind != kind {
			continue
		}

		if t.APIVersion != apiVersion {
			continue
		}

		if t.Match == nil || t.Match(obj) {
			return i
		}
	}

	return -1
}

// transformCRDConversionWebhook sets port and caBundle on the conversion webhook
// clientConfig of a CRD.
func transformCRDConversionWebhook(obj map[string]any, port int64, caBundle string) error {
	clientConfig, found, err := unstructured.NestedMap(obj, "spec", "conversion", "webhook", "clientConfig")
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("spec.conversion.webhook.clientConfig not found")
	}
	if _, ok := clientConfig["service"].(map[string]any); !ok {
		return fmt.Errorf("spec.conversion.webhook.clientConfig.service is not a map")
	}
	if err := unstructured.SetNestedField(obj, caBundle, "spec", "conversion", "webhook", "clientConfig", "caBundle"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(obj, port, "spec", "conversion", "webhook", "clientConfig", "service", "port"); err != nil {
		return err
	}
	return nil
}

// transformServiceToExternalName converts a Service spec to ExternalName type.
func transformServiceToExternalName(obj map[string]any, externalName string) error {
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return fmt.Errorf("spec is not a map")
	}
	delete(spec, "ports")
	delete(spec, "selector")
	spec["type"] = "ExternalName"
	spec["externalName"] = externalName
	return nil
}

// transformWebhookClientConfigs sets port and caBundle on every webhook
// entry's clientConfig.service.
func transformWebhookClientConfigs(obj map[string]any, port int64, caBundle string) error {
	webhooks, ok := obj["webhooks"].([]any)
	if !ok {
		return nil
	}
	for i, wh := range webhooks {
		whMap, ok := wh.(map[string]any)
		if !ok {
			return fmt.Errorf("webhook[%d] is not a map", i)
		}
		clientConfig, found, err := unstructured.NestedMap(whMap, "clientConfig")
		if err != nil {
			return fmt.Errorf("webhook[%d].clientConfig: %w", i, err)
		}
		if !found {
			return fmt.Errorf("webhook[%d].clientConfig not found", i)
		}
		if _, ok := clientConfig["service"].(map[string]any); !ok {
			return fmt.Errorf("webhook[%d].clientConfig.service is not a map", i)
		}
		if err := unstructured.SetNestedField(whMap, caBundle, "clientConfig", "caBundle"); err != nil {
			return fmt.Errorf("webhook[%d].clientConfig.caBundle: %w", i, err)
		}
		if err := unstructured.SetNestedField(whMap, port, "clientConfig", "service", "port"); err != nil {
			return fmt.Errorf("webhook[%d].clientConfig.service.port: %w", i, err)
		}
	}
	return nil
}

// transformAPIService sets port on the service reference of an APIService.
func transformAPIService(obj map[string]any, port int64) error {
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return fmt.Errorf("spec is not a map")
	}
	_, found, err := unstructured.NestedMap(spec, "service")
	if err != nil {
		return fmt.Errorf("spec.service: %w", err)
	}
	if !found {
		return fmt.Errorf("spec.service not found")
	}
	if err := unstructured.SetNestedField(obj, port, "spec", "service", "port"); err != nil {
		return fmt.Errorf("spec.service.port: %w", err)
	}
	return nil
}

var defaultTransformers = []resourceTransformer{
	{
		Kind:       "Namespace",
		APIVersion: "v1",
	},
	{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Delete:     true,
	},
	{
		Kind:       "ConfigMap",
		APIVersion: "v1",
		Delete:     true,
	},
	{
		Kind:       "Secret",
		APIVersion: "v1",
		Delete:     true,
	},
	{
		Kind:       "ServiceAccount",
		APIVersion: "v1",
		Delete:     true,
	},
	{
		Kind:       "Role",
		APIVersion: "rbac.authorization.k8s.io/v1",
		Delete:     true,
	},
	{
		Kind:       "ClusterRole",
		APIVersion: "rbac.authorization.k8s.io/v1",
		Delete:     true,
	},
	{
		Kind:       "RoleBinding",
		APIVersion: "rbac.authorization.k8s.io/v1",
		Delete:     true,
	},
	{
		Kind:       "ClusterRoleBinding",
		APIVersion: "rbac.authorization.k8s.io/v1",
		Delete:     true,
	},
}
