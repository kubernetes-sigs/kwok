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
	"strconv"
	"strings"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/envs"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

var (
	// DefaultCluster the default cluster name
	DefaultCluster = "kwok"

	// WorkDir is the directory of the work spaces.
	WorkDir = envs.GetEnvWithPrefix("WORKDIR", path.WorkDir())

	// ClustersDir is the directory of the clusters.
	ClustersDir = path.Join(WorkDir, "clusters")
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
				"kind", v1alpha1.KwokctlConfigurationKind,
			)
		}
	}
	if conf == nil {
		logger := log.FromContext(ctx)
		logger.Debug("No configuration",
			"kind", v1alpha1.KwokctlConfigurationKind,
		)
		conf, err := internalversion.ConvertToInternalVersionKwokctlConfiguration(setKwokctlConfigurationDefaults(&v1alpha1.KwokctlConfiguration{}))
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
				"kind", v1alpha1.KwokConfigurationKind,
			)
		}
	}
	if conf == nil {
		logger := log.FromContext(ctx)
		logger.Debug("No configuration",
			"kind", v1alpha1.KwokConfigurationKind,
		)
		conf, err := internalversion.ConvertToInternalVersionKwokConfiguration(setKwokConfigurationDefaults(&v1alpha1.KwokConfiguration{}))
		if err != nil {
			logger.Error("Get kwok configuration failed", err)
			return &internalversion.KwokConfiguration{}
		}
		addToContext(ctx, conf)
		return conf
	}
	return conf
}

func setStageDefaults(config *v1alpha1.Stage) *v1alpha1.Stage {
	if config == nil {
		config = &v1alpha1.Stage{}
	}
	v1alpha1.SetObjectDefaults_Stage(config)
	return config
}

func setKwokConfigurationDefaults(config *v1alpha1.KwokConfiguration) *v1alpha1.KwokConfiguration {
	if config == nil {
		config = &v1alpha1.KwokConfiguration{}
	}

	v1alpha1.SetObjectDefaults_KwokConfiguration(config)

	return config
}

func setKwokctlConfigurationDefaults(config *v1alpha1.KwokctlConfiguration) *v1alpha1.KwokctlConfiguration {
	if config == nil {
		config = &v1alpha1.KwokctlConfiguration{}
	}
	conf := &config.Options

	if conf.KwokVersion == "" {
		conf.KwokVersion = consts.Version
	}
	conf.KwokVersion = addPrefixV(envs.GetEnvWithPrefix("VERSION", conf.KwokVersion))

	if conf.KubeVersion == "" {
		conf.KubeVersion = consts.KubeVersion
	}
	conf.KubeVersion = addPrefixV(envs.GetEnvWithPrefix("KUBE_VERSION", conf.KubeVersion))

	if conf.SecurePort == nil {
		conf.SecurePort = format.Ptr(parseRelease(conf.KubeVersion) > 12)
	}
	conf.SecurePort = format.Ptr(envs.GetEnvWithPrefix("SECURE_PORT", *conf.SecurePort))

	if conf.QuietPull == nil {
		conf.QuietPull = format.Ptr(false)
	}
	conf.QuietPull = format.Ptr(envs.GetEnvWithPrefix("QUIET_PULL", *conf.QuietPull))

	conf.Runtime = envs.GetEnvWithPrefix("RUNTIME", conf.Runtime)
	if len(conf.Runtimes) == 0 {
		conf.Runtimes = []string{
			consts.RuntimeTypeDocker,
			consts.RuntimeTypeNerdctl,
		}
		if runtime.GOOS == "linux" {
			conf.Runtimes = append(conf.Runtimes, consts.RuntimeTypeBinary)
		}
	}

	conf.Mode = envs.GetEnvWithPrefix("MODE", conf.Mode)

	if conf.CacheDir == "" {
		conf.CacheDir = path.Join(WorkDir, "cache")
	}

	if conf.BinSuffix == "" {
		if runtime.GOOS == "windows" {
			conf.BinSuffix = ".exe"
		}
	}

	setKwokctlKubernetesConfig(conf)

	setKwokctlKwokConfig(conf)

	setKwokctlEtcdConfig(conf)

	setKwokctlKindConfig(conf)

	setKwokctlDockerConfig(conf)

	setKwokctlPrometheusConfig(conf)

	v1alpha1.SetObjectDefaults_KwokctlConfiguration(config)

	return config
}

