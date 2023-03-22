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
