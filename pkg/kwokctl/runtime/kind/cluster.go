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

package kind

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/image"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

type Cluster struct {
	*runtime.Cluster
}

func NewCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
	}, nil
}

func (c *Cluster) Install(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	inClusterKubeconfig := "/etc/kubernetes/scheduler.conf"

	var featureGates []string
	var runtimeConfig []string
	if conf.KubeFeatureGates != "" {
		featureGates = strings.Split(strings.ReplaceAll(conf.KubeFeatureGates, "=", ": "), ",")
	}
	if conf.KubeRuntimeConfig != "" {
		runtimeConfig = strings.Split(strings.ReplaceAll(conf.KubeRuntimeConfig, "=", ": "), ",")
	}

	auditLogPath := ""
	auditPolicyPath := ""
	if conf.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		err = file.Create(auditLogPath, 0640)
		if err != nil {
			return err
		}

		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
		err = file.Copy(conf.KubeAuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	schedulerConfigPath := ""
	if !conf.DisableKubeScheduler && conf.KubeSchedulerConfig != "" {
		schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
		err = k8s.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, inClusterKubeconfig)
		if err != nil {
			return err
		}
	}

	configPath := c.GetWorkdirPath(runtime.ConfigName)
	kindYaml, err := BuildKind(BuildKindConfig{
		KubeApiserverPort: conf.KubeApiserverPort,
		PrometheusPort:    conf.PrometheusPort,
		FeatureGates:      featureGates,
		RuntimeConfig:     runtimeConfig,
		AuditPolicy:       auditPolicyPath,
		AuditLog:          auditLogPath,
		SchedulerConfig:   schedulerConfigPath,
		ConfigPath:        configPath,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(c.GetWorkdirPath(runtime.KindName), []byte(kindYaml), 0640)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KindName, err)
	}

	kwokControllerPod, err := BuildKwokControllerPod(BuildKwokControllerPodConfig{
		KwokControllerImage: conf.KwokControllerImage,
		Name:                c.Name(),
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(c.GetWorkdirPath(runtime.KwokPod), []byte(kwokControllerPod), 0640)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KwokPod, err)
	}

	if conf.PrometheusPort != 0 {
		prometheusDeploy, err := BuildPrometheusDeployment(BuildPrometheusDeploymentConfig{
			PrometheusImage: conf.PrometheusImage,
			Name:            c.Name(),
		})
		if err != nil {
			return err
		}
		err = os.WriteFile(c.GetWorkdirPath(runtime.PrometheusDeploy), []byte(prometheusDeploy), 0640)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", runtime.PrometheusDeploy, err)
		}
	}

	images := []string{
		conf.KindNodeImage,
		conf.KwokControllerImage,
	}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}
	err = image.PullImages(ctx, "docker", images, conf.QuietPull)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) Up(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	logger := log.FromContext(ctx)

	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	args := []string{
		"create", "cluster",
		"--config", c.GetWorkdirPath(runtime.KindName),
		"--name", c.Name(),
		"--image", conf.KindNodeImage,
	}

	deadline, ok := ctx.Deadline()
	if ok {
		wait := time.Until(deadline)
		if wait < 0 {
			wait = time.Minute
		}
		args = append(args, "--wait", format.HumanDuration(wait))
	} else {
		args = append(args, "--wait", "1m")
	}

	err = exec.Exec(ctx, "", exec.IOStreams{
		ErrOut: os.Stderr,
		Out:    os.Stderr,
	}, kindPath, args...)
	if err != nil {
		return err
	}

	images := []string{conf.KwokControllerImage}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}
	err = loadImages(ctx, kindPath, c.Name(), images)
	if err != nil {
		return err
	}

	kubeconfig, err := c.InHostKubeconfig()
	if err != nil {
		return err
	}

	kubeconfigBuf := bytes.NewBuffer(nil)
	err = c.Kubectl(ctx, exec.IOStreams{
		Out: kubeconfigBuf,
	}, "config", "view", "--minify=true", "--raw=true")
	if err != nil {
		return err
	}

	err = os.WriteFile(kubeconfig, kubeconfigBuf.Bytes(), 0640)
	if err != nil {
		return err
	}

	err = exec.Exec(ctx, "", exec.IOStreams{}, "docker", "cp", c.GetWorkdirPath(runtime.KwokPod), c.Name()+"-control-plane:/etc/kubernetes/manifests/kwok-controller.yaml")
	if err != nil {
		return err
	}

	config.Components = append(config.Components,
		internalversion.Component{
			Name: "etcd",
		},
		internalversion.Component{
			Name: "kube-apiserver",
		},
		internalversion.Component{
			Name: "kwok-controller",
		},
	)

	if conf.PrometheusPort != 0 {
		err = c.Kubectl(ctx, exec.IOStreams{
			ErrOut: os.Stderr,
		}, "apply", "-f", c.GetWorkdirPath(runtime.PrometheusDeploy))
		if err != nil {
			return err
		}

		config.Components = append(config.Components,
			internalversion.Component{
				Name: "prometheus",
			},
		)
	}

	err = c.Kubectl(ctx, exec.IOStreams{}, "cordon", c.Name()+"-control-plane")
	if err != nil {
		logger.Error("Failed cordon node", err)
	}

	if conf.DisableKubeScheduler {
		err := c.StopComponent(ctx, "kube-scheduler")
		if err != nil {
			logger.Error("Failed to disable kube-scheduler", err)
		}
	} else {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: "kube-scheduler",
			},
		)
	}

	if conf.DisableKubeControllerManager {
		err := c.StopComponent(ctx, "kube-controller-manager")
		if err != nil {
			logger.Error("Failed to disable kube-controller-manager", err)
		}
	} else {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: "kube-controller-manager",
			},
		)
	}

	err = c.SetConfig(ctx, config)
	if err != nil {
		logger.Error("Failed to set config", err)
	}
	err = c.Save(ctx)
	if err != nil {
		logger.Error("Failed to update cluster", err)
	}

	// set the context in default kubeconfig
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "contexts."+c.Name()+".cluster", "kind-"+c.Name())
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "contexts."+c.Name()+".user", "kind-"+c.Name())

	return nil
}

