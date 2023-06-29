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
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/flowcontrol"

	"sigs.k8s.io/kwok/pkg/client/clientset/versioned"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// Clientset is a set of Kubernetes clients.
type Clientset struct {
	masterURL       string
	kubeconfigPath  string
	restConfig      *rest.Config
	discoveryClient discovery.CachedDiscoveryInterface
	restMapper      meta.RESTMapper
	restClient      *rest.RESTClient
	clientConfig    clientcmd.ClientConfig
	typedClient     *kubernetes.Clientset
	kwokClient      *versioned.Clientset
	dynamicClient   *dynamic.DynamicClient

	opts []Option
}

// Option is a function that configures a Clientset.
type Option func(*Clientset)

// WithImpersonate sets the impersonation config.
func WithImpersonate(impersonateConfig rest.ImpersonationConfig) Option {
	return func(c *Clientset) {
		c.restConfig.Impersonate = impersonateConfig
	}
}

// NewClientset creates a new Clientset.
func NewClientset(masterURL, kubeconfigPath string, opts ...Option) (*Clientset, error) {
	return &Clientset{
		masterURL:      masterURL,
		kubeconfigPath: kubeconfigPath,
		opts:           opts,
	}, nil
}

// ToRESTConfig returns a REST config.
func (g *Clientset) ToRESTConfig() (*rest.Config, error) {
	if g.restConfig == nil {
		var restConfig *rest.Config
		if g.kubeconfigPath == "" && g.masterURL == "" {
			clientConfig, err := rest.InClusterConfig()
			if err != nil {
				return nil, fmt.Errorf("could not get in ClusterConfig: %w", err)
			}
			restConfig = clientConfig
		} else {
			clientConfig, err := g.ToRawKubeConfigLoader().ClientConfig()
			if err != nil {
				return nil, fmt.Errorf("could not get Kubernetes config: %w", err)
			}
			restConfig = clientConfig
		}
		restConfig.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
		restConfig.UserAgent = version.DefaultUserAgent()
		restConfig.NegotiatedSerializer = unstructuredscheme.NewUnstructuredNegotiatedSerializer()
		g.restConfig = restConfig

		for _, opt := range g.opts {
			opt(g)
		}
	}
	return g.restConfig, nil
}

// ToDiscoveryClient returns a discovery client.
func (g *Clientset) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if g.discoveryClient == nil {
		clientset, err := g.ToTypedClient()
		if err != nil {
			return nil, err
		}
		discoveryClient := &cachedDiscoveryInterface{clientset.DiscoveryClient}
		g.discoveryClient = discoveryClient
	}
	return g.discoveryClient, nil
}

// ToRESTMapper returns a REST mapper.
func (g *Clientset) ToRESTMapper() (meta.RESTMapper, error) {
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

// ToRESTClient returns a REST client.
func (g *Clientset) ToRESTClient() (*rest.RESTClient, error) {
	if g.restClient == nil {
		restConfig, err := g.ToRESTConfig()
		if err != nil {
			return nil, err
		}
		restClient, err := rest.RESTClientFor(restConfig)
		if err != nil {
			return nil, fmt.Errorf("could not get Kubernetes REST client: %w", err)
		}
		g.restClient = restClient
	}
	return g.restClient, nil
}

// ToRawKubeConfigLoader returns a raw kubeconfig loader.
func (g *Clientset) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	if g.clientConfig == nil {
		g.clientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: g.kubeconfigPath},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: g.masterURL}})
	}
	return g.clientConfig
}

// ToTypedKwokClient returns a typed kwok client.
func (g *Clientset) ToTypedKwokClient() (*versioned.Clientset, error) {
	if g.kwokClient == nil {
		restConfig, err := g.ToRESTConfig()
		if err != nil {
			return nil, err
		}
		typedKwokClient, err := versioned.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("could not get Kubernetes typedClient: %w", err)
		}
		g.kwokClient = typedKwokClient
	}
	return g.kwokClient, nil
}

// ToTypedClient returns a typed Kubernetes client.
func (g *Clientset) ToTypedClient() (*kubernetes.Clientset, error) {
	if g.typedClient == nil {
		restConfig, err := g.ToRESTConfig()
		if err != nil {
			return nil, err
		}
		typedClient, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("could not get Kubernetes typedClient: %w", err)
		}
		g.typedClient = typedClient
	}
	return g.typedClient, nil
}

// ToDynamicClient returns a dynamic Kubernetes client.
func (g *Clientset) ToDynamicClient() (*dynamic.DynamicClient, error) {
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

type cachedDiscoveryInterface struct {
	discovery.DiscoveryInterface
}

var _ discovery.CachedDiscoveryInterface = &cachedDiscoveryInterface{}

func (d *cachedDiscoveryInterface) Fresh() bool {
	return false
}

func (d *cachedDiscoveryInterface) Invalidate() {}