func setKwokctlKubernetesConfig(conf *v1alpha1.KwokctlConfigurationOptions) {
	if conf.DisableKubeScheduler == nil {
		conf.DisableKubeScheduler = format.Ptr(false)
	}
	conf.DisableKubeScheduler = format.Ptr(envs.GetEnvWithPrefix("DISABLE_KUBE_SCHEDULER", *conf.DisableKubeScheduler))

	if conf.DisableKubeControllerManager == nil {
		conf.DisableKubeControllerManager = format.Ptr(false)
	}
	conf.DisableKubeControllerManager = format.Ptr(envs.GetEnvWithPrefix("DISABLE_KUBE_CONTROLLER_MANAGER", *conf.DisableKubeControllerManager))

	if conf.KubeAuthorization == nil {
		conf.KubeAuthorization = format.Ptr(false)
	}
	conf.KubeAuthorization = format.Ptr(envs.GetEnvWithPrefix("KUBE_AUTHORIZATION", *conf.KubeAuthorization))

	conf.KubeApiserverPort = envs.GetEnvWithPrefix("KUBE_APISERVER_PORT", conf.KubeApiserverPort)

	if conf.KubeFeatureGates == "" {
		if conf.Mode == v1alpha1.ModeStableFeatureGateAndAPI {
			conf.KubeFeatureGates = k8s.GetFeatureGates(parseRelease(conf.KubeVersion))
		}
	}
	conf.KubeFeatureGates = envs.GetEnvWithPrefix("KUBE_FEATURE_GATES", conf.KubeFeatureGates)

	if conf.KubeRuntimeConfig == "" {
		if conf.Mode == v1alpha1.ModeStableFeatureGateAndAPI {
			conf.KubeRuntimeConfig = k8s.GetRuntimeConfig(parseRelease(conf.KubeVersion))
		}
	}
	conf.KubeRuntimeConfig = envs.GetEnvWithPrefix("KUBE_RUNTIME_CONFIG", conf.KubeRuntimeConfig)

	conf.KubeAuditPolicy = envs.GetEnvWithPrefix("KUBE_AUDIT_POLICY", conf.KubeAuditPolicy)

	if conf.KubeBinaryPrefix == "" {
		conf.KubeBinaryPrefix = consts.KubeBinaryPrefix + "/" + conf.KubeVersion + "/bin/" + runtime.GOOS + "/" + runtime.GOARCH
	}
	conf.KubeBinaryPrefix = envs.GetEnvWithPrefix("KUBE_BINARY_PREFIX", conf.KubeBinaryPrefix)

	if conf.KubectlBinary == "" {
		conf.KubectlBinary = conf.KubeBinaryPrefix + "/kubectl" + conf.BinSuffix
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

	if conf.KubeSchedulerImage == "" {
		conf.KubeSchedulerImage = joinImageURI(conf.KubeImagePrefix, "kube-scheduler", conf.KubeVersion)
	}
	conf.KubeSchedulerImage = envs.GetEnvWithPrefix("KUBE_SCHEDULER_IMAGE", conf.KubeSchedulerImage)

	conf.KubeSchedulerPort = envs.GetEnvWithPrefix("KUBE_SCHEDULER_PORT", conf.KubeSchedulerPort)
}

func setKwokctlKwokConfig(conf *v1alpha1.KwokctlConfigurationOptions) {
	if conf.KwokBinaryPrefix == "" {
		conf.KwokBinaryPrefix = consts.BinaryPrefix
	}
	conf.KwokBinaryPrefix = envs.GetEnvWithPrefix("BINARY_PREFIX", conf.KwokBinaryPrefix+"/"+conf.KwokVersion)

	if conf.KwokControllerBinary == "" {
		conf.KwokControllerBinary = conf.KwokBinaryPrefix + "/kwok-" + runtime.GOOS + "-" + runtime.GOARCH + conf.BinSuffix
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

func setKwokctlEtcdConfig(conf *v1alpha1.KwokctlConfigurationOptions) {
	if conf.EtcdVersion == "" {
		conf.EtcdVersion = k8s.GetEtcdVersion(parseRelease(conf.KubeVersion))
	}
	conf.EtcdVersion = trimPrefixV(envs.GetEnvWithPrefix("ETCD_VERSION", conf.EtcdVersion))

	if conf.EtcdBinaryPrefix == "" {
		conf.EtcdBinaryPrefix = consts.EtcdBinaryPrefix + "/v" + strings.TrimSuffix(conf.EtcdVersion, "-0")
	}
	conf.EtcdBinaryPrefix = envs.GetEnvWithPrefix("ETCD_BINARY_PREFIX", conf.EtcdBinaryPrefix)

	conf.EtcdBinary = envs.GetEnvWithPrefix("ETCD_BINARY", conf.EtcdBinary)

	if conf.EtcdBinaryTar == "" {
		conf.EtcdBinaryTar = conf.EtcdBinaryPrefix + "/etcd-v" + strings.TrimSuffix(conf.EtcdVersion, "-0") + "-" + runtime.GOOS + "-" + runtime.GOARCH + "." + func() string {
			if runtime.GOOS == "linux" {
				return "tar.gz"
			}
			return "zip"
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
}

func setKwokctlKindConfig(conf *v1alpha1.KwokctlConfigurationOptions) {
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
	conf.KindVersion = addPrefixV(envs.GetEnvWithPrefix("KIND_VERSION", conf.KindVersion))

	if conf.KindBinaryPrefix == "" {
		conf.KindBinaryPrefix = consts.KindBinaryPrefix + "/" + conf.KindVersion
	}
	conf.KindBinaryPrefix = envs.GetEnvWithPrefix("KIND_BINARY_PREFIX", conf.KindBinaryPrefix)

	if conf.KindBinary == "" {
		conf.KindBinary = conf.KindBinaryPrefix + "/kind-" + runtime.GOOS + "-" + runtime.GOARCH + conf.BinSuffix
	}
	conf.KindBinary = envs.GetEnvWithPrefix("KIND_BINARY", conf.KindBinary)
}

func setKwokctlDockerConfig(conf *v1alpha1.KwokctlConfigurationOptions) {
	if conf.DockerComposeVersion == "" {
		conf.DockerComposeVersion = consts.DockerComposeVersion
	}
	conf.DockerComposeVersion = addPrefixV(envs.GetEnvWithPrefix("DOCKER_COMPOSE_VERSION", conf.DockerComposeVersion))

	if conf.DockerComposeBinaryPrefix == "" {
		conf.DockerComposeBinaryPrefix = consts.DockerComposeBinaryPrefix + "/" + conf.DockerComposeVersion
	}
	conf.DockerComposeBinaryPrefix = envs.GetEnvWithPrefix("DOCKER_COMPOSE_BINARY_PREFIX", conf.DockerComposeBinaryPrefix)

	if conf.DockerComposeBinary == "" {
		conf.DockerComposeBinary = conf.DockerComposeBinaryPrefix + "/docker-compose-" + runtime.GOOS + "-" + archAlias(runtime.GOARCH) + conf.BinSuffix
	}
	conf.DockerComposeBinary = envs.GetEnvWithPrefix("DOCKER_COMPOSE_BINARY", conf.DockerComposeBinary)
}

func setKwokctlPrometheusConfig(conf *v1alpha1.KwokctlConfigurationOptions) {
	conf.PrometheusPort = envs.GetEnvWithPrefix("PROMETHEUS_PORT", conf.PrometheusPort)

	if conf.PrometheusVersion == "" {
		conf.PrometheusVersion = consts.PrometheusVersion
	}
	conf.PrometheusVersion = addPrefixV(envs.GetEnvWithPrefix("PROMETHEUS_VERSION", conf.PrometheusVersion))

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
		conf.PrometheusBinaryTar = conf.PrometheusBinaryPrefix + "/prometheus-" + strings.TrimPrefix(conf.PrometheusVersion, "v") + "." + runtime.GOOS + "-" + runtime.GOARCH + "." + func() string {
			if runtime.GOOS == "windows" {
				return "zip"
			}
			return "tar.gz"
		}()
	}
	conf.PrometheusBinaryTar = envs.GetEnvWithPrefix("PROMETHEUS_BINARY_TAR", conf.PrometheusBinaryTar)
}

// joinImageURI joins the image URI.
func joinImageURI(prefix, name, version string) string {
	return prefix + "/" + name + ":" + version
}

// parseRelease returns the release of the version.
func parseRelease(version string) int {
	release := strings.Split(version, ".")
	if len(release) < 2 {
		return -1
	}
	r, err := strconv.ParseInt(release[1], 10, 64)
	if err != nil {
		return -1
	}
	return int(r)
}

// trimPrefixV returns the version without the prefix 'v'.
func trimPrefixV(version string) string {
	if len(version) <= 1 {
		return version
	}

	// Not a semantic version or unprefixed 'v'
	if version[0] != 'v' ||
		!strings.Contains(version, ".") ||
		version[1] < '0' ||
		version[1] > '9' {
		return version
	}
	return version[1:]
}

// addPrefixV returns the version with the prefix 'v'.
func addPrefixV(version string) string {
	if version == "" {
		return version
	}

	// Not a semantic version or prefixed 'v'
	if !strings.Contains(version, ".") ||
		version[0] < '0' ||
		version[0] > '9' {
		return version
	}
	return "v" + version
}

var archMapping = map[string]string{
	"arm64": "aarch64",
	"arm":   "armv7",
	"amd64": "x86_64",
	"386":   "x86",
}

// archAlias returns the alias of the given arch
func archAlias(arch string) string {
	if v, ok := archMapping[arch]; ok {
		return v
	}
	return arch
}
