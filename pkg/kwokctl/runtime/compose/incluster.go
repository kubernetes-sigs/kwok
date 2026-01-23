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

package compose

import (
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

// The following paths are used for in-cluster configuration.
const (
	// InClusterSecretPath is the path where the service account token and CA certificate are mounted.
	inClusterSecretPath = "/var/run/secrets/kubernetes.io/serviceaccount"
	// InClusterTokenPath is the path to the service account token.
	inClusterTokenPath = inClusterSecretPath + "/token"
	// InClusterCACertPath is the path to the CA certificate.
	inClusterCACertPath = inClusterSecretPath + "/ca.crt"
)

// inClusterConfig holds configuration for in-cluster Kubernetes client configuration.
type inClusterConfig struct {
	// Host is the Kubernetes API server host (KUBERNETES_SERVICE_HOST).
	Host string
	// Port is the Kubernetes API server port (KUBERNETES_SERVICE_PORT).
	Port uint32
	// TokenHostPath is the path to the token file on the host.
	TokenHostPath string
	// CACertHostPath is the path to the CA certificate on the host.
	CACertHostPath string
}

// ApplyTo applies the in-cluster configuration to the given component.
func (c inClusterConfig) ApplyTo(component *internalversion.Component) {
	component.Envs = append(component.Envs,
		internalversion.Env{
			Name:  "KUBERNETES_SERVICE_HOST",
			Value: c.Host,
		},
		internalversion.Env{
			Name:  "KUBERNETES_SERVICE_PORT",
			Value: format.String(c.Port),
		},
	)
	component.Volumes = append(component.Volumes,
		internalversion.Volume{
			Name:      "serviceaccount-token",
			HostPath:  c.TokenHostPath,
			MountPath: inClusterTokenPath,
			ReadOnly:  true,
		},
		internalversion.Volume{
			Name:      "serviceaccount-ca",
			HostPath:  c.CACertHostPath,
			MountPath: inClusterCACertPath,
			ReadOnly:  true,
		},
	)
}
