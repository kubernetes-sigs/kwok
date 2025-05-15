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
	"strings"
	"time"

	"github.com/nxadm/tail"

	"sigs.k8s.io/kwok/kustomize/crd"
	"sigs.k8s.io/kwok/kustomize/metrics/resource"
	"sigs.k8s.io/kwok/kustomize/metrics/usage"
	nodefast "sigs.k8s.io/kwok/kustomize/stage/node/fast"
	nodeheartbeat "sigs.k8s.io/kwok/kustomize/stage/node/heartbeat"
	nodeheartbeatwithlease "sigs.k8s.io/kwok/kustomize/stage/node/heartbeat-with-lease"
	podfast "sigs.k8s.io/kwok/kustomize/stage/pod/fast"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/snapshot"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// The following functions are used to get the path of the cluster
var (
	ConfigName              = consts.ConfigName
	InHostKubeconfigName    = "kubeconfig.yaml"
	InClusterKubeconfigName = "kubeconfig"
	EtcdDataDirName         = "etcd"
	PkiName                 = "pki"
	ManifestsName           = "manifests"
	Prometheus              = "prometheus.yaml"
	KindName                = "kind.yaml"
	AuditPolicyName         = "audit.yaml"
	AuditLogName            = "audit.log"
	SchedulerConfigName     = "scheduler.yaml"
	ApiserverTracingConfig  = "apiserver-tracing-config.yaml"
)

// Cluster is the cluster
type Cluster struct {
	workdir string
	name    string
	dryRun  bool
	conf    *internalversion.KwokctlConfiguration

	clientset client.Clientset
}

