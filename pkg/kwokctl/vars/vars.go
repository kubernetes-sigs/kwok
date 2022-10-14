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

package vars

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
)

var (
	// ProjectName is the name of the project.
	ProjectName = "kwok"

	// DefaultCluster the default cluster name
	DefaultCluster = "kwok"

	// WorkDir is the directory of the work spaces.
	WorkDir = getEnv("KWOK_WORKDIR", func() string {
		dir, err := os.UserHomeDir()
		if err != nil || dir == "" {
			return utils.PathJoin(os.TempDir(), ProjectName)
		}
		return utils.PathJoin(dir, "."+ProjectName)
	}())

	// ClustersDir is the directory of the clusters.
	ClustersDir = utils.PathJoin(WorkDir, "clusters")

	// CacheDir is the directory of the cache.
	CacheDir = utils.PathJoin(WorkDir, "cache")

	// KubeApiserverPort is the port to expose apiserver.
	KubeApiserverPort = getEnvInt("KWOK_KUBE_APISERVER_PORT", 0)

	// Runtime is the runtime to use.
	Runtime = getEnv("KWOK_RUNTIME", "docker")

	// PrometheusPort is the port to expose Prometheus metrics.
	PrometheusPort = getEnvInt("KWOK_PROMETHEUS_PORT", 0)

	// KwokVersion is the version of the fake to use.
	KwokVersion = addPrefixV(getEnv("KWOK_VERSION", consts.Version))

	// KubeVersion is the version of Kubernetes to use.
	KubeVersion = addPrefixV(getEnv("KWOK_KUBE_VERSION", consts.KubeVersion))

	// EtcdVersion is the version of etcd to use.
	EtcdVersion = trimPrefixV(getEnv("KWOK_ETCD_VERSION", k8s.GetEtcdVersion(parseRelease(KubeVersion))))

	// PrometheusVersion is the version of Prometheus to use.
	PrometheusVersion = addPrefixV(getEnv("KWOK_PROMETHEUS_VERSION", "v2.35.0"))

	// SecurePort is the Apiserver use TLS.
	SecurePort = getEnvBool("KWOK_SECURE_PORT", parseRelease(KubeVersion) > 19)

	// QuietPull is the flag to quiet the pull.
	QuietPull = getEnvBool("KWOK_QUIET_PULL", false)

	// KubeImagePrefix is the prefix of the kubernetes image.
	KubeImagePrefix = getEnv("KWOK_KUBE_IMAGE_PREFIX", "registry.k8s.io")

	// EtcdImagePrefix is the prefix of the etcd image.
	EtcdImagePrefix = getEnv("KWOK_ETCD_IMAGE_PREFIX", KubeImagePrefix)

	// KwokImagePrefix is the prefix of the fake image.
	KwokImagePrefix = getEnv("KWOK_IMAGE_PREFIX", consts.ImagePrefix)

	// PrometheusImagePrefix is the prefix of the Prometheus image.
	PrometheusImagePrefix = getEnv("KWOK_PROMETHEUS_IMAGE_PREFIX", "docker.io/prom")

	// EtcdImage is the image of etcd.
	EtcdImage = getEnv("KWOK_ETCD_IMAGE", joinImageURI(EtcdImagePrefix, "etcd", EtcdVersion))

	// KubeApiserverImage is the image of kube-apiserver.
	KubeApiserverImage = getEnv("KWOK_KUBE_APISERVER_IMAGE", joinImageURI(KubeImagePrefix, "kube-apiserver", KubeVersion))

	// KubeControllerManagerImage is the image of kube-controller-manager.
	KubeControllerManagerImage = getEnv("KWOK_KUBE_CONTROLLER_MANAGER_IMAGE", joinImageURI(KubeImagePrefix, "kube-controller-manager", KubeVersion))

	// KubeSchedulerImage is the image of kube-scheduler.
	KubeSchedulerImage = getEnv("KWOK_KUBE_SCHEDULER_IMAGE", joinImageURI(KubeImagePrefix, "kube-scheduler", KubeVersion))

	// KwokControllerImage is the image of kwok.
	KwokControllerImage = getEnv("KWOK_CONTROLLER_IMAGE", joinImageURI(KwokImagePrefix, "kwok", KwokVersion))

	// PrometheusImage is the image of Prometheus.
	PrometheusImage = getEnv("KWOK_PROMETHEUS_IMAGE", joinImageURI(PrometheusImagePrefix, "prometheus", PrometheusVersion))

	// KindNodeImagePrefix is the prefix of the kind node image.
	KindNodeImagePrefix = getEnv("KWOK_KIND_NODE_IMAGE_PREFIX", "docker.io/kindest")

	// KindNodeImage is the image of kind node.
	KindNodeImage = getEnv("KWOK_KIND_NODE_IMAGE", joinImageURI(KindNodeImagePrefix, "node", KubeVersion))

	// KubeBinaryPrefix is the prefix of the kubernetes binary.
	KubeBinaryPrefix = getEnv("KWOK_KUBE_BINARY_PREFIX", "https://dl.k8s.io/release/"+KubeVersion+"/bin/"+runtime.GOOS+"/"+runtime.GOARCH)

	// KubeApiserverBinary is the binary of kube-apiserver.
	KubeApiserverBinary = getEnv("KWOK_KUBE_APISERVER_BINARY", KubeBinaryPrefix+"/kube-apiserver"+BinSuffix)

	// KubeControllerManagerBinary is the binary of kube-controller-manager.
	KubeControllerManagerBinary = getEnv("KWOK_KUBE_CONTROLLER_MANAGER_BINARY", KubeBinaryPrefix+"/kube-controller-manager"+BinSuffix)

	// KubeSchedulerBinary is the binary of kube-scheduler.
	KubeSchedulerBinary = getEnv("KWOK_KUBE_SCHEDULER_BINARY", KubeBinaryPrefix+"/kube-scheduler"+BinSuffix)

	// MustKubectlBinary is the binary of kubectl.
	MustKubectlBinary = "https://dl.k8s.io/release/" + KubeVersion + "/bin/" + runtime.GOOS + "/" + runtime.GOARCH + "/kubectl" + BinSuffix

	// KubectlBinary is the binary of kubectl.
	KubectlBinary = getEnv("KWOK_KUBECTL_BINARY", KubeBinaryPrefix+"/kubectl"+BinSuffix)

	// EtcdBinaryPrefix is the prefix of the etcd binary.
	EtcdBinaryPrefix = getEnv("KWOK_ETCD_BINARY_PREFIX", "https://github.com/etcd-io/etcd/releases/download")

	// EtcdBinary is the binary of etcd.
	EtcdBinary = getEnv("KWOK_ETCD_BINARY", "")

	// EtcdBinaryTar is the tar of the binary of etcd.
	EtcdBinaryTar = getEnv("KWOK_ETCD_BINARY_TAR", EtcdBinaryPrefix+"/v"+strings.TrimSuffix(EtcdVersion, "-0")+"/etcd-v"+strings.TrimSuffix(EtcdVersion, "-0")+"-"+runtime.GOOS+"-"+runtime.GOARCH+"."+func() string {
		if runtime.GOOS == "linux" {
			return "tar.gz"
		}
		return "zip"
	}())

	// KwokBinaryPrefix is the prefix of the kwok binary.
	KwokBinaryPrefix = getEnv("KWOK_BINARY_PREFIX", consts.BinaryPrefix)

	// KwokControllerBinary is the binary of kwok.
	KwokControllerBinary = getEnv("KWOK_CONTROLLER_BINARY", KwokBinaryPrefix+"/"+KwokVersion+"/"+consts.BinaryName+BinSuffix)

	// PrometheusBinaryPrefix is the prefix of the Prometheus binary.
	PrometheusBinaryPrefix = getEnv("KWOK_PROMETHEUS_BINARY_PREFIX", "https://github.com/prometheus/prometheus/releases/download")

	// PrometheusBinary  is the binary of Prometheus.
	PrometheusBinary = getEnv("KWOK_PROMETHEUS_BINARY", "")

	// PrometheusBinaryTar is the tar of binary of Prometheus.
	PrometheusBinaryTar = getEnv("KWOK_PROMETHEUS_BINARY_TAR", PrometheusBinaryPrefix+"/"+PrometheusVersion+"/prometheus-"+strings.TrimPrefix(PrometheusVersion, "v")+"."+runtime.GOOS+"-"+runtime.GOARCH+"."+func() string {
		if runtime.GOOS == "windows" {
			return "zip"
		}
		return "tar.gz"
	}())

	BinSuffix = func() string {
		if runtime.GOOS == "windows" {
			return ".exe"
		}
		return ""
	}()

	// Mode is several default parameter templates for clusters
	Mode = getEnv("KWOK_MODE", "")

	// ModeStableFeatureGateAndAPI is intended to reduce cluster configuration requirements
	// Disables all Alpha feature by default, as well as Beta feature that are not eventually GA
	ModeStableFeatureGateAndAPI = "StableFeatureGateAndAPI"

	// KubeFeatureGates is a set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes.
	KubeFeatureGates = getEnv("KWOK_KUBE_FEATURE_GATES", func() string {
		if Mode == ModeStableFeatureGateAndAPI {
			return k8s.GetFeatureGates(parseRelease(KubeVersion))
		}
		return ""
	}())

	// KubeRuntimeConfig is a set of key=value pairs that enable or disable built-in APIs.
	KubeRuntimeConfig = getEnv("KWOK_KUBE_RUNTIME_CONFIG", func() string {
		if Mode == ModeStableFeatureGateAndAPI {
			return k8s.GetRuntimeConfig(parseRelease(KubeVersion))
		}
		return ""
	}())
)

// getEnv returns the value of the environment variable named by the key.
func getEnv(key, def string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return value
}

// getEnvInt returns the value of the environment variable named by the key.
func getEnvInt(key string, def int) int {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// getEnvBool returns the value of the environment variable named by the key.
func getEnvBool(key string, def bool) bool {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// joinImageURI joins the image URI.
func joinImageURI(prefix, name, version string) string {
	return prefix + "/" + name + ":" + version
}

// parseRelease returns the release of the version.
func parseRelease(version string) int {
	release := strings.Split(version, ".")
	if len(release) < 2 {
		return 0
	}
	r, err := strconv.ParseInt(release[1], 10, 64)
	if err != nil {
		return 0
	}
	return int(r)
}

// trimPrefixV returns the version without the prefix 'v'.
func trimPrefixV(version string) string {
	if len(version) <= 1 {
		return version
	}

	if version[0] != 'v' ||
		version[1] < '0' || version[1] > '9' {
		return version
	}
	return version[1:]
}

// addPrefixV returns the version with the prefix 'v'.
func addPrefixV(version string) string {
	if version == "" {
		return version
	}

	if version[0] < '0' || version[0] > '9' {
		return version
	}
	return "v" + version
}
