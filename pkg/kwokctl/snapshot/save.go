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

package snapshot

import (
	"context"
	"fmt"
	"io"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/pager"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// Save saves the snapshot of cluster
func Save(ctx context.Context, kubeconfigPath string, w io.Writer, resources []string) error {
	clientset, err := client.NewClientset("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}
	restConfig, err := clientset.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get rest config: %w", err)
	}

	restMapper, err := clientset.ToRESTMapper()
	if err != nil {
		return fmt.Errorf("failed to get rest mapper: %w", err)
	}
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	gvrs := make([]schema.GroupVersionResource, 0, len(resources))
	for _, resource := range resources {
		mapping, err := mappingFor(restMapper, resource)
		if err != nil {
			return fmt.Errorf("failed to get mapping for resource %q: %w", resource, err)
		}
		gvrs = append(gvrs, mapping.Resource)
	}

	encoder := yaml.NewEncoder(w)
	for _, gvr := range gvrs {
		nri := dynClient.Resource(gvr)

		listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			var list runtime.Object
			var err error
			err = retry.OnError(retry.DefaultBackoff, retriable, func() error {
				list, err = nri.List(ctx, opts)
				return err
			})
			return list, err
		})

		if err := listPager.EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
			if o, ok := obj.(metav1.Object); ok {
				clearUnstructured(o)
			}
			return encoder.Encode(obj)
		}); err != nil {
			return fmt.Errorf("failed to list resource %q: %w", gvr.Resource, err)
		}
	}

	return nil
}

func retriable(err error) bool {
	return apierrors.IsInternalError(err) ||
		apierrors.IsServiceUnavailable(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsTimeout(err) ||
		apierrors.IsServerTimeout(err) ||
		net.IsConnectionRefused(err)
}
