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

package kind

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

func (c *Cluster) addKind(ctx context.Context, env *env) (err error) {
	logger := log.FromContext(ctx)
	conf := &env.kwokctlConfig.Options

	err = c.EnsureImage(ctx, c.runtime, conf.KindNodeImage)
	if err != nil {
		logger.Warn("Failed to ensure kind node image, attempting to build locally",
			"image", conf.KindNodeImage,
			"kubeVersion", conf.KubeVersion,
			"err", err,
		)

		buildErr := c.buildKindNodeImage(ctx, conf.KindNodeImage, conf.KubeVersion)
		if buildErr != nil {
			return fmt.Errorf("failed to ensure and build kind node image %q: %w", conf.KindNodeImage, errors.Join(err, buildErr))
		}
	}

	var featureGates []string
	var runtimeConfig []string
	if conf.KubeFeatureGates != "" {
		featureGates = strings.Split(conf.KubeFeatureGates, ",")
	}
	if conf.KubeRuntimeConfig != "" {
		runtimeConfig = strings.Split(conf.KubeRuntimeConfig, ",")
	}

	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	err = c.MkdirAll(pkiPath)
	if err != nil {
		return err
	}

	manifestsPath := c.GetWorkdirPath(runtime.ManifestsName)
	err = c.MkdirAll(manifestsPath)
	if err != nil {
		return err
	}

	if conf.KubeAuditPolicy != "" {
		err = c.MkdirAll(c.GetWorkdirPath("logs"))
		if err != nil {
			return err
		}

		err = c.CreateFile(env.auditLogPath)
		if err != nil {
			return err
		}

		err = c.CopyFile(conf.KubeAuditPolicy, env.auditPolicyPath)
		if err != nil {
			return err
		}
	}

	schedulerConfigPath := ""
	if conf.KubeSchedulerConfig != "" && slices.Contains(env.components, consts.ComponentKubeScheduler) {
		schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
		err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, env.schedulerConfigPath)
		if err != nil {
			return err
		}
	}

	kubeApiserverTracingConfigPath := ""
	if conf.JaegerPort != 0 {
		kubeApiserverTracingConfigData, err := components.BuildKubeApiserverTracing(components.BuildKubeApiserverTracingConfig{
			Endpoint: conf.BindAddress + ":4317",
		})
		if err != nil {
			return fmt.Errorf("failed to generate kubeApiserverTracingConfig yaml: %w", err)
		}
		kubeApiserverTracingConfigPath = c.GetWorkdirPath(runtime.ApiserverTracingConfig)

		err = c.WriteFile(kubeApiserverTracingConfigPath, []byte(kubeApiserverTracingConfigData))
		if err != nil {
			return fmt.Errorf("failed to write kubeApiserverTracingConfig yaml: %w", err)
		}
	}

	var prometheusPatches internalversion.ComponentPatches
	if conf.PrometheusPort != 0 {
		prometheusPatches = runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentPrometheus)
		prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)

		prometheusPatches.ExtraVolumes = append(prometheusPatches.ExtraVolumes, internalversion.Volume{
			Name:      "prometheus-config",
			HostPath:  prometheusConfigPath,
			MountPath: env.prometheusConfigPath,
		})
	}

	kubeVersion, err := version.ParseVersion(conf.KubeVersion)
	if err != nil {
		return err
	}

	etcdComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentEtcd)

	kubeApiserverComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentKubeApiserver)
	kubeSchedulerComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentKubeScheduler)
	kubeControllerManagerComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentKubeControllerManager)
	kwokControllerComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentKwokController)
	for _, patch := range env.kwokctlConfig.ComponentsPatches {
		switch patch.Name {
		case consts.ComponentEtcd:
			args := filterDuplicatedExtraArgs(ctx, etcdComponentPatches.ExtraArgs, patch.ExtraArgs)
			etcdComponentPatches.ExtraArgs = args
		case consts.ComponentKubeApiserver:
			args := filterDuplicatedExtraArgs(ctx, kubeApiserverComponentPatches.ExtraArgs, patch.ExtraArgs)
			kubeApiserverComponentPatches.ExtraArgs = args
		case consts.ComponentKubeScheduler:
			args := filterDuplicatedExtraArgs(ctx, kubeSchedulerComponentPatches.ExtraArgs, patch.ExtraArgs)
			kubeSchedulerComponentPatches.ExtraArgs = args
		case consts.ComponentKubeControllerManager:
			args := filterDuplicatedExtraArgs(ctx, kubeControllerManagerComponentPatches.ExtraArgs, patch.ExtraArgs)
			kubeControllerManagerComponentPatches.ExtraArgs = args
		case consts.ComponentKwokController:
			args := filterDuplicatedExtraArgs(ctx, kwokControllerComponentPatches.ExtraArgs, patch.ExtraArgs)
			kwokControllerComponentPatches.ExtraArgs = args
		}
	}
	extraLogVolumes := runtime.GetLogVolumes(ctx)
	kwokControllerExtraVolumes := kwokControllerComponentPatches.ExtraVolumes
	kwokControllerExtraVolumes = append(kwokControllerExtraVolumes, extraLogVolumes...)
	if len(etcdComponentPatches.ExtraEnvs) > 0 ||
		len(kubeApiserverComponentPatches.ExtraEnvs) > 0 ||
		len(kubeSchedulerComponentPatches.ExtraEnvs) > 0 ||
		len(kubeControllerManagerComponentPatches.ExtraEnvs) > 0 {
		logger.Warn("extraEnvs config in etcd, kube-apiserver, kube-scheduler or kube-controller-manager is not supported in kind")
	}
	kindYaml, err := BuildKind(BuildKindConfig{
		BindAddress:                   conf.BindAddress,
		KubeApiserverPort:             conf.KubeApiserverPort,
		KubeApiserverInsecurePort:     conf.KubeApiserverInsecurePort,
		EtcdPort:                      conf.EtcdPort,
		JaegerPort:                    conf.JaegerPort,
		PrometheusPort:                conf.PrometheusPort,
		KwokControllerPort:            conf.KwokControllerPort,
		FeatureGates:                  featureGates,
		RuntimeConfig:                 runtimeConfig,
		AuditPolicy:                   env.auditPolicyPath,
		AuditLog:                      env.auditLogPath,
		SchedulerConfig:               schedulerConfigPath,
		TracingConfigPath:             kubeApiserverTracingConfigPath,
		Workdir:                       c.Workdir(),
		Verbosity:                     env.verbosity,
		EtcdExtraArgs:                 etcdComponentPatches.ExtraArgs,
		EtcdExtraVolumes:              etcdComponentPatches.ExtraVolumes,
		ApiserverExtraArgs:            kubeApiserverComponentPatches.ExtraArgs,
		ApiserverExtraVolumes:         kubeApiserverComponentPatches.ExtraVolumes,
		SchedulerExtraArgs:            kubeSchedulerComponentPatches.ExtraArgs,
		SchedulerExtraVolumes:         kubeSchedulerComponentPatches.ExtraVolumes,
		ControllerManagerExtraArgs:    kubeControllerManagerComponentPatches.ExtraArgs,
		ControllerManagerExtraVolumes: kubeControllerManagerComponentPatches.ExtraVolumes,
		KwokControllerExtraVolumes:    kwokControllerExtraVolumes,
		PrometheusExtraVolumes:        prometheusPatches.ExtraVolumes,
		DisableQPSLimits:              conf.DisableQPSLimits,
		KubeVersion:                   kubeVersion,
		EtcdQuotaBackendSize:          conf.EtcdQuotaBackendSize,
	})
	if err != nil {
		return err
	}
	err = c.WriteFile(c.GetWorkdirPath(runtime.KindName), []byte(kindYaml))
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KindName, err)
	}

	return nil
}

