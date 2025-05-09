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

package config

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	configv1alpha1 "sigs.k8s.io/kwok/pkg/apis/config/v1alpha1"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/envs"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

var (
	// DefaultCluster the default cluster name
	DefaultCluster = "kwok"

	// WorkDir is the directory of the work spaces.
	WorkDir = envs.GetEnvWithPrefix("WORKDIR", path.WorkDir())

	// ClustersDir is the directory of the clusters.
	ClustersDir = path.Join(WorkDir, "clusters")

	// GOOS is the operating system target for which the code is compiled.
	GOOS = runtime.GOOS

	// GOARCH is the architecture target for which the code is compiled.
	GOARCH = runtime.GOARCH

	// windows is the windows system
	windows = "windows"

	// linux is the linux system
	linux = "linux"

	// binarySuffixTar is the file suffix tar.gz
	binarySuffixTar = "tar.gz"

	// binarySuffixZip is the file suffix zip
	binarySuffixZip = "zip"
)

// ClusterName returns the cluster name.
func ClusterName(name string) string {
	return fmt.Sprintf("%s-%s", consts.ProjectName, name)
}

// GetKwokctlConfiguration get the configuration of the kwokctl.
func GetKwokctlConfiguration(ctx context.Context) (conf *internalversion.KwokctlConfiguration) {
	configs := FilterWithTypeFromContext[*internalversion.KwokctlConfiguration](ctx)
	if len(configs) != 0 {
		conf = configs[0]
		if len(configs) > 1 {
			logger := log.FromContext(ctx)
			logger.Warn("Too many same kind configurations",
				"kind", configv1alpha1.KwokctlConfigurationKind,
			)
		}
	}
	if conf == nil {
		logger := log.FromContext(ctx)
		logger.Debug("No configuration",
			"kind", configv1alpha1.KwokctlConfigurationKind,
		)
		conf, err := internalversion.ConvertToInternalKwokctlConfiguration(setKwokctlConfigurationDefaults(&configv1alpha1.KwokctlConfiguration{}))
		if err != nil {
			logger.Error("Get kwokctl configuration failed", err)
			return &internalversion.KwokctlConfiguration{}
		}
		addToContext(ctx, conf)
		return conf
	}
	return conf
}

// GetKwokConfiguration get the configuration of the kwok.
func GetKwokConfiguration(ctx context.Context) (conf *internalversion.KwokConfiguration) {
	configs := FilterWithTypeFromContext[*internalversion.KwokConfiguration](ctx)
	if len(configs) != 0 {
		conf = configs[0]
		if len(configs) > 1 {
			logger := log.FromContext(ctx)
			logger.Warn("Too many same kind configurations",
				"kind", configv1alpha1.KwokConfigurationKind,
			)
		}
	}
	if conf == nil {
		logger := log.FromContext(ctx)
		logger.Debug("No configuration",
			"kind", configv1alpha1.KwokConfigurationKind,
		)
		conf, err := internalversion.ConvertToInternalKwokConfiguration(setKwokConfigurationDefaults(&configv1alpha1.KwokConfiguration{}))
		if err != nil {
			logger.Error("Get kwok configuration failed", err)
			return &internalversion.KwokConfiguration{}
		}
		addToContext(ctx, conf)
		return conf
	}
	return conf
}

func convertToInternalStage(config *v1alpha1.Stage) (*internalversion.Stage, error) {
	obj := setStageDefaults(config)
	return internalversion.ConvertToInternalStage(obj)
}

func setStageDefaults(config *v1alpha1.Stage) *v1alpha1.Stage {
	if config == nil {
		config = &v1alpha1.Stage{}
	}
	v1alpha1.SetObjectDefaults_Stage(config)
	return config
}

func convertToInternalKwokConfiguration(config *configv1alpha1.KwokConfiguration) (*internalversion.KwokConfiguration, error) {
	obj := setKwokConfigurationDefaults(config)
	return internalversion.ConvertToInternalKwokConfiguration(obj)
}

func setKwokConfigurationDefaults(config *configv1alpha1.KwokConfiguration) *configv1alpha1.KwokConfiguration {
	if config == nil {
		config = &configv1alpha1.KwokConfiguration{}
	}

	configv1alpha1.SetObjectDefaults_KwokConfiguration(config)

	return config
}