// NewCluster creates a new cluster
func NewCluster(name, workdir string) *Cluster {
	return &Cluster{
		name:    name,
		workdir: workdir,
		dryRun:  dryrun.DryRun,
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
	logger := log.FromContext(ctx)
	logger = logger.With("kwokctlVersion", consts.Version)
	if conf.Status.Version == "" {
		logger.Warn("The cluster was created by a older version of kwokctl, " +
			"please recreate the cluster",
		)
		conf.Status.Version = "0.0.0"
	} else if conf.Status.Version != consts.Version {
		currentVer, err := version.ParseVersion(conf.Status.Version)
		if err != nil {
			return nil, err
		}
		ver, err := version.ParseVersion(consts.Version)
		if err != nil {
			return nil, err
		}
		switch {
		case currentVer.LT(ver):
			logger.Warn("The cluster was created by a older version of kwokctl, "+
				"please recreate the cluster",
				"createdByVersion", conf.Status.Version,
			)
		case currentVer.GT(ver):
			logger.Warn("The cluster was created by a newer version of kwokctl, "+
				"please upgrade kwokctl or recreate the cluster",
				"createdByVersion", conf.Status.Version,
			)
		}
	}

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
	c.conf = conf.DeepCopy()
	return nil
}

// Save saves the cluster config
func (c *Cluster) Save(ctx context.Context) error {
	if c.conf == nil {
		return nil
	}

	if c.IsDryRun() {
		dryrun.PrintMessage("# Save cluster config to %s", c.GetWorkdirPath(ConfigName))
		return nil
	}

	var objs []config.InternalObject
	conf := c.conf.DeepCopy()
	if conf.Status.Version == "" {
		conf.Status.Version = consts.Version
	}
	objs = append(objs, conf)

	kwokConfigs := config.FilterWithTypeFromContext[*internalversion.KwokConfiguration](ctx)
	objs = appendIntoInternalObjects(objs, kwokConfigs...)

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.StageKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.Stage](ctx)
		if len(stages) == 0 {
			var nodeStatusUpdateFrequencyMilliseconds = conf.Options.NodeStatusUpdateFrequencyMilliseconds
			var nodeLeaseDurationSeconds = conf.Options.NodeLeaseDurationSeconds

			if conf.Options.Runtime == consts.RuntimeTypeKind ||
				conf.Options.Runtime == consts.RuntimeTypeKindPodman ||
				conf.Options.Runtime == consts.RuntimeTypeKindNerdctl ||
				conf.Options.Runtime == consts.RuntimeTypeKindLima ||
				conf.Options.Runtime == consts.RuntimeTypeKindFinch {
				nodeStatusUpdateFrequencyMilliseconds = 0
				nodeLeaseDurationSeconds = 40
			}

			defaultNodeStages, err := getDefaultNodeStages(nodeStatusUpdateFrequencyMilliseconds, nodeLeaseDurationSeconds != 0)
			if err != nil {
				return err
			}
			objs = appendIntoInternalObjects(objs, defaultNodeStages...)

			defaultPodStages, err := getDefaultPodStages()
			if err != nil {
				return err
			}
			objs = appendIntoInternalObjects(objs, defaultPodStages...)
		} else {
			objs = appendIntoInternalObjects(objs, stages...)
		}
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.MetricKind) {
		metrics := config.FilterWithTypeFromContext[*internalversion.Metric](ctx)
		if len(metrics) != 0 {
			objs = appendIntoInternalObjects(objs, metrics...)
		} else if c.conf.Options.EnableMetricsServer {
			m, err := config.UnmarshalWithType[*internalversion.Metric, string](resource.DefaultMetricsResource)
			if err != nil {
				return err
			}
			objs = appendIntoInternalObjects(objs, m)
		}
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.ResourceUsageKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.ResourceUsage](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.ClusterResourceUsageKind) {
		cru := config.FilterWithTypeFromContext[*internalversion.ClusterResourceUsage](ctx)
		if len(cru) != 0 {
			objs = appendIntoInternalObjects(objs, cru...)
		} else if c.conf.Options.EnableMetricsServer {
			m, err := config.UnmarshalWithType[*internalversion.ClusterResourceUsage, string](usage.DefaultUsageFromAnnotation)
			if err != nil {
				return err
			}
			objs = appendIntoInternalObjects(objs, m)
		}
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.AttachKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.Attach](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.ClusterAttachKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.ClusterAttach](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.ExecKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.Exec](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.ClusterExecKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.ClusterExec](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.LogsKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.Logs](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.ClusterLogsKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.ClusterLogs](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.PortForwardKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.PortForward](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.ClusterPortForwardKind) {
		stages := config.FilterWithTypeFromContext[*internalversion.ClusterPortForward](ctx)
		objs = appendIntoInternalObjects(objs, stages...)
	}

	return config.Save(ctx, c.GetWorkdirPath(ConfigName), objs)
}

func appendIntoInternalObjects[T config.InternalObject](objs []config.InternalObject, a ...T) []config.InternalObject {
	for _, o := range a {
		objs = append(objs, o)
	}
	return objs
}

func getDefaultNodeStages(updateFrequency int64, lease bool) ([]config.InternalObject, error) {
	objs := []config.InternalObject{}

	nodeInitStage, err := config.UnmarshalWithType[*internalversion.Stage](nodefast.DefaultNodeInit)
	if err != nil {
		return nil, err
	}
	objs = append(objs, nodeInitStage)

	rawHeartbeat := nodeheartbeat.DefaultNodeHeartbeat
	if lease {
		rawHeartbeat = nodeheartbeatwithlease.DefaultNodeHeartbeatWithLease
	}

	nodeHeartbeatStage, err := config.UnmarshalWithType[*internalversion.Stage](rawHeartbeat)
	if err != nil {
		return nil, err
	}

	if updateFrequency > 0 {
		durationMilliseconds := format.ElemOrDefault(nodeHeartbeatStage.Spec.Delay.DurationMilliseconds)
		jitterDurationMilliseconds := format.ElemOrDefault(nodeHeartbeatStage.Spec.Delay.JitterDurationMilliseconds)
		if updateFrequency > durationMilliseconds {
			durationMilliseconds = updateFrequency
		}
		if jitterUpdateFrequency := updateFrequency + updateFrequency/10; jitterUpdateFrequency > jitterDurationMilliseconds {
			jitterDurationMilliseconds = jitterUpdateFrequency
		}
		nodeHeartbeatStage.Spec.Delay.DurationMilliseconds = format.Ptr(durationMilliseconds)
		nodeHeartbeatStage.Spec.Delay.JitterDurationMilliseconds = format.Ptr(jitterDurationMilliseconds)
	}

	objs = append(objs, nodeHeartbeatStage)
	return objs, nil
}