func loadImages(ctx context.Context, command, kindCluster string, images []string) error {
	logger := log.FromContext(ctx)
	for _, image := range images {
		err := exec.Exec(ctx, "", exec.IOStreams{},
			command, "load", "docker-image",
			image,
			"--name", kindCluster,
		)
		if err != nil {
			return err
		}
		logger.Info("Loaded image", "image", image)
	}
	return nil
}

func (c *Cluster) WaitReady(ctx context.Context, timeout time.Duration) error {
	var (
		err     error
		waitErr error
		ready   bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.PollImmediateWithContext(ctx, time.Second, timeout, func(ctx context.Context) (bool, error) {
		ready, err = c.Ready(ctx)
		if err != nil {
			logger.Debug("Cluster is not ready",
				"err", err,
			)
		}
		return ready, nil
	})
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	ok, err := c.Cluster.Ready(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	out := bytes.NewBuffer(nil)
	err = c.KubectlInCluster(ctx, exec.IOStreams{
		Out: out,
	}, "get", "pod", "--namespace=kube-system", "--field-selector=status.phase!=Running", "--output=json")
	if err != nil {
		return false, err
	}

	var data corev1.PodList
	err = json.Unmarshal(out.Bytes(), &data)
	if err != nil {
		return false, err
	}

	if len(data.Items) != 0 {
		logger := log.FromContext(ctx)
		logger.Debug("Components not all running",
			"components", slices.Map(data.Items, func(item corev1.Pod) interface{} {
				return struct {
					Pod   string
					Phase string
				}{
					Pod:   log.KObj(&item).String(),
					Phase: string(item.Status.Phase),
				}
			}),
		)
		return false, nil
	}
	return true, nil
}

func (c *Cluster) Down(ctx context.Context) error {
	// unset the context in default kubeconfig
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "unset", "contexts."+c.Name()+".cluster")
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "unset", "contexts."+c.Name()+".user")

	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	err = exec.Exec(ctx, "", exec.IOStreams{
		ErrOut: os.Stderr,
		Out:    os.Stderr,
	}, kindPath, "delete", "cluster", "--name", c.Name())
	if err != nil {
		logger.Error("Failed to delete cluster", err)
	}

	return nil
}

func (c *Cluster) Start(ctx context.Context) error {
	err := exec.Exec(ctx, "", exec.IOStreams{}, "docker", "start", c.getClusterName())
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Stop(ctx context.Context) error {
	err := exec.Exec(ctx, "", exec.IOStreams{}, "docker", "stop", c.getClusterName())
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) StartComponent(ctx context.Context, name string) error {
	err := exec.Exec(ctx, "", exec.IOStreams{}, "docker", "exec", c.Name()+"-control-plane", "mv", "/etc/kubernetes/"+name+".yaml.bak", "/etc/kubernetes/manifests/"+name+".yaml")
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) StopComponent(ctx context.Context, name string) error {
	err := exec.Exec(ctx, "", exec.IOStreams{}, "docker", "exec", c.Name()+"-control-plane", "mv", "/etc/kubernetes/manifests/"+name+".yaml", "/etc/kubernetes/"+name+".yaml.bak")
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) getClusterName() string {
	return c.Name() + "-control-plane"
}

func (c *Cluster) getComponentName(name string) string {
	clusterName := c.getClusterName()
	switch name {
	case "prometheus":
	default:
		name = name + "-" + clusterName
	}
	return name
}

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	componentName := c.getComponentName(name)

	args := []string{"logs", "-n", "kube-system"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, componentName)

	err := c.Kubectl(ctx, exec.IOStreams{
		ErrOut: out,
		Out:    out,
	}, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, false)
}

func (c *Cluster) LogsFollow(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, true)
}

// ListBinaries list binaries in the cluster
func (c *Cluster) ListBinaries(ctx context.Context) ([]string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	conf := &config.Options

	return []string{
		conf.KubectlBinary,
	}, nil
}

// ListImages list images in the cluster
func (c *Cluster) ListImages(ctx context.Context) ([]string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	conf := &config.Options

	return []string{
		conf.KindNodeImage,
		conf.KwokControllerImage,
		conf.PrometheusImage,
	}, nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, stm exec.IOStreams, args ...string) error {
	etcdContainerName := c.getComponentName("etcd")

	return c.KubectlInCluster(ctx, stm,
		append([]string{"exec", "-i", "-n", "kube-system", etcdContainerName, "--", "etcdctl", "--endpoints=127.0.0.1:2379", "--cert=/etc/kubernetes/pki/etcd/server.crt", "--key=/etc/kubernetes/pki/etcd/server.key", "--cacert=/etc/kubernetes/pki/etcd/ca.crt"}, args...)...)
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
		kindPath := c.GetBinPath("kind" + conf.BinSuffix)
		err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KindBinary, kindPath, 0755, conf.QuietPull)
		if err != nil {
			return "", err
		}
		return kindPath, nil
	}

	return "kind", nil
}
