/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	context "context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
	apisv1alpha1 "sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	scheme "sigs.k8s.io/kwok/pkg/client/clientset/versioned/scheme"
)

// ClusterResourceUsagesGetter has a method to return a ClusterResourceUsageInterface.
// A group's client should implement this interface.
type ClusterResourceUsagesGetter interface {
	ClusterResourceUsages() ClusterResourceUsageInterface
}

// ClusterResourceUsageInterface has methods to work with ClusterResourceUsage resources.
type ClusterResourceUsageInterface interface {
	Create(ctx context.Context, clusterResourceUsage *apisv1alpha1.ClusterResourceUsage, opts v1.CreateOptions) (*apisv1alpha1.ClusterResourceUsage, error)
	Update(ctx context.Context, clusterResourceUsage *apisv1alpha1.ClusterResourceUsage, opts v1.UpdateOptions) (*apisv1alpha1.ClusterResourceUsage, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, clusterResourceUsage *apisv1alpha1.ClusterResourceUsage, opts v1.UpdateOptions) (*apisv1alpha1.ClusterResourceUsage, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*apisv1alpha1.ClusterResourceUsage, error)
	List(ctx context.Context, opts v1.ListOptions) (*apisv1alpha1.ClusterResourceUsageList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *apisv1alpha1.ClusterResourceUsage, err error)
	ClusterResourceUsageExpansion
}

// clusterResourceUsages implements ClusterResourceUsageInterface
type clusterResourceUsages struct {
	*gentype.ClientWithList[*apisv1alpha1.ClusterResourceUsage, *apisv1alpha1.ClusterResourceUsageList]
}

// newClusterResourceUsages returns a ClusterResourceUsages
func newClusterResourceUsages(c *KwokV1alpha1Client) *clusterResourceUsages {
	return &clusterResourceUsages{
		gentype.NewClientWithList[*apisv1alpha1.ClusterResourceUsage, *apisv1alpha1.ClusterResourceUsageList](
			"clusterresourceusages",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *apisv1alpha1.ClusterResourceUsage { return &apisv1alpha1.ClusterResourceUsage{} },
			func() *apisv1alpha1.ClusterResourceUsageList { return &apisv1alpha1.ClusterResourceUsageList{} },
		),
	}
}