func filterDuplicatedExtraArgs(ctx context.Context, extraArgs, passedExtraArgs []internalversion.ExtraArgs) []internalversion.ExtraArgs {
	logger := log.FromContext(ctx)
	extraArgsMap := make(map[string]internalversion.ExtraArgs)
	for _, args := range extraArgs {
		extraArgsMap[args.Key] = args
	}
	for _, args := range passedExtraArgs {
		if _, ok := extraArgsMap[args.Key]; ok {
			logger.Warn("duplicated extraArgs and will be overwritten",
				"key", args.Key,
				"value", args.Value,
			)
		}
		extraArgsMap[args.Key] = args
	}
	result := make([]internalversion.ExtraArgs, 0, len(extraArgsMap))
	for _, args := range extraArgsMap {
		result = append(result, args)
	}
	return result
}

func (c *Cluster) buildKindNodeImage(ctx context.Context, image, kubeVersion string) error {
	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	return c.Exec(utilsexec.WithAllWriteToErrOut(c.withProviderEnv(ctx)), kindPath,
		"build", "node-image",
		"--image", image,
		kubeVersion,
	)
}

// https://github.com/kubernetes-sigs/kind/blob/7b017b2ce14a7fdea9d3ed2fa259c38c927e2dd1/pkg/internal/runtime/runtime.go
func (c *Cluster) withProviderEnv(ctx context.Context) context.Context {
	provider := c.runtime
	ctx = utilsexec.WithEnv(ctx, []string{
		"KIND_EXPERIMENTAL_PROVIDER=" + provider,
	})
	return ctx
}

// preDownloadKind pre-download and cache kind
func (c *Cluster) preDownloadKind(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options

	_, err = exec.LookPath("kind")
	if err != nil {
		// kind does not exist, try to download it
		kindPath, err := c.EnsureBinary(ctx, "kind", conf.KindBinary)
		if err != nil {
			return "", err
		}
		return kindPath, nil
	}

	return "kind", nil
}
