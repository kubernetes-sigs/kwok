/*
Copyright 2022 The Kubernetes Authors.

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
	"fmt"
	"strings"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildKubeApiserverComponentConfig is the configuration for building a kube-apiserver component.
type BuildKubeApiserverComponentConfig struct {
	Runtime               string
	ProjectName           string
	Binary                string
	Image                 string
	Version               version.Version
	Workdir               string
	BindAddress           string
	Port                  uint32
	EtcdAddress           string
	EtcdPort              uint32
	KubeRuntimeConfig     string
	KubeFeatureGates      string
	SecurePort            bool
	KubeAuthorization     bool
	KubeAdmission         bool
	AuditPolicyPath       string
	AuditLogPath          string
	CaCertPath            string
	AdminCertPath         string
	AdminKeyPath          string
	Verbosity             log.Level
	DisableQPSLimits      bool
	TracingConfigPath     string
	EtcdPrefix            string
	CORSAllowedOriginList []string
}

// BuildKubeApiserverComponent builds a kube-apiserver component.
func BuildKubeApiserverComponent(conf BuildKubeApiserverComponentConfig) (component internalversion.Component, err error) {
	if conf.EtcdPort == 0 {
		conf.EtcdPort = 2379
	}

	kubeApiserverArgs := []string{
		"--etcd-prefix=" + conf.EtcdPrefix,
		"--allow-privileged=true",
	}

	if len(conf.CORSAllowedOriginList) != 0 {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--cors-allowed-origins="+strings.Join(conf.CORSAllowedOriginList, ","),
		)
	}

	if conf.KubeAdmission {
		if conf.Version.LT(version.NewVersion(1, 21, 0)) && !conf.KubeAuthorization {
			return component, fmt.Errorf("the kube-apiserver version is less than 1.21.0, and the --kube-authorization is not enabled, so the --kube-admission cannot be enabled")
		}
	} else {
		// TODO: use enable-admission-plugins and disable-admission-plugins instead of admission-control
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--admission-control=",
		)
	}

	if conf.KubeRuntimeConfig != "" {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--runtime-config="+conf.KubeRuntimeConfig,
		)
	}

	var featureGates []string
	if conf.KubeFeatureGates != "" {
		featureGates = append(featureGates, strings.Split(conf.KubeFeatureGates, ",")...)
	}

	if conf.TracingConfigPath != "" {
		if conf.Version.LT(version.NewVersion(1, 22, 0)) {
			return component, fmt.Errorf("the kube-apiserver version is less than 1.22.0, so the --jaeger-port cannot be enabled")
		} else if conf.Version.LT(version.NewVersion(1, 27, 0)) {
			featureGates = append(featureGates, "APIServerTracing=true")
		}
	}

	if len(featureGates) != 0 {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--feature-gates="+strings.Join(featureGates, ","),
		)
	}

	if conf.DisableQPSLimits {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--max-requests-inflight=0",
			"--max-mutating-requests-inflight=0",
		)

		// FeatureGate APIPriorityAndFairness is not available before 1.17.0
		if conf.Version.GE(version.NewVersion(1, 18, 0)) {
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--enable-priority-and-fairness=false",
			)
		}
	}

	var ports []internalversion.Port
	var volumes []internalversion.Volume
	var metric *internalversion.ComponentMetric

	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--etcd-servers=http://"+conf.EtcdAddress+":2379",
		)
	} else {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--etcd-servers=http://"+conf.EtcdAddress+":"+format.String(conf.EtcdPort),
		)
	}

	if conf.SecurePort {
		if conf.KubeAuthorization {
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--authorization-mode=Node,RBAC",
			)
		}

		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			ports = []internalversion.Port{
				{
					HostPort: conf.Port,
					Port:     6443,
				},
			}
			volumes = append(volumes,
				internalversion.Volume{
					HostPath:  conf.CaCertPath,
					MountPath: "/etc/kubernetes/pki/ca.crt",
					ReadOnly:  true,
				},
				internalversion.Volume{
					HostPath:  conf.AdminCertPath,
					MountPath: "/etc/kubernetes/pki/admin.crt",
					ReadOnly:  true,
				},
				internalversion.Volume{
					HostPath:  conf.AdminKeyPath,
					MountPath: "/etc/kubernetes/pki/admin.key",
					ReadOnly:  true,
				},
			)
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--bind-address="+conf.BindAddress,
				"--secure-port=6443",
				"--tls-cert-file=/etc/kubernetes/pki/admin.crt",
				"--tls-private-key-file=/etc/kubernetes/pki/admin.key",
				"--client-ca-file=/etc/kubernetes/pki/ca.crt",
				"--service-account-key-file=/etc/kubernetes/pki/admin.key",
				"--service-account-signing-key-file=/etc/kubernetes/pki/admin.key",
				"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
				"--proxy-client-key-file=/etc/kubernetes/pki/admin.key",
				"--proxy-client-cert-file=/etc/kubernetes/pki/admin.crt",
			)
			metric = &internalversion.ComponentMetric{
				Scheme:             "https",
				Host:               conf.ProjectName + "-" + consts.ComponentKubeApiserver + ":6443",
				Path:               "/metrics",
				CertPath:           "/etc/kubernetes/pki/admin.crt",
				KeyPath:            "/etc/kubernetes/pki/admin.key",
				InsecureSkipVerify: true,
			}
		} else {
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--bind-address="+conf.BindAddress,
				"--secure-port="+format.String(conf.Port),
				"--tls-cert-file="+conf.AdminCertPath,
				"--tls-private-key-file="+conf.AdminKeyPath,
				"--client-ca-file="+conf.CaCertPath,
				"--service-account-key-file="+conf.AdminKeyPath,
				"--service-account-signing-key-file="+conf.AdminKeyPath,
				"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
				"--proxy-client-key-file="+conf.AdminKeyPath,
				"--proxy-client-cert-file="+conf.AdminCertPath,
			)
			metric = &internalversion.ComponentMetric{
				Scheme:             "https",
				Host:               net.LocalAddress + ":" + format.String(conf.Port),
				Path:               "/metrics",
				CertPath:           conf.AdminCertPath,
				KeyPath:            conf.AdminKeyPath,
				InsecureSkipVerify: true,
			}
		}
	} else {
		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			ports = []internalversion.Port{
				{
					HostPort: conf.Port,
					Port:     8080,
				},
			}

			kubeApiserverArgs = append(kubeApiserverArgs,
				"--insecure-bind-address="+conf.BindAddress,
				"--insecure-port=8080",
			)
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   conf.ProjectName + "-" + consts.ComponentKubeApiserver + ":8080",
				Path:   "/metrics",
			}
		} else {
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--insecure-bind-address="+conf.BindAddress,
				"--insecure-port="+format.String(conf.Port),
			)
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   net.LocalAddress + ":" + format.String(conf.Port),
				Path:   "/metrics",
			}
		}
	}

	if conf.AuditPolicyPath != "" {
		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			volumes = append(volumes,
				internalversion.Volume{
					HostPath:  conf.AuditPolicyPath,
					MountPath: "/etc/kubernetes/audit-policy.yaml",
					ReadOnly:  true,
				},
				internalversion.Volume{
					HostPath:  conf.AuditLogPath,
					MountPath: "/var/log/kubernetes/audit/audit.log",
					ReadOnly:  false,
				},
			)
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--audit-policy-file=/etc/kubernetes/audit-policy.yaml",
				"--audit-log-path=/var/log/kubernetes/audit/audit.log",
			)
		} else {
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--audit-policy-file="+conf.AuditPolicyPath,
				"--audit-log-path="+conf.AuditLogPath,
			)
		}
	}

	if conf.TracingConfigPath != "" {
		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			volumes = append(volumes,
				internalversion.Volume{
					HostPath:  conf.TracingConfigPath,
					MountPath: "/etc/kubernetes/apiserver-tracing-config.yaml",
					ReadOnly:  true,
				},
			)
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--tracing-config-file=/etc/kubernetes/apiserver-tracing-config.yaml",
			)
		} else {
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--tracing-config-file="+conf.TracingConfigPath,
			)
		}
	}

	if conf.Verbosity != log.LevelInfo {
		kubeApiserverArgs = append(kubeApiserverArgs, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	envs := []internalversion.Env{}

	links := []string{consts.ComponentEtcd}
	if conf.TracingConfigPath != "" {
		links = append(links, consts.ComponentJaeger)
	}

	return internalversion.Component{
		Name:    consts.ComponentKubeApiserver,
		Version: conf.Version.String(),
		Links:   links,
		Command: []string{consts.ComponentKubeApiserver},
		Ports:   ports,
		Volumes: volumes,
		Args:    kubeApiserverArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		Metric:  metric,
		WorkDir: conf.Workdir,
		Envs:    envs,
	}, nil
}
