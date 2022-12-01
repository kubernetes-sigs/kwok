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
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/log"
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
	conf, err := c.Config()
	if err != nil {
		return err
	}

	var featureGates []string
	var runtimeConfig []string
	if conf.FeatureGates != "" {
		featureGates = strings.Split(strings.ReplaceAll(conf.FeatureGates, "=", ": "), ",")
	}
	if conf.RuntimeConfig != "" {
		runtimeConfig = strings.Split(strings.ReplaceAll(conf.RuntimeConfig, "=", ": "), ",")
	}

	auditLogPath := ""
	auditPolicyPath := ""
	if conf.AuditPolicy != "" {
		auditLogPath = utils.PathJoin(conf.Workdir, "logs", runtime.AuditLogName)
		err = utils.CreateFile(auditLogPath, 0644)
		if err != nil {
			return err
		}

		auditPolicyPath = utils.PathJoin(conf.Workdir, runtime.AuditPolicyName)
		err = utils.CopyFile(conf.AuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	kindYaml, err := BuildKind(BuildKindConfig{
		KubeApiserverPort: conf.KubeApiserverPort,
		PrometheusPort:    conf.PrometheusPort,
		FeatureGates:      featureGates,
		RuntimeConfig:     runtimeConfig,
		AuditPolicy:       auditPolicyPath,
		AuditLog:          auditLogPath,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(utils.PathJoin(conf.Workdir, runtime.KindName), []byte(kindYaml), 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KindName, err)
	}

	kwokControllerDeploy, err := BuildKwokControllerDeployment(BuildKwokControllerDeploymentConfig{
		KwokControllerImage: conf.KwokControllerImage,
		Name:                conf.Name,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(utils.PathJoin(conf.Workdir, runtime.KwokDeploy), []byte(kwokControllerDeploy), 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KwokDeploy, err)
	}

	if conf.PrometheusPort != 0 {
		prometheusDeploy, err := BuildPrometheusDeployment(BuildPrometheusDeploymentConfig{
			PrometheusImage: conf.PrometheusImage,
			Name:            conf.Name,
		})
		if err != nil {
			return err
		}
		err = os.WriteFile(utils.PathJoin(conf.Workdir, runtime.PrometheusDeploy), []byte(prometheusDeploy), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", runtime.PrometheusDeploy, err)
		}
	}

	var out io.Writer = os.Stderr
	if conf.QuietPull {
		out = nil
	}
	images := []string{
		conf.KindNodeImage,
		conf.KwokControllerImage,
	}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}
	logger := log.FromContext(ctx)
	for _, image := range images {
		err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "inspect",
			image,
		)
		if err != nil {
			logger.Info("Pull image", "image", image)
			err = utils.Exec(ctx, "", utils.IOStreams{
				Out:    out,
				ErrOut: out,
			}, "docker", "pull",
				image,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Cluster) Up(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	err = utils.Exec(ctx, "", utils.IOStreams{
		ErrOut: os.Stderr,
	}, conf.Runtime, "create", "cluster",
		"--config", utils.PathJoin(conf.Workdir, runtime.KindName),
		"--name", conf.Name,
		"--image", conf.KindNodeImage,
	)
	if err != nil {
		return err
	}

	kubeconfig, err := c.InHostKubeconfig()
	if err != nil {
		return err
	}

	kubeconfigBuf := bytes.NewBuffer(nil)
	err = c.Kubectl(ctx, utils.IOStreams{
		Out: kubeconfigBuf,
	}, "config", "view", "--minify=true", "--raw=true")
	if err != nil {
		return err
	}

	err = os.WriteFile(kubeconfig, kubeconfigBuf.Bytes(), 0644)
	if err != nil {
		return err
	}

	err = c.WaitReady(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to wait for kube-apiserver ready: %v", err)
	}

	err = c.Kubectl(ctx, utils.IOStreams{}, "cordon", conf.Name+"-control-plane")
	if err != nil {
		return err
	}

	err = utils.Exec(ctx, "", utils.IOStreams{}, "kind", "load", "docker-image",
		conf.KwokControllerImage,
		"--name", conf.Name,
	)
	if err != nil {
		return err
	}
	err = c.Kubectl(ctx, utils.IOStreams{
		ErrOut: os.Stderr,
	}, "apply", "-f", utils.PathJoin(conf.Workdir, runtime.KwokDeploy))
	if err != nil {
		return err
	}

	if conf.PrometheusPort != 0 {
		err = utils.Exec(ctx, "", utils.IOStreams{}, "kind", "load", "docker-image",
			conf.PrometheusImage,
			"--name", conf.Name,
		)
		if err != nil {
			return err
		}
		err = c.Kubectl(ctx, utils.IOStreams{
			ErrOut: os.Stderr,
		}, "apply", "-f", utils.PathJoin(conf.Workdir, runtime.PrometheusDeploy))
		if err != nil {
			return err
		}
	}

	// set the context in default kubeconfig
	c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+conf.Name+".cluster", "kind-"+conf.Name)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+conf.Name+".user", "kind-"+conf.Name)

	if conf.DisableKubeControllerManager {
		// waiting for kwok-controller and other addons to be ready
		// Fixme: use a better way to wait for pods to be ready when the kube-controller-manager is disabled
		err = c.Kubectl(ctx, utils.IOStreams{
			ErrOut: os.Stderr,
		}, "-n", "kube-system", "wait", "deployment/kwok-controller", "--for", "condition=Available=true", "--timeout=5m")
		if err != nil {
			return err
		}
	}

	if conf.DisableKubeScheduler {
		if err := c.Stop(ctx, "kube-scheduler"); err != nil {
			return err
		}
	}

	if conf.DisableKubeControllerManager {
		if err := c.Stop(ctx, "kube-controller-manager"); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cluster) Down(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	// unset the context in default kubeconfig
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "contexts."+conf.Name+".cluster")
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "contexts."+conf.Name+".user")

	logger := log.FromContext(ctx)
	err = utils.Exec(ctx, "", utils.IOStreams{
		ErrOut: os.Stderr,
	}, conf.Runtime, "delete", "cluster", "--name", conf.Name)
	if err != nil {
		logger.Error("Failed to delete cluster", err)
	}

	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "exec", conf.Name+"-control-plane", "mv", "/etc/kubernetes/"+name+".yaml.bak", "/etc/kubernetes/manifests/"+name+".yaml")
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Stop(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "exec", conf.Name+"-control-plane", "mv", "/etc/kubernetes/manifests/"+name+".yaml", "/etc/kubernetes/"+name+".yaml.bak")
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) getClusterName() (string, error) {
	conf, err := c.Config()
	if err != nil {
		return "", err
	}
	return conf.Name + "-control-plane", nil
}

