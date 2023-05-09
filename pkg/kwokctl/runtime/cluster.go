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
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nxadm/tail"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwok/controllers"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/stages"
)

// The following functions are used to get the path of the cluster
var (
	ConfigName              = consts.ConfigName
	InHostKubeconfigName    = "kubeconfig.yaml"
	InClusterKubeconfigName = "kubeconfig"
	EtcdDataDirName         = "etcd"
	PkiName                 = "pki"
	ComposeName             = "docker-compose.yaml"
	Prometheus              = "prometheus.yaml"
	KindName                = "kind.yaml"
	KwokPod                 = "kwok-controller-pod.yaml"
	PrometheusDeploy        = "prometheus-deployment.yaml"
	AuditPolicyName         = "audit.yaml"
	AuditLogName            = "audit.log"
	SchedulerConfigName     = "scheduler.yaml"
)

// Cluster is the cluster
type Cluster struct {
	workdir string
	name    string
	conf    *internalversion.KwokctlConfiguration
}

// NewCluster creates a new cluster
func NewCluster(name, workdir string) *Cluster {
	return &Cluster{
		name:    name,
		workdir: workdir,
	}
}

// Config returns the cluster config
func (c *Cluster) Config(ctx context.Context) (*internalversion.KwokctlConfiguration, error) {
	if c.conf != nil {
		return c.conf, nil
	}
	conf, err := c.Load(ctx)
	if err != nil {
		return conf, err
	}
	c.conf = conf
	return conf, nil
}

// Name returns the cluster name
func (c *Cluster) Name() string {
	return c.name
}

// Workdir returns the cluster workdir
func (c *Cluster) Workdir() string {
	return c.workdir
}

// Load loads the cluster config
func (c *Cluster) Load(ctx context.Context) (*internalversion.KwokctlConfiguration, error) {
	objs, err := config.Load(ctx, c.GetWorkdirPath(ConfigName))
	if err != nil {
		return nil, err
	}

	configs := config.FilterWithType[*internalversion.KwokctlConfiguration](objs)
	if len(configs) == 0 {
		return nil, fmt.Errorf("failed to load config")
	}
	return configs[0], nil
}

// SetConfig sets the cluster config
func (c *Cluster) SetConfig(ctx context.Context, conf *internalversion.KwokctlConfiguration) error {
	c.conf = conf
	return nil
}

// Save saves the cluster config
func (c *Cluster) Save(ctx context.Context) error {
	if c.conf == nil {
		return nil
	}

	objs := []config.InternalObject{
		c.conf,
	}

	others := config.FilterWithoutTypeFromContext[*internalversion.KwokctlConfiguration](ctx)
	objs = append(objs, others...)

	if updateFrequency := c.conf.Options.NodeStatusUpdateFrequencyMilliseconds; updateFrequency > 0 &&
		c.conf.Options.Runtime != consts.RuntimeTypeKind &&
		c.conf.Options.Runtime != consts.RuntimeTypeKindPodman &&
		len(config.FilterWithTypeFromContext[*internalversion.Stage](ctx)) == 0 {
		nodeStages, err := controllers.NewStagesFromYaml([]byte(stages.DefaultNodeStages))
		if err != nil {
			return err
		}
		hasUpdate := false
		for _, stage := range nodeStages {
			if stage.Name == "node-heartbeat" {
				stage.Spec.Delay.DurationMilliseconds = format.Ptr(updateFrequency)
				stage.Spec.Delay.JitterDurationMilliseconds = format.Ptr(updateFrequency + updateFrequency/10)
				hasUpdate = true
			}
			objs = append(objs, stage)
		}
		if !hasUpdate {
			return fmt.Errorf("failed to update node heartbeat stage")
		}
	}

	return config.Save(ctx, c.GetWorkdirPath(ConfigName), objs)
}

func (c *Cluster) kubectlPath(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options

	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		kubectlPath = c.GetBinPath("kubectl" + conf.BinSuffix)
		err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KubectlBinary, kubectlPath, 0750, conf.QuietPull)
		if err != nil {
			return "", err
		}
	}
	return kubectlPath, nil
}

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
	_, err := c.kubectlPath(ctx)
	return err
}

