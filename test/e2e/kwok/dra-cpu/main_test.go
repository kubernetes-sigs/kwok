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

// Package dracpu_test is a test environment comparing the kwok DRA CPU
// simulation with the real dra-driver-cpu running in the same kind cluster.
// It requires the helm binary to install the real driver.
package dracpu_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

var (
	testEnv     env.Environment
	pwd         = os.Getenv("PWD")
	rootDir     = path.Join(pwd, "../../../..")
	clusterName = envconf.RandomName("kwok-e2e-dra", 20)
	namespace   = envconf.RandomName("ns", 16)
	testImage   = "localhost/kwok:test"
	kindConfig  = path.Join(pwd, "kind-config.yaml")
)

const (
	draDriverCPUChart   = "oci://registry.k8s.io/dra-driver-cpu/charts/dra-driver-cpu"
	draDriverCPUVersion = "0.2.0"
	draDriverCPURelease = "dra-driver-cpu"
)

// installDRADriverCPU installs the real dra-driver-cpu via its helm chart so
// the suite can compare the kwok simulation against it. The chart also
// installs the dra.cpu DeviceClass used by both sides. The image is pinned to
// the release registry because the chart defaults reference the staging
// registry, whose images are garbage collected. The node affinity keeps the
// DaemonSet off kwok fake nodes.
func installDRADriverCPU() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		affinity := `affinity={"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"type","operator":"NotIn","values":["kwok"]}]}]}}}`
		cmd := exec.CommandContext(ctx, "helm", "install", draDriverCPURelease, draDriverCPUChart,
			"--version", draDriverCPUVersion,
			"--namespace", "kube-system",
			"--kubeconfig", cfg.KubeconfigFile(),
			"--set", "image.repository=registry.k8s.io/dra-driver-cpu/dra-driver-cpu",
			"--set", "image.tag=v"+draDriverCPUVersion,
			"--set-json", affinity,
			"--wait",
			"--timeout", "5m",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return ctx, fmt.Errorf("helm install dra-driver-cpu: %w\n%s", err, out)
		}
		return ctx, nil
	}
}

// createDRACPUStages renders the shipped kustomize/stage/dra/cpu
// kustomization (stages + RBAC) and creates everything except the
// DeviceClass, which the dra-driver-cpu helm chart installs and owns.
func createDRACPUStages(dir string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}
		resMap, err := krusty.MakeKustomizer(krusty.MakeDefaultOptions()).Run(filesys.MakeFsOnDisk(), dir)
		if err != nil {
			return ctx, fmt.Errorf("render kustomization %s: %w", dir, err)
		}
		for _, res := range resMap.Resources() {
			if res.GetKind() != "DeviceClass" {
				continue
			}
			if err := resMap.Remove(res.CurId()); err != nil {
				return ctx, err
			}
		}
		yml, err := resMap.AsYaml()
		if err != nil {
			return ctx, err
		}
		if err := decoder.DecodeEach(ctx, bytes.NewReader(yml), decoder.CreateHandler(r)); err != nil {
			return ctx, fmt.Errorf("apply kustomization %s: %w", dir, err)
		}
		return ctx, nil
	}
}

func TestMain(m *testing.M) {
	testEnv = helper.Environment()

	deploy := pwd
	crs := path.Join(rootDir, "kustomize/stage/fast")
	draCPUDir := path.Join(rootDir, "kustomize/stage/dra/cpu")
	testEnv.Setup(
		helper.BuildKwokImage(rootDir, testImage, consts.RuntimeTypeDocker),
		envfuncs.CreateClusterWithConfig(kind.NewProvider(), clusterName, kindConfig),
		helper.WaitForAllNodesReady(),
		envfuncs.LoadImageToCluster(clusterName, testImage),
		helper.CreateByKustomize(deploy),
		helper.WaitForAllPodsReady(),
		helper.CreateByKustomize(crs),
		// The CPU stage kustomization is rendered as shipped, with the
		// chart-owned DeviceClass filtered out, and applied before the
		// driver installation so the kwok controller has long settled on
		// the stages by the time the test creates the fake node.
		createDRACPUStages(draCPUDir),
		installDRADriverCPU(),
		helper.CreateNamespace(namespace),
	)
	testEnv.Finish(
		envfuncs.DestroyCluster(clusterName),
	)
	os.Exit(testEnv.Run(m))
}
