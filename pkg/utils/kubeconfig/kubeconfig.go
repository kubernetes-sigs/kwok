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

package kubeconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"sigs.k8s.io/kwok/pkg/utils/envs"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

const (
	recommendedConfigPathEnvVar = "KUBECONFIG"
	recommendedHomeDir          = ".kube"
	recommendedFileName         = "config"
)

// GetRecommendedKubeconfigPath returns the recommended config file based on the current environment
func GetRecommendedKubeconfigPath() string {
	defaultPath := path.Join(envs.GetEnv("HOME", ""), recommendedHomeDir, recommendedFileName)
	defaultPath = envs.GetEnv(recommendedConfigPathEnvVar, defaultPath)
	return defaultPath
}

// Config is a struct that contains the information needed to create a kubeconfig file
type Config struct {
	Cluster *clientcmdapi.Cluster
	User    *clientcmdapi.AuthInfo
	Context *clientcmdapi.Context
}

// AddContext adds a context to the kubeconfig file
func AddContext(kubeconfigPath, contextName string, config *Config) error {
	err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0750)
	if err != nil {
		return err
	}
	return ModifyContext(kubeconfigPath, func(kubeconfig *clientcmdapi.Config) error {
		if config.Cluster != nil {
			if kubeconfig.Clusters == nil {
				kubeconfig.Clusters = map[string]*clientcmdapi.Cluster{}
			}
			kubeconfig.Clusters[contextName] = config.Cluster
		}

		if config.User != nil {
			if kubeconfig.AuthInfos == nil {
				kubeconfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{}
			}
			kubeconfig.AuthInfos[contextName] = config.User
		}

		if config.Context != nil {
			if kubeconfig.Contexts == nil {
				kubeconfig.Contexts = map[string]*clientcmdapi.Context{}
			}
			kubeconfig.Contexts[contextName] = config.Context
		}

		kubeconfig.CurrentContext = contextName
		return nil
	})
}

// RemoveContext removes a context from the kubeconfig file
func RemoveContext(kubeconfigPath, contextName string) error {
	return ModifyContext(kubeconfigPath, func(kubeconfig *clientcmdapi.Config) error {
		if kubeconfig.Contexts != nil {
			delete(kubeconfig.Contexts, contextName)
		}
		if kubeconfig.Clusters != nil {
			delete(kubeconfig.Clusters, contextName)
		}
		if kubeconfig.AuthInfos != nil {
			delete(kubeconfig.AuthInfos, contextName)
		}
		if kubeconfig.CurrentContext == contextName {
			kubeconfig.CurrentContext = ""
		}
		return nil
	})
}

// ModifyContext modifies the kubeconfig file
func ModifyContext(kubeconfigPath string, fun func(kubeconfig *clientcmdapi.Config) error) error {
	// load kubeconfig file
	kubeconfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			kubeconfig = &clientcmdapi.Config{}
		} else {
			return fmt.Errorf("failed to load kubeconfig file: %w", err)
		}
	}

	modified := kubeconfig.DeepCopy()
	err = fun(modified)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(kubeconfig, modified) {
		return nil
	}

	pathOptions := &clientcmd.PathOptions{
		GlobalFile:   kubeconfigPath,
		LoadingRules: &clientcmd.ClientConfigLoadingRules{},
	}
	// write config file
	if err := clientcmd.ModifyConfig(pathOptions, *modified, false); err != nil {
		return fmt.Errorf("failed to modify kubeconfig file: %w", err)
	}
	return nil
}