// Uninstall uninstalls the cluster.
func (c *Cluster) Uninstall(ctx context.Context) error {
	// cleanup workdir
	return os.RemoveAll(c.Workdir())
}

// Ready returns true if the cluster is ready
func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	out := bytes.NewBuffer(nil)
	err := c.KubectlInCluster(exec.WithAllWriteTo(ctx, out), "get", "--raw", "/healthz")
	if err != nil {
		return false, err
	}
	if !bytes.Equal(out.Bytes(), []byte("ok")) {
		logger := log.FromContext(ctx)
		logger.Debug("Check Ready",
			"method", "get /healthz",
			"response", out,
		)
		return false, nil
	}
	return true, nil
}

// WaitReady waits for the cluster to be ready.
func (c *Cluster) WaitReady(ctx context.Context, timeout time.Duration) error {
	var (
		err     error
		waitErr error
		ready   bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		ready, err = c.Ready(ctx)
		if err != nil {
			logger.Debug("Cluster is not ready",
				"err", err,
			)
		}
		return ready, nil
	}, wait.WithTimeout(timeout), wait.WithImmediate())
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// GetComponent returns the component by name
func (c *Cluster) GetComponent(ctx context.Context, name string) (internalversion.Component, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return internalversion.Component{}, err
	}
	component, ok := slices.Find(config.Components, func(component internalversion.Component) bool {
		return component.Name == name
	})
	if !ok {
		return internalversion.Component{}, fmt.Errorf("%w: %s", ErrComponentNotFound, name)
	}

	return component, nil
}

// Kubectl runs kubectl.
func (c *Cluster) Kubectl(ctx context.Context, args ...string) error {
	kubectlPath, err := c.kubectlPath(ctx)
	if err != nil {
		return err
	}

	return exec.Exec(ctx, kubectlPath, args...)
}

// KubectlInCluster runs kubectl in the cluster.
func (c *Cluster) KubectlInCluster(ctx context.Context, args ...string) error {
	kubectlPath, err := c.kubectlPath(ctx)
	if err != nil {
		return err
	}

	return exec.Exec(ctx, kubectlPath, append([]string{"--kubeconfig", c.GetWorkdirPath(InHostKubeconfigName)}, args...)...)
}

// AuditLogs returns the audit logs of the cluster.
func (c *Cluster) AuditLogs(ctx context.Context, out io.Writer) error {
	logs := c.GetLogPath(AuditLogName)

	f, err := os.OpenFile(logs, os.O_RDONLY, 0640)
	if err != nil {
		return err
	}
	logger := log.FromContext(ctx)
	defer func() {
		err = f.Close()
		if err != nil {
			logger.Error("Failed to close file", err)
		}
	}()

	_, err = io.Copy(out, f)
	return err
}

// AuditLogsFollow follows the audit logs of the cluster.
func (c *Cluster) AuditLogsFollow(ctx context.Context, out io.Writer) error {
	logs := c.GetLogPath(AuditLogName)

	t, err := tail.TailFile(logs, tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return err
	}
	logger := log.FromContext(ctx)
	defer func() {
		err = t.Stop()
		if err != nil {
			logger.Error("Failed to stop tail file", err)
		}
	}()

	go func() {
		for line := range t.Lines {
			_, err = out.Write([]byte(line.Text + "\n"))
			if err != nil {
				logger.Error("Failed to write line text", err)
			}
		}
	}()
	<-ctx.Done()
	return nil
}

// GetWorkdirPath returns the path to the file in the workdir.
func (c *Cluster) GetWorkdirPath(name string) string {
	return path.Join(c.workdir, name)
}

// GetBinPath returns the path to the given binary name.
func (c *Cluster) GetBinPath(name string) string {
	return path.Join(c.workdir, "bin", name)
}

// GetLogPath returns the path of the given log name.
func (c *Cluster) GetLogPath(name string) string {
	return path.Join(c.workdir, "logs", name)
}
