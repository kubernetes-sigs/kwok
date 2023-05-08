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

package kind

import (
	"context"
	"net"
	"net/url"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
)

// AddContext add the context of cluster to kubeconfig
func (c *Cluster) AddContext(ctx context.Context, kubeconfigPath string) error {
	kubeConfig := &kubeconfig.Config{
		Context: &clientcmdapi.Context{
			Cluster:  "kind-" + c.Name(),
			AuthInfo: "kind-" + c.Name(),
		},
	}
	err := kubeconfig.AddContext(kubeconfigPath, c.Name(), kubeConfig)
	if err != nil {
		return err
	}
	return nil
}

// RemoveContext remove the context of cluster from kubeconfig
func (c *Cluster) RemoveContext(ctx context.Context, kubeconfigPath string) error {
	err := kubeconfig.RemoveContext(kubeconfigPath, c.Name())
	if err != nil {
		return err
	}
	return nil
}

// fillKubeconfigContextServer fill the server of cluster to kubeconfig
// because the server of cluster not set in kind, so we need to fill it.
func (c *Cluster) fillKubeconfigContextServer() error {
	kubeconfigPath := kubeconfig.GetRecommendedKubeconfigPath()
	name := "kind-" + c.Name()
	err := kubeconfig.ModifyContext(kubeconfigPath, withFillContextServer(name, "0.0.0.0"))
	if err != nil {
		return err
	}
	return nil
}

func withFillContextServer(name string, bindAddress string) func(kubeconfig *clientcmdapi.Config) error {
	return func(kubeconfig *clientcmdapi.Config) error {
		if kubeconfig.Clusters == nil {
			return nil
		}
		if kubeconfig.Clusters[name] == nil {
			return nil
		}
		if kubeconfig.Clusters[name].Server == "" {
			return nil
		}

		server, err := url.Parse(kubeconfig.Clusters[name].Server)
		if err != nil {
			return err
		}

		host, port, err := net.SplitHostPort(server.Host)
		if err != nil {
			return err
		}
		if host != "" {
			return nil
		}

		server.Host = net.JoinHostPort(bindAddress, port)
		kubeconfig.Clusters[name].Server = server.String()
		return nil
	}
}
