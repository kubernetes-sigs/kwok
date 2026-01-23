/*
Copyright 2024 The Kubernetes Authors.

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
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

// The following paths are used for in-cluster configuration.
const (
	// InClusterSecretPath is the path where the service account token and CA certificate are mounted.
	InClusterSecretPath = "/var/run/secrets/kubernetes.io/serviceaccount"
	// InClusterTokenPath is the path to the service account token.
	InClusterTokenPath = InClusterSecretPath + "/token"
	// InClusterCACertPath is the path to the CA certificate.
	InClusterCACertPath = InClusterSecretPath + "/ca.crt"
)

// InClusterConfig holds configuration for in-cluster Kubernetes client configuration.
type InClusterConfig struct {
	// Host is the Kubernetes API server host (KUBERNETES_SERVICE_HOST).
	Host string
	// Port is the Kubernetes API server port (KUBERNETES_SERVICE_PORT).
	Port uint32
	// TokenHostPath is the path to the token file on the host.
	TokenHostPath string
	// CACertHostPath is the path to the CA certificate on the host.
	CACertHostPath string
}

// InClusterEnvs returns the environment variables for in-cluster configuration.
func InClusterEnvs(conf InClusterConfig) []internalversion.Env {
	return []internalversion.Env{
		{
			Name:  "KUBERNETES_SERVICE_HOST",
			Value: conf.Host,
		},
		{
			Name:  "KUBERNETES_SERVICE_PORT",
			Value: format.String(conf.Port),
		},
	}
}

// InClusterVolumes returns the volumes for in-cluster configuration.
func InClusterVolumes(conf InClusterConfig) []internalversion.Volume {
	return []internalversion.Volume{
		{
			Name:      "serviceaccount-token",
			HostPath:  conf.TokenHostPath,
			MountPath: InClusterTokenPath,
			ReadOnly:  true,
		},
		{
			Name:      "serviceaccount-ca",
			HostPath:  conf.CACertHostPath,
			MountPath: InClusterCACertPath,
			ReadOnly:  true,
		},
	}
}
