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
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildKubeApiserverComponentConfig is the configuration for building a kube-apiserver component.
type BuildKubeApiserverComponentConfig struct {
	Runtime           string
	ProjectName       string
	Binary            string
	Image             string
	Version           version.Version
	Workdir           string
	BindAddress       string
	Port              uint32
	EtcdAddress       string
	EtcdPort          uint32
	KubeRuntimeConfig string
	KubeFeatureGates  string
	SecurePort        bool
	KubeAuthorization bool
	KubeAdmission     bool
	AuditPolicyPath   string
	AuditLogPath      string
	CaCertPath        string
	AdminCertPath     string
	AdminKeyPath      string
	Verbosity         log.Level
	DisableQPSLimits  bool
	TracingConfigPath string
	EtcdPrefix        string
}

// BuildKubeApiserverComponent builds a kube-apiserver component.
func BuildKubeApiserverComponent(conf BuildKubeApiserverComponentConfig) (component internalversion.Component, err error) {
	var args []string
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	var metric *internalversion.ComponentMetric

	if conf.EtcdPort == 0 {
		conf.EtcdPort = 2379
	}

	args = append(args,
		"--etcd-prefix="+conf.EtcdPrefix,
		"--allow-privileged=true",
		"--endpoint-reconciler-type=none",
	)

	if conf.KubeAdmission {
		if conf.Version.LT(version.NewVersion(1, 21, 0)) && !conf.KubeAuthorization {
			return component, fmt.Errorf("the kube-apiserver version is less than 1.21.0, and the --kube-authorization is not enabled, so the --kube-admission cannot be enabled")
		}
	} else {
		// TODO: use enable-admission-plugins and disable-admission-plugins instead of admission-control
		args = append(args,
			"--admission-control=",
		)
	}

	if conf.KubeRuntimeConfig != "" {
		args = append(args,
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
		args = append(args,
			"--feature-gates="+strings.Join(featureGates, ","),
		)
	}

	if conf.DisableQPSLimits {
		args = append(args,
			"--max-requests-inflight=0",
			"--max-mutating-requests-inflight=0",
		)

		// FeatureGate APIPriorityAndFairness is not available before 1.17.0
		if conf.Version.GE(version.NewVersion(1, 18, 0)) {
			args = append(args,
				"--enable-priority-and-fairness=false",
			)
		}
	}

	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		args = append(args,
			"--etcd-servers=http://"+conf.EtcdAddress+":2379",
		)
	} else {
		args = append(args,
			"--etcd-servers=http://"+conf.EtcdAddress+":"+format.String(conf.EtcdPort),
		)
	}

	if conf.SecurePort {
		if conf.KubeAuthorization {
			args = append(args,
				"--authorization-mode=Node,RBAC",
			)
		}

		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			ports = append(
				ports,
				internalversion.Port{
					Name:     schemeHTTPS,
					HostPort: conf.Port,
					Port:     6443,
					Protocol: internalversion.ProtocolTCP,
				},
			)
			volumes = append(volumes,
				internalversion.Volume{
					HostPath:  conf.CaCertPath,
					MountPath: pkiCACertPath,
					ReadOnly:  true,
				},
				internalversion.Volume{
					HostPath:  conf.AdminCertPath,
					MountPath: pkiAdminCertPath,
					ReadOnly:  true,
				},
				internalversion.Volume{
					HostPath:  conf.AdminKeyPath,
					MountPath: pkiAdminKeyPath,
					ReadOnly:  true,
				},
			)
			args = append(args,
				"--bind-address="+conf.BindAddress,
				"--secure-port=6443",
				"--tls-cert-file="+pkiAdminCertPath,
				"--tls-private-key-file="+pkiAdminKeyPath,
				"--client-ca-file="+pkiCACertPath,
				"--service-account-key-file="+pkiAdminKeyPath,
				"--service-account-signing-key-file="+pkiAdminKeyPath,
				"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
				"--proxy-client-key-file="+pkiAdminKeyPath,
				"--proxy-client-cert-file="+pkiAdminCertPath,
				"--requestheader-client-ca-file="+pkiCACertPath,
				"--requestheader-allowed-names="+pki.DefaultCN,
				"--requestheader-username-headers=X-Remote-User",
				"--requestheader-group-headers=X-Remote-Group",
				"--requestheader-extra-headers-prefix=X-Remote-Extra-",
			)
			metric = &internalversion.ComponentMetric{
				Scheme:             schemeHTTPS,
				Host:               conf.ProjectName + "-" + consts.ComponentKubeApiserver + ":6443",
				Path:               metricsPath,
				CertPath:           pkiAdminCertPath,
				KeyPath:            pkiAdminKeyPath,
				InsecureSkipVerify: true,
			}
		} else {
			ports = append(
				ports,
				internalversion.Port{
					Name:     schemeHTTPS,
					HostPort: 0,
					Port:     conf.Port,
					Protocol: internalversion.ProtocolTCP,
				},
			)
			args = append(args,
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
				"--requestheader-client-ca-file="+conf.CaCertPath,
				"--requestheader-allowed-names="+pki.DefaultCN,
				"--requestheader-username-headers=X-Remote-User",
				"--requestheader-group-headers=X-Remote-Group",
				"--requestheader-extra-headers-prefix=X-Remote-Extra-",
			)
			metric = &internalversion.ComponentMetric{
				Scheme:             schemeHTTPS,
				Host:               utilsnet.LocalAddress + ":" + format.String(conf.Port),
				Path:               metricsPath,
				CertPath:           conf.AdminCertPath,
				KeyPath:            conf.AdminKeyPath,
				InsecureSkipVerify: true,
			}
		}
	} else {
		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			ports = append(
				ports,
				internalversion.Port{
					Name:     "http",
					HostPort: conf.Port,
					Port:     8080,
					Protocol: internalversion.ProtocolTCP,
				},
			)

			args = append(args,
				"--insecure-bind-address="+conf.BindAddress,
				"--insecure-port=8080",
			)
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   conf.ProjectName + "-" + consts.ComponentKubeApiserver + ":8080",
				Path:   metricsPath,
			}
		} else {
			ports = append(
				ports,
				internalversion.Port{
					Name:     "http",
					HostPort: 0,
					Port:     conf.Port,
					Protocol: internalversion.ProtocolTCP,
				},
			)
			args = append(args,
				"--insecure-bind-address="+conf.BindAddress,
				"--insecure-port="+format.String(conf.Port),
			)
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   utilsnet.LocalAddress + ":" + format.String(conf.Port),
				Path:   metricsPath,
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
			args = append(args,
				"--audit-policy-file=/etc/kubernetes/audit-policy.yaml",
				"--audit-log-path=/var/log/kubernetes/audit/audit.log",
			)
		} else {
			args = append(args,
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
			args = append(args,
				"--tracing-config-file=/etc/kubernetes/apiserver-tracing-config.yaml",
			)
		} else {
			args = append(args,
				"--tracing-config-file="+conf.TracingConfigPath,
			)
		}
	}

	if conf.Verbosity != log.LevelInfo {
		args = append(args, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

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
		Args:    args,
		Binary:  conf.Binary,
		Image:   conf.Image,
		Metric:  metric,
		WorkDir: conf.Workdir,
	}, nil
}