func (c *Cluster) getComponentName(name string) (string, error) {
	clusterName, err := c.getClusterName()
	if err != nil {
		return "", err
	}
	switch name {
	case "kwok-controller", "prometheus":
	default:
		name = name + "-" + clusterName
	}
	return name, nil
}

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	name, err := c.getComponentName(name)
	if err != nil {
		return err
	}

	args := []string{"logs", "-n", "kube-system"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, name)

	err = c.Kubectl(ctx, utils.IOStreams{
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
func (c *Cluster) ListBinaries(ctx context.Context, actual bool) ([]string, error) {
	if !actual {
		return []string{
			vars.KubectlBinary,
		}, nil
	}
	_, err := c.Config()
	if err != nil {
		return nil, err
	}

	return []string{
		vars.KubectlBinary,
	}, nil
}

// ListImages list images in the cluster
func (c *Cluster) ListImages(ctx context.Context, actual bool) ([]string, error) {
	if !actual {
		return []string{
			vars.KindNodeImage,
			vars.KwokControllerImage,
			vars.PrometheusImage,
		}, nil
	}
	conf, err := c.Config()
	if err != nil {
		return nil, err
	}
	return []string{
		conf.KindNodeImage,
		conf.KwokControllerImage,
		conf.PrometheusImage,
	}, nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, stm utils.IOStreams, args ...string) error {
	etcdContainerName, err := c.getComponentName("etcd")
	if err != nil {
		return err
	}

	return c.KubectlInCluster(ctx, stm,
		append([]string{"exec", "-i", "-n", "kube-system", etcdContainerName, "--", "etcdctl", "--endpoints=127.0.0.1:2379", "--cert=/etc/kubernetes/pki/etcd/server.crt", "--key=/etc/kubernetes/pki/etcd/server.key", "--cacert=/etc/kubernetes/pki/etcd/ca.crt"}, args...)...)
}