func convertToInternalKwokctlConfiguration(config *configv1alpha1.KwokctlConfiguration) (*internalversion.KwokctlConfiguration, error) {
	obj := setKwokctlConfigurationDefaults(config)
	return internalversion.ConvertToInternalKwokctlConfiguration(obj)
}

func setKwokctlConfigurationDefaults(config *configv1alpha1.KwokctlConfiguration) *configv1alpha1.KwokctlConfiguration {
	if config == nil {
		config = &configv1alpha1.KwokctlConfiguration{}
	}

	configv1alpha1.SetObjectDefaults_KwokctlConfiguration(config)

	conf := &config.Options

	if conf.KwokVersion == "" {
		conf.KwokVersion = consts.Version
	}
	conf.KwokVersion = version.AddPrefixV(envs.GetEnvWithPrefix("VERSION", conf.KwokVersion))

	if conf.KubeVersion == "" {
		conf.KubeVersion = consts.KubeVersion
	}
	conf.KubeVersion = version.AddPrefixV(envs.GetEnvWithPrefix("KUBE_VERSION", conf.KubeVersion))

	if conf.SecurePort == nil {
		minor := parseRelease(conf.KubeVersion)
		conf.SecurePort = format.Ptr(minor > 12 || minor == -1)
	}
	conf.SecurePort = format.Ptr(envs.GetEnvWithPrefix("SECURE_PORT", *conf.SecurePort))

	if conf.KubeAuthorization == nil {
		conf.KubeAuthorization = conf.SecurePort
	}

	if conf.KubeAdmission == nil {
		conf.KubeAdmission = conf.KubeAuthorization
	}

	conf.QuietPull = format.Ptr(envs.GetEnvWithPrefix("QUIET_PULL", *conf.QuietPull))

	conf.Runtime = envs.GetEnvWithPrefix("RUNTIME", conf.Runtime)
	if conf.Runtime == "" && len(conf.Runtimes) == 0 {
		conf.Runtimes = []string{
			consts.RuntimeTypeDocker,
			consts.RuntimeTypePodman,
		}
		if GOOS == linux {
			conf.Runtimes = append(conf.Runtimes,
				consts.RuntimeTypeNerdctl,
			)
		}
		conf.Runtimes = append(conf.Runtimes,
			consts.RuntimeTypeBinary,
		)
	}
	if conf.Runtime == "" && len(conf.Runtimes) == 1 {
		conf.Runtime = conf.Runtimes[0]
	}

	conf.Mode = envs.GetEnvWithPrefix("MODE", conf.Mode)

	if conf.CacheDir == "" {
		conf.CacheDir = path.Join(WorkDir, "cache")
	}

	if conf.BinSuffix == "" {
		if GOOS == windows {
			conf.BinSuffix = ".exe"
		}
	}

	// Disable node lease duration seconds for kubernetes < 1.14
	if conf.NodeLeaseDurationSeconds != 0 {
		minor := parseRelease(conf.KubeVersion)
		if minor < 14 && minor != -1 {
			conf.NodeLeaseDurationSeconds = 0
		}
	}

	setKwokctlKubernetesConfig(conf)

	setKwokctlKwokConfig(conf)

	setKwokctlEtcdConfig(conf)

	setKwokctlKindConfig(conf)

	setKwokctlDashboardConfig(conf)

	setKwokctlPrometheusConfig(conf)

	setKwokctlJaegerConfig(conf)

	setMetricsServerConfig(conf)

	setKectlConfig(conf)

	return config
}

func setKectlConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	if conf.KectlVersion == "" {
		conf.KectlVersion = consts.KectlVersion
	}

	if conf.KectlBinary == "" {
		conf.KectlBinary = consts.KectlBinaryPrefix + "/" + version.AddPrefixV(conf.KectlVersion) + "/kectl_" + GOOS + "_" + GOARCH + conf.BinSuffix
	}
}

func setKwokctlKubernetesConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	conf.DisableKubeScheduler = format.Ptr(envs.GetEnvWithPrefix("DISABLE_KUBE_SCHEDULER", *conf.DisableKubeScheduler))
	conf.DisableKubeControllerManager = format.Ptr(envs.GetEnvWithPrefix("DISABLE_KUBE_CONTROLLER_MANAGER", *conf.DisableKubeControllerManager))
	if len(conf.Components) == 0 {
		conf.Components = []string{
			consts.ComponentEtcd,
			consts.ComponentKubeApiserver,
			consts.ComponentKubeControllerManager,
			consts.ComponentKubeScheduler,
			consts.ComponentKwokController,
		}
	}

	conf.KubeAuthorization = format.Ptr(envs.GetEnvWithPrefix("KUBE_AUTHORIZATION", *conf.KubeAuthorization))
	conf.KubeAdmission = envs.GetEnvWithPrefix("KUBE_ADMISSION", conf.KubeAdmission)

	conf.KubeApiserverPort = envs.GetEnvWithPrefix("KUBE_APISERVER_PORT", conf.KubeApiserverPort)
	conf.KubeApiserverInsecurePort = envs.GetEnvWithPrefix("KUBE_APISERVER_INSECURE_PORT", conf.KubeApiserverInsecurePort)

	if conf.KubeFeatureGates == "" {
		if conf.Mode == configv1alpha1.ModeStableFeatureGateAndAPI {
			conf.KubeFeatureGates = k8s.GetFeatureGates(parseRelease(conf.KubeVersion))
		}
	}
	conf.KubeFeatureGates = envs.GetEnvWithPrefix("KUBE_FEATURE_GATES", conf.KubeFeatureGates)

	if conf.KubeRuntimeConfig == "" {
		if conf.Mode == configv1alpha1.ModeStableFeatureGateAndAPI {
			conf.KubeRuntimeConfig = k8s.GetRuntimeConfig(parseRelease(conf.KubeVersion))
		}
	}
	conf.KubeRuntimeConfig = envs.GetEnvWithPrefix("KUBE_RUNTIME_CONFIG", conf.KubeRuntimeConfig)

	conf.KubeAuditPolicy = envs.GetEnvWithPrefix("KUBE_AUDIT_POLICY", conf.KubeAuditPolicy)

	kubectlBinaryPrefix := conf.KubeBinaryPrefix
	if conf.KubeBinaryPrefix == "" {
		// https://www.downloadkubernetes.com/
		// No provided for control plane components outside of Linux,
		// but kubectl is an exception.
		kubectlBinaryPrefix = consts.KubeBinaryPrefix + "/" + conf.KubeVersion + "/bin/" + GOOS + "/" + GOARCH
		if GOOS == linux {
			conf.KubeBinaryPrefix = kubectlBinaryPrefix
		} else {
			conf.KubeBinaryPrefix = consts.KubeBinaryUnofficialPrefix + "/" + conf.KubeVersion + "-kwok.0-" + GOOS + "-" + GOARCH
		}
	}
	conf.KubeBinaryPrefix = envs.GetEnvWithPrefix("KUBE_BINARY_PREFIX", conf.KubeBinaryPrefix)

	if conf.KubectlBinary == "" {
		conf.KubectlBinary = kubectlBinaryPrefix + "/kubectl" + conf.BinSuffix
	}
	conf.KubectlBinary = envs.GetEnvWithPrefix("KUBECTL_BINARY", conf.KubectlBinary)

	if conf.KubeApiserverBinary == "" {
		conf.KubeApiserverBinary = conf.KubeBinaryPrefix + "/kube-apiserver" + conf.BinSuffix
	}
	conf.KubeApiserverBinary = envs.GetEnvWithPrefix("KUBE_APISERVER_BINARY", conf.KubeApiserverBinary)

	if conf.KubeControllerManagerBinary == "" {
		conf.KubeControllerManagerBinary = conf.KubeBinaryPrefix + "/kube-controller-manager" + conf.BinSuffix
	}
	conf.KubeControllerManagerBinary = envs.GetEnvWithPrefix("KUBE_CONTROLLER_MANAGER_BINARY", conf.KubeControllerManagerBinary)

	if conf.KubeSchedulerBinary == "" {
		conf.KubeSchedulerBinary = conf.KubeBinaryPrefix + "/kube-scheduler" + conf.BinSuffix
	}
	conf.KubeSchedulerBinary = envs.GetEnvWithPrefix("KUBE_SCHEDULER_BINARY", conf.KubeSchedulerBinary)

	if conf.KubeImagePrefix == "" {
		conf.KubeImagePrefix = consts.KubeImagePrefix
	}
	conf.KubeImagePrefix = envs.GetEnvWithPrefix("KUBE_IMAGE_PREFIX", conf.KubeImagePrefix)

	if conf.KubeApiserverImage == "" {
		conf.KubeApiserverImage = joinImageURI(conf.KubeImagePrefix, "kube-apiserver", conf.KubeVersion)
	}
	conf.KubeApiserverImage = envs.GetEnvWithPrefix("KUBE_APISERVER_IMAGE", conf.KubeApiserverImage)

	if conf.KubeControllerManagerImage == "" {
		conf.KubeControllerManagerImage = joinImageURI(conf.KubeImagePrefix, "kube-controller-manager", conf.KubeVersion)
	}
	conf.KubeControllerManagerImage = envs.GetEnvWithPrefix("KUBE_CONTROLLER_MANAGER_IMAGE", conf.KubeControllerManagerImage)

	conf.KubeControllerManagerPort = envs.GetEnvWithPrefix("KUBE_CONTROLLER_MANAGER_PORT", conf.KubeControllerManagerPort)

	if conf.KubeSchedulerImage == "" {
		conf.KubeSchedulerImage = joinImageURI(conf.KubeImagePrefix, "kube-scheduler", conf.KubeVersion)
	}
	conf.KubeSchedulerImage = envs.GetEnvWithPrefix("KUBE_SCHEDULER_IMAGE", conf.KubeSchedulerImage)

	if conf.KubectlImage == "" {
		conf.KubectlImage = joinImageURI(conf.KubeImagePrefix, "kubectl", conf.KubeVersion)
	}
	conf.KubectlImage = envs.GetEnvWithPrefix("KUBECTL_IMAGE", conf.KubectlImage)

	conf.KubeSchedulerPort = envs.GetEnvWithPrefix("KUBE_SCHEDULER_PORT", conf.KubeSchedulerPort)
}

func setKwokctlKwokConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	if conf.KwokBinaryPrefix == "" {
		conf.KwokBinaryPrefix = consts.BinaryPrefix + "/" + conf.KwokVersion
	}
	conf.KwokBinaryPrefix = envs.GetEnvWithPrefix("BINARY_PREFIX", conf.KwokBinaryPrefix)

	if conf.KwokControllerBinary == "" {
		conf.KwokControllerBinary = conf.KwokBinaryPrefix + "/kwok-" + GOOS + "-" + GOARCH + conf.BinSuffix
	}
	conf.KwokControllerBinary = envs.GetEnvWithPrefix("CONTROLLER_BINARY", conf.KwokControllerBinary)

	if conf.KwokImagePrefix == "" {
		conf.KwokImagePrefix = consts.ImagePrefix
	}
	conf.KwokImagePrefix = envs.GetEnvWithPrefix("IMAGE_PREFIX", conf.KwokImagePrefix)

	if conf.KwokControllerImage == "" {
		conf.KwokControllerImage = joinImageURI(conf.KwokImagePrefix, "kwok", conf.KwokVersion)
	}
	conf.KwokControllerImage = envs.GetEnvWithPrefix("CONTROLLER_IMAGE", conf.KwokControllerImage)
	conf.KwokControllerPort = envs.GetEnvWithPrefix("CONTROLLER_PORT", conf.KwokControllerPort)
}

func setKwokctlEtcdConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	if conf.EtcdVersion == "" {
		conf.EtcdVersion = k8s.GetEtcdVersion(parseRelease(conf.KubeVersion))
	}
	conf.EtcdVersion = version.TrimPrefixV(envs.GetEnvWithPrefix("ETCD_VERSION", conf.EtcdVersion))

	if conf.EtcdBinaryPrefix == "" {
		conf.EtcdBinaryPrefix = consts.EtcdBinaryPrefix + "/v" + strings.TrimSuffix(conf.EtcdVersion, "-0")
	}
	conf.EtcdBinaryPrefix = envs.GetEnvWithPrefix("ETCD_BINARY_PREFIX", conf.EtcdBinaryPrefix)

	conf.EtcdBinary = envs.GetEnvWithPrefix("ETCD_BINARY", conf.EtcdBinary)

	if conf.EtcdBinaryTar == "" {
		conf.EtcdBinaryTar = conf.EtcdBinaryPrefix + "/etcd-v" + strings.TrimSuffix(conf.EtcdVersion, "-0") + "-" + GOOS + "-" + GOARCH + "." + func() string {
			if GOOS == linux {
				return binarySuffixTar
			}
			return binarySuffixZip
		}()
	}
	conf.EtcdBinaryTar = envs.GetEnvWithPrefix("ETCD_BINARY_TAR", conf.EtcdBinaryTar)

	if conf.EtcdImagePrefix == "" {
		conf.EtcdImagePrefix = conf.KubeImagePrefix
	}
	conf.EtcdImagePrefix = envs.GetEnvWithPrefix("ETCD_IMAGE_PREFIX", conf.EtcdImagePrefix)

	if conf.EtcdImage == "" {
		conf.EtcdImage = joinImageURI(conf.EtcdImagePrefix, "etcd", conf.EtcdVersion)
	}
	conf.EtcdImage = envs.GetEnvWithPrefix("ETCD_IMAGE", conf.EtcdImage)

	conf.EtcdPort = envs.GetEnvWithPrefix("ETCD_PORT", conf.EtcdPort)

	if conf.EtcdBinary == "" {
		conf.EtcdBinary = conf.EtcdBinaryTar + "#etcd" + conf.BinSuffix
	}

	if conf.EtcdctlBinary == "" {
		conf.EtcdctlBinary = conf.EtcdBinaryTar + "#etcdctl" + conf.BinSuffix
	}
}

func setKwokctlKindConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	if conf.KindNodeImagePrefix == "" {
		conf.KindNodeImagePrefix = consts.KindNodeImagePrefix
	}
	conf.KindNodeImagePrefix = envs.GetEnvWithPrefix("KIND_NODE_IMAGE_PREFIX", conf.KindNodeImagePrefix)

	if conf.KindNodeImage == "" {
		conf.KindNodeImage = joinImageURI(conf.KindNodeImagePrefix, "node", conf.KubeVersion)
	}
	conf.KindNodeImage = envs.GetEnvWithPrefix("KIND_NODE_IMAGE", conf.KindNodeImage)

	if conf.KindVersion == "" {
		conf.KindVersion = consts.KindVersion
	}
	conf.KindVersion = version.AddPrefixV(envs.GetEnvWithPrefix("KIND_VERSION", conf.KindVersion))

	if conf.KindBinaryPrefix == "" {
		conf.KindBinaryPrefix = consts.KindBinaryPrefix + "/" + conf.KindVersion
	}
	conf.KindBinaryPrefix = envs.GetEnvWithPrefix("KIND_BINARY_PREFIX", conf.KindBinaryPrefix)

	if conf.KindBinary == "" {
		conf.KindBinary = conf.KindBinaryPrefix + "/kind-" + GOOS + "-" + GOARCH + conf.BinSuffix
	}
	conf.KindBinary = envs.GetEnvWithPrefix("KIND_BINARY", conf.KindBinary)
}

func setKwokctlDashboardConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	if conf.DashboardVersion == "" {
		conf.DashboardVersion = consts.DashboardVersion
	}
	conf.DashboardVersion = version.AddPrefixV(envs.GetEnvWithPrefix("DASHBOARD_VERSION", conf.DashboardVersion))

	if conf.DashboardImagePrefix == "" {
		conf.DashboardImagePrefix = consts.DashboardImagePrefix
	}
	conf.DashboardImagePrefix = envs.GetEnvWithPrefix("DASHBOARD_IMAGE_PREFIX", conf.DashboardImagePrefix)

	if conf.DashboardImage == "" {
		conf.DashboardImage = joinImageURI(conf.DashboardImagePrefix, "dashboard", conf.DashboardVersion)
	}
	conf.DashboardImage = envs.GetEnvWithPrefix("DASHBOARD_IMAGE", conf.DashboardImage)

	if conf.DashboardMetricsScraperVersion == "" {
		conf.DashboardMetricsScraperVersion = consts.DashboardMetricsScraperVersion
	}
	conf.DashboardMetricsScraperVersion = version.AddPrefixV(envs.GetEnvWithPrefix("DASHBOARD_METRICS_SCRAPER_VERSION", conf.DashboardMetricsScraperVersion))

	if conf.DashboardMetricsScraperImage == "" {
		conf.DashboardMetricsScraperImage = joinImageURI(conf.DashboardImagePrefix, "metrics-scraper", conf.DashboardMetricsScraperVersion)
	}
	conf.DashboardMetricsScraperImage = envs.GetEnvWithPrefix("DASHBOARD_METRICS_SCRAPER_IMAGE", conf.DashboardMetricsScraperImage)

	// TODO: Add dashboard binary
	// if conf.DashboardBinaryPrefix == "" {
	// 	conf.DashboardBinaryPrefix = consts.DashboardBinaryPrefix + "/" + conf.DashboardVersion
	// }
	// conf.DashboardBinaryPrefix = envs.GetEnvWithPrefix("DASHBOARD_BINARY_PREFIX", conf.DashboardBinaryPrefix)\
	//
	// if conf.DashboardBinary == "" {
	// 	conf.DashboardBinary = conf.DashboardBinaryPrefix + "/dashboard-" + GOOS + "-" + GOARCH + conf.BinSuffix
	// }
	// conf.DashboardBinary = envs.GetEnvWithPrefix("DASHBOARD_BINARY", conf.DashboardBinary)
}

func setKwokctlPrometheusConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	conf.PrometheusPort = envs.GetEnvWithPrefix("PROMETHEUS_PORT", conf.PrometheusPort)

	if conf.PrometheusVersion == "" {
		conf.PrometheusVersion = consts.PrometheusVersion
	}
	conf.PrometheusVersion = version.AddPrefixV(envs.GetEnvWithPrefix("PROMETHEUS_VERSION", conf.PrometheusVersion))

	if conf.PrometheusImagePrefix == "" {
		conf.PrometheusImagePrefix = consts.PrometheusImagePrefix
	}
	conf.PrometheusImagePrefix = envs.GetEnvWithPrefix("PROMETHEUS_IMAGE_PREFIX", conf.PrometheusImagePrefix)

	if conf.PrometheusImage == "" {
		conf.PrometheusImage = joinImageURI(conf.PrometheusImagePrefix, "prometheus", conf.PrometheusVersion)
	}
	conf.PrometheusImage = envs.GetEnvWithPrefix("PROMETHEUS_IMAGE", conf.PrometheusImage)

	if conf.PrometheusBinaryPrefix == "" {
		conf.PrometheusBinaryPrefix = consts.PrometheusBinaryPrefix + "/" + conf.PrometheusVersion
	}
	conf.PrometheusBinaryPrefix = envs.GetEnvWithPrefix("PROMETHEUS_BINARY_PREFIX", conf.PrometheusBinaryPrefix)

	conf.PrometheusBinary = envs.GetEnvWithPrefix("PROMETHEUS_BINARY", conf.PrometheusBinary)

	if conf.PrometheusBinaryTar == "" {
		conf.PrometheusBinaryTar = conf.PrometheusBinaryPrefix + "/prometheus-" + strings.TrimPrefix(conf.PrometheusVersion, "v") + "." + GOOS + "-" + GOARCH + "." + func() string {
			if GOOS == windows {
				return binarySuffixZip
			}
			return binarySuffixTar
		}()
	}
	conf.PrometheusBinaryTar = envs.GetEnvWithPrefix("PROMETHEUS_BINARY_TAR", conf.PrometheusBinaryTar)

	if conf.PrometheusBinary == "" {
		conf.PrometheusBinary = conf.PrometheusBinaryTar + "#prometheus" + conf.BinSuffix
	}
}

func setKwokctlJaegerConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	conf.JaegerPort = envs.GetEnvWithPrefix("JAEGER_PORT", conf.JaegerPort)

	if conf.JaegerVersion == "" {
		conf.JaegerVersion = consts.JaegerVersion
	}
	conf.JaegerVersion = version.AddPrefixV(envs.GetEnvWithPrefix("JAEGER_VERSION", conf.JaegerVersion))

	if conf.JaegerImagePrefix == "" {
		conf.JaegerImagePrefix = consts.JaegerImagePrefix
	}
	conf.JaegerImagePrefix = envs.GetEnvWithPrefix("JAEGER_IMAGE_PREFIX", conf.JaegerImagePrefix)

	if conf.JaegerImage == "" {
		conf.JaegerImage = joinImageURI(conf.JaegerImagePrefix, "all-in-one", strings.TrimPrefix(conf.JaegerVersion, "v"))
	}
	conf.JaegerImage = envs.GetEnvWithPrefix("JAEGER_IMAGE", conf.JaegerImage)

	if conf.JaegerBinaryPrefix == "" {
		conf.JaegerBinaryPrefix = consts.JaegerBinaryPrefix + "/" + conf.JaegerVersion
	}
	conf.JaegerBinaryPrefix = envs.GetEnvWithPrefix("JAEGER_BINARY_PREFIX", conf.JaegerBinaryPrefix)

	conf.JaegerBinary = envs.GetEnvWithPrefix("JAEGER_BINARY", conf.JaegerBinary)

	if conf.JaegerBinaryTar == "" {
		conf.JaegerBinaryTar = conf.JaegerBinaryPrefix + "/jaeger-" + strings.TrimPrefix(conf.JaegerVersion, "v") + "-" + GOOS + "-" + GOARCH + "." + func() string {
			if GOOS == windows {
				return binarySuffixZip
			}
			return binarySuffixTar
		}()
	}
	conf.JaegerBinaryTar = envs.GetEnvWithPrefix("JAEGER_BINARY_TAR", conf.JaegerBinaryTar)

	if conf.JaegerBinary == "" {
		conf.JaegerBinary = conf.JaegerBinaryTar + "#jaeger-all-in-one" + conf.BinSuffix
	}
}

func setMetricsServerConfig(conf *configv1alpha1.KwokctlConfigurationOptions) {
	if conf.MetricsServerVersion == "" {
		conf.MetricsServerVersion = consts.MetricsServerVersion
	}
	conf.MetricsServerVersion = version.AddPrefixV(envs.GetEnvWithPrefix("METRICS_SERVER_VERSION", conf.MetricsServerVersion))

	if conf.MetricsServerImagePrefix == "" {
		conf.MetricsServerImagePrefix = consts.MetricsServerImagePrefix
	}
	conf.MetricsServerImagePrefix = envs.GetEnvWithPrefix("METRICS_SERVER_IMAGE_PREFIX", conf.MetricsServerImagePrefix)

	if conf.MetricsServerImage == "" {
		conf.MetricsServerImage = joinImageURI(conf.MetricsServerImagePrefix, "metrics-server", version.AddPrefixV(conf.MetricsServerVersion))
	}
	conf.MetricsServerImage = envs.GetEnvWithPrefix("METRICS_SERVER_IMAGE", conf.MetricsServerImage)

	if conf.MetricsServerBinaryPrefix == "" {
		conf.MetricsServerBinaryPrefix = consts.MetricsServerBinaryPrefix + "/" + conf.MetricsServerVersion
	}
	conf.MetricsServerBinaryPrefix = envs.GetEnvWithPrefix("METRICS_SERVER_BINARY_PREFIX", conf.MetricsServerBinaryPrefix)

	if conf.MetricsServerBinaryPrefix != "" &&
		conf.MetricsServerBinary == "" {
		conf.MetricsServerBinary = conf.MetricsServerBinaryPrefix + "/metrics-server-" + GOOS + "-" + GOARCH + conf.BinSuffix
	}
	conf.MetricsServerBinary = envs.GetEnvWithPrefix("METRICS_SERVER_BINARY", conf.MetricsServerBinary)
}

// joinImageURI joins the image URI.
func joinImageURI(prefix, name, version string) string {
	return prefix + "/" + name + ":" + version
}

// parseRelease returns the release of the version.
func parseRelease(ver string) int {
	v, err := version.ParseVersion(ver)
	if err != nil {
		return -1
	}
	return int(v.Minor)
}
