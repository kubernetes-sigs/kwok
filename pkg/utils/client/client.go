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
	"os"
	"path/filepath"
	"runtime"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/flowcontrol"
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
}

// NewClientset creates a new Clientset.
func NewClientset(masterURL, kubeconfigPath string) (*Clientset, error) {
	return &Clientset{
		masterURL:      masterURL,
		kubeconfigPath: kubeconfigPath,
	}, nil
}

// ToRESTConfig returns a REST config.
func (g *Clientset) ToRESTConfig() (*rest.Config, error) {
	if g.restConfig == nil {
		restConfig, err := g.ToRawKubeConfigLoader().ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("could not get Kubernetes config: %w", err)
		}
		restConfig.APIPath = "/api"
		restConfig.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
		restConfig.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
		restConfig.UserAgent = defaultKubernetesUserAgent()
		restConfig.NegotiatedSerializer = unstructuredscheme.NewUnstructuredNegotiatedSerializer()
		if err != nil {
			return nil, fmt.Errorf("could not get Kubernetes config: %w", err)
		}
		g.restConfig = restConfig
	}
	return g.restConfig, nil
}

// ToDiscoveryClient returns a discovery client.
func (g *Clientset) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if g.discoveryClient == nil {
		restConfig, err := g.ToRESTConfig()
		if err != nil {
			return nil, err
		}
		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("could not get Kubernetes client: %w", err)
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

type cachedDiscoveryInterface struct {
	discovery.DiscoveryInterface
}

var _ discovery.CachedDiscoveryInterface = &cachedDiscoveryInterface{}

func (d *cachedDiscoveryInterface) Fresh() bool {
	return false
}

func (d *cachedDiscoveryInterface) Invalidate() {}

// buildUserAgent builds a User-Agent string from given args.
func buildUserAgent(command, os, arch string) string {
	return fmt.Sprintf("%s (%s/%s)", command, os, arch)
}

// adjustCommand returns the last component of the
// OS-specific command path for use in User-Agent.
func adjustCommand(p string) string {
	// Unlikely, but better than returning "".
	if len(p) == 0 {
		return "unknown"
	}
	return filepath.Base(p)
}

// defaultKubernetesUserAgent returns a User-Agent string built from static global vars.
func defaultKubernetesUserAgent() string {
	return buildUserAgent(
		adjustCommand(os.Args[0]),
		runtime.GOOS,
		runtime.GOARCH,
	)
}