func getDefaultPodStages() ([]config.InternalObject, error) {
	return slices.MapWithError([]string{
		podfast.DefaultPodReady,
		podfast.DefaultPodComplete,
		podfast.DefaultPodDelete,
	}, config.UnmarshalWithType[config.InternalObject, string])
}

// KubectlPath returns the path to the kubectl binary. It first tries to find kubectl in the system PATH.
// If not found, it will download and install the kubectl binary using the configured version.
// Returns the path to the kubectl binary or an error if it cannot be found or installed.
func (c *Cluster) KubectlPath(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options

	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		kubectlPath, err = c.EnsureBinary(ctx, "kubectl", conf.KubectlBinary)
		if err != nil {
			return "", err
		}
	}
	return kubectlPath, nil
}

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
	return c.MkdirAll(c.Workdir())
}

// Uninstall uninstalls the cluster.
func (c *Cluster) Uninstall(ctx context.Context) error {
	// cleanup workdir
	return c.RemoveAll(c.Workdir())
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
			"method", "get",
			"path", "/healthz",
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

// ListComponents returns the list of components
func (c *Cluster) ListComponents(ctx context.Context) ([]internalversion.Component, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	return config.Components, nil
}

// Components returns component names of the cluster
func (c *Cluster) Components(ctx context.Context) ([]string, error) {
	conf, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	enable := conf.Options.Enable
	disable := conf.Options.Disable

	if conf.Options.DisableKubeControllerManager {
		disable = append(disable, consts.ComponentKubeControllerManager)
	}

	if conf.Options.DisableKubeScheduler {
		disable = append(disable, consts.ComponentKubeScheduler)
	}

	if conf.Options.EnableMetricsServer {
		enable = append(enable, consts.ComponentMetricsServer)
	}
	if conf.Options.KubeApiserverInsecurePort != 0 {
		enable = append(enable, consts.ComponentKubeApiserverInsecureProxy)
	}
	if conf.Options.DashboardPort != 0 {
		enable = append(enable, consts.ComponentDashboard)
	}
	if conf.Options.PrometheusPort != 0 {
		enable = append(enable, consts.ComponentPrometheus)
	}
	if conf.Options.JaegerPort != 0 {
		enable = append(enable, consts.ComponentJaeger)
	}

	components := conf.Options.Components
	if len(enable) != 0 {
		components = append(components, enable...)
	}

	if len(disable) != 0 {
		components = slices.Filter(components, func(s string) bool {
			return !slices.Contains(disable, s)
		})
	}

	components = slices.Unique(components)

	_, hasEtcd := slices.Find(components, func(s string) bool {
		return s == consts.ComponentEtcd
	})
	if !hasEtcd {
		return nil, fmt.Errorf("etcd must be enabled")
	}

	_, hasApiserver := slices.Find(components, func(s string) bool {
		return s == consts.ComponentKubeApiserver
	})
	if !hasApiserver {
		return nil, fmt.Errorf("kube-apiserver must be enabled")
	}

	return components, nil
}

// Kubectl runs kubectl.
func (c *Cluster) Kubectl(ctx context.Context, args ...string) error {
	kubectlPath, err := c.KubectlPath(ctx)
	if err != nil {
		return err
	}

	return c.Exec(ctx, kubectlPath, args...)
}

// KubectlInCluster runs kubectl in the cluster.
func (c *Cluster) KubectlInCluster(ctx context.Context, args ...string) error {
	kubectlPath, err := c.KubectlPath(ctx)
	if err != nil {
		return err
	}

	return c.Exec(ctx, kubectlPath, append([]string{"--kubeconfig", c.GetWorkdirPath(InHostKubeconfigName)}, args...)...)
}

// AuditLogs returns the audit logs of the cluster.
func (c *Cluster) AuditLogs(ctx context.Context, out io.Writer) error {
	logs := c.GetLogPath(AuditLogName)
	if c.IsDryRun() {
		if file, ok := dryrun.IsCatToFileWriter(out); ok {
			dryrun.PrintMessage("cp %s %s", logs, file)
		} else {
			dryrun.PrintMessage("cat %s", logs)
		}
		return nil
	}

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
	if c.IsDryRun() {
		dryrun.PrintMessage("tail -f %s", logs)
		return nil
	}

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

func (c *Cluster) etcdctlPath(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options
	etcdctlPath, err := c.EnsureBinary(ctx, "etcdctl", conf.EtcdctlBinary)
	if err != nil {
		return "", err
	}
	return etcdctlPath, nil
}

// Etcdctl runs etcdctl.
func (c *Cluster) Etcdctl(ctx context.Context, args ...string) error {
	etcdctlPath, err := c.etcdctlPath(ctx)
	if err != nil {
		return err
	}

	// If using versions earlier than v3.4, set `ETCDCTL_API=3` to use v3 API.
	ctx = exec.WithEnv(ctx, []string{"ETCDCTL_API=3"})
	return c.Exec(ctx, etcdctlPath, args...)
}

// GetClientset returns the clientset of the cluster.
func (c *Cluster) GetClientset(ctx context.Context) (client.Clientset, error) {
	if c.clientset != nil {
		return c.clientset, nil
	}
	kubeconfigPath := c.GetWorkdirPath(InHostKubeconfigName)
	clientset, err := client.NewClientset("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	c.clientset = clientset
	return clientset, nil
}

// IsDryRun returns true if the runtime is in dry-run mode
func (c *Cluster) IsDryRun() bool {
	return c.dryRun
}

// InitCRDs initializes the CRDs.
func (c *Cluster) InitCRDs(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	crds := conf.EnableCRDs
	if len(crds) == 0 {
		return nil
	}

	if c.IsDryRun() {
		dryrun.PrintMessage("# Init CRDs %s", strings.Join(crds, ","))
		return nil
	}

	clientset, err := c.GetClientset(ctx)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	for _, name := range crds {
		c, ok := crdDefines[name]
		if !ok {
			return fmt.Errorf("no crd define found for %s", name)
		}
		_, _ = buf.WriteString("\n---\n")
		_, _ = buf.Write(c)
	}

	logger := log.FromContext(ctx)
	ctx = log.NewContext(ctx, logger.With("crds", strings.Join(crds, ",")))

	loader, err := snapshot.NewLoader(snapshot.LoadConfig{
		Clientset: clientset,
		NoFilers:  true,
	})
	if err != nil {
		return err
	}

	decoder := yaml.NewDecoder(buf)
	err = loader.Load(ctx, decoder)
	if err != nil {
		return err
	}

	return nil
}

var crdDefines = map[string][]byte{
	v1alpha1.StageKind:                crd.Stage,
	v1alpha1.AttachKind:               crd.Attach,
	v1alpha1.ClusterAttachKind:        crd.ClusterAttach,
	v1alpha1.ExecKind:                 crd.Exec,
	v1alpha1.ClusterExecKind:          crd.ClusterExec,
	v1alpha1.PortForwardKind:          crd.PortForward,
	v1alpha1.ClusterPortForwardKind:   crd.ClusterPortForward,
	v1alpha1.LogsKind:                 crd.Logs,
	v1alpha1.ClusterLogsKind:          crd.ClusterLogs,
	v1alpha1.ResourceUsageKind:        crd.ResourceUsage,
	v1alpha1.ClusterResourceUsageKind: crd.ClusterResourceUsage,
	v1alpha1.MetricKind:               crd.Metric,
}
