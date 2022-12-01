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

package runtime

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/nxadm/tail"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
)

var (
	RawClusterConfigName    = "kwok.yaml"
	InHostKubeconfigName    = "kubeconfig.yaml"
	InClusterKubeconfigName = "kubeconfig"
	EtcdDataDirName         = "etcd"
	PkiName                 = "pki"
	ComposeName             = "docker-compose.yaml"
	Prometheus              = "prometheus.yaml"
	KindName                = "kind.yaml"
	KwokDeploy              = "kwok-controller-deployment.yaml"
	PrometheusDeploy        = "prometheus-deployment.yaml"
	AuditPolicyName         = "audit.yaml"
	AuditLogName            = "audit.log"
)

type Cluster struct {
	workdir string
	name    string
	conf    *Config
}

func NewCluster(name, workdir string) *Cluster {
	return &Cluster{
		name:    name,
		workdir: workdir,
	}
}

func (c *Cluster) Config() (Config, error) {
	if c.conf != nil {
		return *c.conf, nil
	}
	conf, err := c.Load()
	if err != nil {
		return conf, err
	}
	c.conf = &conf
	return conf, nil
}

func (c *Cluster) Load() (conf Config, err error) {
	file, err := os.ReadFile(utils.PathJoin(c.workdir, RawClusterConfigName))
	if err != nil {
		return conf, err
	}
	err = yaml.Unmarshal(file, &conf)
	if err != nil {
		return conf, err
	}
	return conf, nil
}

func (c *Cluster) InHostKubeconfig() (string, error) {
	conf, err := c.Config()
	if err != nil {
		return "", err
	}

	return utils.PathJoin(conf.Workdir, InHostKubeconfigName), nil
}

func (c *Cluster) Init(ctx context.Context, conf Config) error {
	return c.Update(ctx, conf)
}

func (c *Cluster) Update(ctx context.Context, conf Config) error {
	config, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	err = os.MkdirAll(c.workdir, 0755)
	if err != nil {
		return err
	}
	c.conf = &conf

	err = os.WriteFile(utils.PathJoin(c.workdir, RawClusterConfigName), config, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) kubectlPath(ctx context.Context) (string, error) {
	conf, err := c.Config()
	if err != nil {
		return "", err
	}

	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		bin := utils.PathJoin(conf.Workdir, "bin")
		kubectlPath = utils.PathJoin(bin, "kubectl"+vars.BinSuffix)
		err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubectlBinary, kubectlPath, 0755, conf.QuietPull)
		if err != nil {
			return "", err
		}
	}
	return kubectlPath, nil
}

func (c *Cluster) Install(ctx context.Context) error {
	_, err := c.kubectlPath(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Uninstall(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	// cleanup workdir
	os.RemoveAll(conf.Workdir)
	return nil
}

func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	out := bytes.NewBuffer(nil)
	err := c.KubectlInCluster(ctx, utils.IOStreams{
		Out:    out,
		ErrOut: out,
	}, "get", "--raw", "/healthz")
	if err != nil {
		return false, err
	}
	if !bytes.Equal(out.Bytes(), []byte("ok")) {
		return false, nil
	}
	return true, nil
}

func (c *Cluster) WaitReady(ctx context.Context, timeout time.Duration) error {
	var err error
	var ready bool
	for i := 0; i < int(timeout/time.Second); i++ {
		ready, err = c.Ready(ctx)
		if ready {
			return nil
		}
		time.Sleep(time.Second)
	}
	return err
}

func (c *Cluster) Kubectl(ctx context.Context, stm utils.IOStreams, args ...string) error {
	kubectlPath, err := c.kubectlPath(ctx)
	if err != nil {
		return err
	}

	return utils.Exec(ctx, "", stm, kubectlPath, args...)
}

func (c *Cluster) KubectlInCluster(ctx context.Context, stm utils.IOStreams, args ...string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	kubectlPath, err := c.kubectlPath(ctx)
	if err != nil {
		return err
	}

	return utils.Exec(ctx, "", stm, kubectlPath,
		append([]string{"--kubeconfig", utils.PathJoin(conf.Workdir, InHostKubeconfigName)}, args...)...)
}

func (c *Cluster) AuditLogs(ctx context.Context, out io.Writer) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	logs := utils.PathJoin(conf.Workdir, "logs", AuditLogName)

	f, err := os.OpenFile(logs, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	io.Copy(out, f)
	return nil
}

func (c *Cluster) AuditLogsFollow(ctx context.Context, out io.Writer) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	logs := utils.PathJoin(conf.Workdir, "logs", AuditLogName)

	t, err := tail.TailFile(logs, tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return err
	}
	defer t.Stop()

	go func() {
		for line := range t.Lines {
			out.Write([]byte(line.Text + "\n"))
		}
	}()
	<-ctx.Done()
	return nil
}
