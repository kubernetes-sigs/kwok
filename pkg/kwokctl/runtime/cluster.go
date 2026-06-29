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
	"context"
	"fmt"
	"slices"

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
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
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
	Kueue                   = "kueue.yaml"
	JobSet                  = "jobset.yaml"
	LWS                     = "lws.yaml"
	Descheduler             = "descheduler.yaml"
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
		name:    config.ClusterName(name),
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
	logger = logger.With(
		"kwokctlVersion", consts.Version,
	)
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
		dryrun.PrintMessagef("# Save cluster config to %s", c.GetWorkdirPath(ConfigName))
		return nil
	}

	components, err := c.Components(ctx)
	if err != nil {
		return err
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

	if slices.Contains(components, consts.ComponentMetricsServer) {
		if !slices.Contains(conf.Options.EnableCRDs, v1alpha1.MetricKind) {
			metrics := config.FilterWithTypeFromContext[*internalversion.Metric](ctx)
			if len(metrics) != 0 {
				objs = appendIntoInternalObjects(objs, metrics...)
			} else {
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
			} else {
				m, err := config.UnmarshalWithType[*internalversion.ClusterResourceUsage, string](usage.DefaultUsageFromAnnotation)
				if err != nil {
					return err
				}
				objs = appendIntoInternalObjects(objs, m)
			}
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
		nodeHeartbeatStage.Spec.Delay.DurationMilliseconds = new(durationMilliseconds)
		nodeHeartbeatStage.Spec.Delay.JitterDurationMilliseconds = new(jitterDurationMilliseconds)
	}

	objs = append(objs, nodeHeartbeatStage)
	return objs, nil
}

func getDefaultPodStages() ([]config.InternalObject, error) {
	return utilsslices.MapWithError([]string{
		podfast.DefaultPodReady,
		podfast.DefaultPodComplete,
		podfast.DefaultPodDelete,
	}, config.UnmarshalWithType[config.InternalObject, string])
}

// GetComponent returns the component by name
func (c *Cluster) GetComponent(ctx context.Context, name string) (internalversion.Component, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return internalversion.Component{}, err
	}
	component, ok := utilsslices.Find(config.Components, func(component internalversion.Component) bool {
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

// getEnabledAndDisabledComponents builds the enable and disable component lists from config.
func (c *Cluster) getEnabledAndDisabledComponents(ctx context.Context) ([]string, []string, error) {
	conf, err := c.Config(ctx)
	if err != nil {
		return nil, nil, err
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
	if conf.Options.PrometheusPort != 0 {
		enable = append(enable, consts.ComponentPrometheus)
	}
	if conf.Options.JaegerPort != 0 {
		enable = append(enable, consts.ComponentJaeger)
	}
	if conf.Options.KueuevizPort != 0 {
		enable = append(enable, consts.ComponentKueue, consts.ComponentKueueviz)
	}
	if slices.Contains(enable, consts.ComponentKueueviz) {
		enable = utilsslices.Filter(enable, func(s string) bool {
			return s != consts.ComponentKueueviz
		})
		enable = append(enable, consts.ComponentKueuevizFrontend, consts.ComponentKueuevizBackend)
	}
	return enable, disable, nil
}

// Components returns component names of the cluster
func (c *Cluster) Components(ctx context.Context) ([]string, error) {
	conf, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	enable, disable, err := c.getEnabledAndDisabledComponents(ctx)
	if err != nil {
		return nil, err
	}

	components := conf.Options.Components
	if len(enable) != 0 {
		components = append(components, enable...)
	}

	if len(disable) != 0 {
		components = utilsslices.Filter(components, func(s string) bool {
			return !slices.Contains(disable, s)
		})
	}

	components = utilsslices.Unique(components)

	_, hasEtcd := utilsslices.Find(components, func(s string) bool {
		return s == consts.ComponentEtcd
	})
	if !hasEtcd {
		return nil, fmt.Errorf("etcd must be enabled")
	}

	_, hasApiserver := utilsslices.Find(components, func(s string) bool {
		return s == consts.ComponentKubeApiserver
	})
	if !hasApiserver {
		return nil, fmt.Errorf("kube-apiserver must be enabled")
	}

	return components, nil
}

// GetWorkdirPath returns the path to the file in the workdir.
func (c *Cluster) GetWorkdirPath(name string) string {
	return utilspath.Join(c.workdir, name)
}

// GetBinPath returns the path to the given binary name.
func (c *Cluster) GetBinPath(name string) string {
	return utilspath.Join(c.workdir, "bin", name)
}

// GetLogPath returns the path of the given log name.
func (c *Cluster) GetLogPath(name string) string {
	return utilspath.Join(c.workdir, "logs", name)
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
