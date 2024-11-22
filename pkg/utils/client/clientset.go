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

package client

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/flowcontrol"

	"sigs.k8s.io/kwok/pkg/utils/version"
)

// Clientset is a set of Kubernetes clients.
type Clientset interface {
	ToRESTConfig() (*rest.Config, error)
	ToRawKubeConfigLoader() clientcmd.ClientConfig
	ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error)
	ToRESTMapper() (meta.RESTMapper, error)
	ToDynamicClient() (dynamic.Interface, error)
	ToImpersonatingDynamicClient() DynamicClientImpersonator
}

// clientset is a set of Kubernetes clients.
type clientset struct {
	masterURL       string
	kubeconfigPath  string
	restConfig      *rest.Config
	discoveryClient discovery.CachedDiscoveryInterface
	restMapper      meta.RESTMapper
	clientConfig    clientcmd.ClientConfig
	dynamicClient   dynamic.Interface

	impersonationCache map[string]dynamic.Interface

	opts []Option
}

// Option is a function that configures a clientset.
type Option func(*clientset)

// WithImpersonate sets the impersonation config.
func WithImpersonate(impersonateConfig rest.ImpersonationConfig) Option {
	return func(c *clientset) {
		c.restConfig.Impersonate = impersonateConfig
	}
}

// NewClientset creates a new clientset.
func NewClientset(masterURL, kubeconfigPath string, opts ...Option) (Clientset, error) {
	return &clientset{
		masterURL:          masterURL,
		kubeconfigPath:     kubeconfigPath,
		opts:               opts,
		impersonationCache: map[string]dynamic.Interface{},
	}, nil
}

// ToRESTConfig returns a REST config.
func (g *clientset) ToRESTConfig() (*rest.Config, error) {
	if g.restConfig == nil {
		var restConfig *rest.Config
		if g.kubeconfigPath == "" {
			clientConfig, err := rest.InClusterConfig()
			if err != nil {
				return nil, fmt.Errorf("could not get in ClusterConfig: %w", err)
			}
			if g.masterURL != "" {
				clientConfig.Host = g.masterURL
			}
			restConfig = clientConfig
		} else {
			clientConfig, err := g.ToRawKubeConfigLoader().ClientConfig()
			if err != nil {
				return nil, fmt.Errorf("could not get Kubernetes config: %w", err)
			}
			restConfig = clientConfig
		}
		restConfig.GroupVersion = &schema.GroupVersion{}
		restConfig.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
		restConfig.UserAgent = version.DefaultUserAgent()
		restConfig.NegotiatedSerializer = unstructuredscheme.NewUnstructuredNegotiatedSerializer()
		// restConfig.Wrap(newRoundTripperPool)
		g.restConfig = restConfig

		for _, opt := range g.opts {
			opt(g)
		}
	}
	return g.restConfig, nil
}

// ToDiscoveryClient returns a discovery client.
func (g *clientset) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if g.discoveryClient == nil {
		restConfig, err := g.ToRESTConfig()
		if err != nil {
			return nil, err
		}
		discoveryCli, err := discovery.NewDiscoveryClientForConfig(restConfig)
		if err != nil {
			return nil, err
		}
		discoveryClient := &cachedDiscoveryInterface{discoveryCli}
		g.discoveryClient = discoveryClient
	}
	return g.discoveryClient, nil
}

// ToRESTMapper returns a REST mapper.
func (g *clientset) ToRESTMapper() (meta.RESTMapper, error) {
	if g.restMapper == nil {
		discoveryClient, err := g.ToDiscoveryClient()
		if err != nil {
			return nil, err
		}

		restMapper := newLazyRESTMapperWithClient(discoveryClient)

		g.restMapper = restMapper
	}
	return g.restMapper, nil
}

// ToRawKubeConfigLoader returns a raw kubeconfig loader.
func (g *clientset) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	if g.clientConfig == nil {
		g.clientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: g.kubeconfigPath},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: g.masterURL}})
	}
	return g.clientConfig
}

// ToDynamicClient returns a dynamic Kubernetes client.
func (g *clientset) ToDynamicClient() (dynamic.Interface, error) {
	if g.dynamicClient == nil {
		restConfig, err := g.ToRESTConfig()
		if err != nil {
			return nil, err
		}
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("could not get Kubernetes dynamicClient: %w", err)
		}
		g.dynamicClient = dynamicClient
	}
	return g.dynamicClient, nil
}

// ToImpersonatingDynamicClient returns a dynamic Kubernetes client.
func (g *clientset) ToImpersonatingDynamicClient() DynamicClientImpersonator {
	return g
}

type cachedDiscoveryInterface struct {
	discovery.DiscoveryInterface
}

var _ discovery.CachedDiscoveryInterface = &cachedDiscoveryInterface{}

func (d *cachedDiscoveryInterface) Fresh() bool {
	return false
}

func (d *cachedDiscoveryInterface) Invalidate() {}
