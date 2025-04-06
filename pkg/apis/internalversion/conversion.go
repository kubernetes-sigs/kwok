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

package internalversion

import (
	"k8s.io/apimachinery/pkg/conversion"

	configv1alpha1 "sigs.k8s.io/kwok/pkg/apis/config/v1alpha1"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

// ConvertToV1alpha1KwokctlConfiguration converts an internal version KwokctlConfiguration to a v1alpha1.KwokctlConfiguration.
func ConvertToV1alpha1KwokctlConfiguration(in *KwokctlConfiguration) (*configv1alpha1.KwokctlConfiguration, error) {
	var out configv1alpha1.KwokctlConfiguration
	out.APIVersion = configv1alpha1.GroupVersion.String()
	out.Kind = configv1alpha1.KwokctlConfigurationKind
	err := Convert_internalversion_KwokctlConfiguration_To_v1alpha1_KwokctlConfiguration(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalKwokctlConfiguration converts a v1alpha1.KwokctlConfiguration to an internal version.
func ConvertToInternalKwokctlConfiguration(in *configv1alpha1.KwokctlConfiguration) (*KwokctlConfiguration, error) {
	var out KwokctlConfiguration
	err := Convert_v1alpha1_KwokctlConfiguration_To_internalversion_KwokctlConfiguration(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1alpha1KwokctlResource converts an internal version KwokctlResource to a v1alpha1.KwokctlResource.
func ConvertToV1alpha1KwokctlResource(in *KwokctlResource) (*configv1alpha1.KwokctlResource, error) {
	var out configv1alpha1.KwokctlResource
	out.APIVersion = configv1alpha1.GroupVersion.String()
	out.Kind = configv1alpha1.KwokctlResourceKind
	err := Convert_internalversion_KwokctlResource_To_v1alpha1_KwokctlResource(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalKwokctlResource converts a v1alpha1.KwokctlResource to an internal version.
func ConvertToInternalKwokctlResource(in *configv1alpha1.KwokctlResource) (*KwokctlResource, error) {
	var out KwokctlResource
	err := Convert_v1alpha1_KwokctlResource_To_internalversion_KwokctlResource(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1alpha1KwokConfiguration converts an internal version KwokConfiguration to a v1alpha1.KwokConfiguration.
func ConvertToV1alpha1KwokConfiguration(in *KwokConfiguration) (*configv1alpha1.KwokConfiguration, error) {
	var out configv1alpha1.KwokConfiguration
	out.APIVersion = configv1alpha1.GroupVersion.String()
	out.Kind = configv1alpha1.KwokConfigurationKind
	err := Convert_internalversion_KwokConfiguration_To_v1alpha1_KwokConfiguration(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalKwokConfiguration converts a v1alpha1.KwokConfiguration to an internal version.
func ConvertToInternalKwokConfiguration(in *configv1alpha1.KwokConfiguration) (*KwokConfiguration, error) {
	var out KwokConfiguration
	err := Convert_v1alpha1_KwokConfiguration_To_internalversion_KwokConfiguration(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1alpha1Stage converts an internal version Stage to a v1alpha1.Stage.
func ConvertToV1alpha1Stage(in *Stage) (*v1alpha1.Stage, error) {
	var out v1alpha1.Stage
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.StageKind
	err := Convert_internalversion_Stage_To_v1alpha1_Stage(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalStage converts a v1alpha1.Stage to an internal version.
func ConvertToInternalStage(in *v1alpha1.Stage) (*Stage, error) {
	var out Stage
	err := Convert_v1alpha1_Stage_To_internalversion_Stage(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1Alpha1ClusterPortForward converts an internal version ClusterPortForward to a v1alpha1.ClusterPortForward.
func ConvertToV1Alpha1ClusterPortForward(in *ClusterPortForward) (*v1alpha1.ClusterPortForward, error) {
	var out v1alpha1.ClusterPortForward
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.ClusterPortForwardKind
	err := Convert_internalversion_ClusterPortForward_To_v1alpha1_ClusterPortForward(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalClusterPortForward converts a v1alpha1.ClusterPortForward to an internal version.
func ConvertToInternalClusterPortForward(in *v1alpha1.ClusterPortForward) (*ClusterPortForward, error) {
	var out ClusterPortForward
	err := Convert_v1alpha1_ClusterPortForward_To_internalversion_ClusterPortForward(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1Alpha1PortForward converts an internal version PortForward to a v1alpha1.PortForward.
func ConvertToV1Alpha1PortForward(in *PortForward) (*v1alpha1.PortForward, error) {
	var out v1alpha1.PortForward
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.PortForwardKind
	err := Convert_internalversion_PortForward_To_v1alpha1_PortForward(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalPortForward converts a v1alpha1.PortForward to an internal version.
func ConvertToInternalPortForward(in *v1alpha1.PortForward) (*PortForward, error) {
	var out PortForward
	err := Convert_v1alpha1_PortForward_To_internalversion_PortForward(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1Alpha1ClusterExec converts an internal version ClusterExec to a v1alpha1.ClusterExec.
func ConvertToV1Alpha1ClusterExec(in *ClusterExec) (*v1alpha1.ClusterExec, error) {
	var out v1alpha1.ClusterExec
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.ClusterExecKind
	err := Convert_internalversion_ClusterExec_To_v1alpha1_ClusterExec(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalClusterExec converts a v1alpha1.ClusterExec to an internal version.
func ConvertToInternalClusterExec(in *v1alpha1.ClusterExec) (*ClusterExec, error) {
	var out ClusterExec
	err := Convert_v1alpha1_ClusterExec_To_internalversion_ClusterExec(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalClusterLogs converts a v1alpha1.ClusterLogs to an internal version.
func ConvertToInternalClusterLogs(in *v1alpha1.ClusterLogs) (*ClusterLogs, error) {
	var out ClusterLogs
	err := Convert_v1alpha1_ClusterLogs_To_internalversion_ClusterLogs(in, &out, nil)
	if err != nil {
		return nil, err
	}
	for i := range out.Spec.Logs {
		logsFile := out.Spec.Logs[i].LogsFile
		if logsFile != "" {
			out.Spec.Logs[i].LogsFile, err = path.Expand(logsFile)
			if err != nil {
				return nil, err
			}
		}
		previousLogsFile := out.Spec.Logs[i].PreviousLogsFile
		if previousLogsFile != "" {
			out.Spec.Logs[i].PreviousLogsFile, err = path.Expand(previousLogsFile)
			if err != nil {
				return nil, err
			}
		}
	}
	return &out, nil
}

// ConvertToV1Alpha1Exec converts an internal version Exec to a v1alpha1.Exec.
func ConvertToV1Alpha1Exec(in *Exec) (*v1alpha1.Exec, error) {
	var out v1alpha1.Exec
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.ExecKind
	err := Convert_internalversion_Exec_To_v1alpha1_Exec(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalExec converts a v1alpha1.Exec to an internal version.
func ConvertToInternalExec(in *v1alpha1.Exec) (*Exec, error) {
	var out Exec
	err := Convert_v1alpha1_Exec_To_internalversion_Exec(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1Alpha1Logs converts an internal version Logs to a v1alpha1.Logs.
func ConvertToV1Alpha1Logs(in *Logs) (*v1alpha1.Logs, error) {
	var out v1alpha1.Logs
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.LogsKind
	err := Convert_internalversion_Logs_To_v1alpha1_Logs(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1Alpha1ClusterLogs converts an internal version ClusterLogs to a v1alpha1.ClusterLogs.
func ConvertToV1Alpha1ClusterLogs(in *ClusterLogs) (*v1alpha1.ClusterLogs, error) {
	var out v1alpha1.ClusterLogs
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.ClusterLogsKind
	err := Convert_internalversion_ClusterLogs_To_v1alpha1_ClusterLogs(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalLogs converts a v1alpha1.Logs to an internal version.
func ConvertToInternalLogs(in *v1alpha1.Logs) (*Logs, error) {
	var out Logs
	err := Convert_v1alpha1_Logs_To_internalversion_Logs(in, &out, nil)
	if err != nil {
		return nil, err
	}
	for i := range out.Spec.Logs {
		logsFile := out.Spec.Logs[i].LogsFile
		if logsFile != "" {
			out.Spec.Logs[i].LogsFile, err = path.Expand(logsFile)
			if err != nil {
				return nil, err
			}
		}
		previousLogsFile := out.Spec.Logs[i].PreviousLogsFile
		if previousLogsFile != "" {
			out.Spec.Logs[i].PreviousLogsFile, err = path.Expand(previousLogsFile)
			if err != nil {
				return nil, err
			}
		}
	}
	return &out, nil
}

// ConvertToV1Alpha1Attach converts an internal version Attach to a v1alpha1.Attach.
func ConvertToV1Alpha1Attach(in *Attach) (*v1alpha1.Attach, error) {
	var out v1alpha1.Attach
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.AttachKind
	err := Convert_internalversion_Attach_To_v1alpha1_Attach(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1Alpha1ClusterAttach converts an internal version ClusterAttach to a v1alpha1.ClusterAttach.
func ConvertToV1Alpha1ClusterAttach(in *ClusterAttach) (*v1alpha1.ClusterAttach, error) {
	var out v1alpha1.ClusterAttach
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.ClusterAttachKind
	err := Convert_internalversion_ClusterAttach_To_v1alpha1_ClusterAttach(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalClusterAttach converts a v1alpha1.ClusterAttach to an internal version.
func ConvertToInternalClusterAttach(in *v1alpha1.ClusterAttach) (*ClusterAttach, error) {
	var out ClusterAttach
	err := Convert_v1alpha1_ClusterAttach_To_internalversion_ClusterAttach(in, &out, nil)
	if err != nil {
		return nil, err
	}
	for i := range out.Spec.Attaches {
		logsFile := out.Spec.Attaches[i].LogsFile
		if logsFile != "" {
			out.Spec.Attaches[i].LogsFile, err = path.Expand(logsFile)
			if err != nil {
				return nil, err
			}
		}
	}
	return &out, nil
}

// ConvertToInternalAttach converts a v1alpha1.Attach to an internal version.
func ConvertToInternalAttach(in *v1alpha1.Attach) (*Attach, error) {
	var out Attach
	err := Convert_v1alpha1_Attach_To_internalversion_Attach(in, &out, nil)
	if err != nil {
		return nil, err
	}
	for i := range out.Spec.Attaches {
		logsFile := out.Spec.Attaches[i].LogsFile
		if logsFile != "" {
			out.Spec.Attaches[i].LogsFile, err = path.Expand(logsFile)
			if err != nil {
				return nil, err
			}
		}
	}
	return &out, nil
}

// ConvertToV1Alpha1ResourceUsage converts an internal version ResourceUsage to a v1alpha1.ResourceUsage.
func ConvertToV1Alpha1ResourceUsage(in *ResourceUsage) (*v1alpha1.ResourceUsage, error) {
	var out v1alpha1.ResourceUsage
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.ResourceUsageKind
	err := Convert_internalversion_ResourceUsage_To_v1alpha1_ResourceUsage(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalResourceUsage converts a v1alpha1.ResourceUsage to an internal version.
func ConvertToInternalResourceUsage(in *v1alpha1.ResourceUsage) (*ResourceUsage, error) {
	var out ResourceUsage
	err := Convert_v1alpha1_ResourceUsage_To_internalversion_ResourceUsage(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1Alpha1ClusterResourceUsage converts an internal version ClusterResourceUsage to a v1alpha1.ClusterResourceUsage.
func ConvertToV1Alpha1ClusterResourceUsage(in *ClusterResourceUsage) (*v1alpha1.ClusterResourceUsage, error) {
	var out v1alpha1.ClusterResourceUsage
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.ClusterResourceUsageKind
	err := Convert_internalversion_ClusterResourceUsage_To_v1alpha1_ClusterResourceUsage(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalClusterResourceUsage converts a v1alpha1.ClusterResourceUsage to an internal version.
func ConvertToInternalClusterResourceUsage(in *v1alpha1.ClusterResourceUsage) (*ClusterResourceUsage, error) {
	var out ClusterResourceUsage
	err := Convert_v1alpha1_ClusterResourceUsage_To_internalversion_ClusterResourceUsage(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToV1Alpha1Metric converts an internal version Metric to a v1alpha1.Metric.
func ConvertToV1Alpha1Metric(in *Metric) (*v1alpha1.Metric, error) {
	var out v1alpha1.Metric
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.MetricKind
	err := Convert_internalversion_Metric_To_v1alpha1_Metric(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertToInternalMetric converts a v1alpha1.Metric to an internal version.
func ConvertToInternalMetric(in *v1alpha1.Metric) (*Metric, error) {
	var out Metric
	v1alpha1.SetObjectDefaults_Metric(in)
	err := Convert_v1alpha1_Metric_To_internalversion_Metric(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Convert_v1alpha1_StageNext_To_internalversion_StageNext converts a v1alpha1.StageNext to an internal version.
func Convert_v1alpha1_StageNext_To_internalversion_StageNext(in *v1alpha1.StageNext, out *StageNext, s conversion.Scope) error {
	err := autoConvert_v1alpha1_StageNext_To_internalversion_StageNext(in, out, s)
	if err != nil {
		return err
	}

	if in.StatusTemplate != "" && len(out.Patches) == 0 {
		template := in.StatusTemplate

		subresource := "status"
		if in.StatusSubresource != nil {
			subresource = *in.StatusSubresource
		}

		var impersonation *ImpersonationConfig
		if in.StatusPatchAs != nil {
			impersonation = &ImpersonationConfig{
				Username: in.StatusPatchAs.Username,
			}
		}
		patch := StagePatch{
			Subresource:   subresource,
			Root:          "status",
			Template:      template,
			Impersonation: impersonation,
		}

		out.Patches = []StagePatch{patch}
	}
	return nil
}

// Convert_internalversion_StageNext_To_v1alpha1_StageNext is an autogenerated conversion function.
func Convert_internalversion_StageNext_To_v1alpha1_StageNext(in *StageNext, out *v1alpha1.StageNext, s conversion.Scope) error {
	err := autoConvert_internalversion_StageNext_To_v1alpha1_StageNext(in, out, s)
	if err != nil {
		return err
	}

	if len(in.Patches) != 0 {
		patch := in.Patches[0]
		if patch.Root == "status" {
			out.StatusSubresource = &patch.Subresource
			out.StatusTemplate = patch.Template
			if patch.Impersonation != nil {
				out.StatusPatchAs = &v1alpha1.ImpersonationConfig{
					Username: patch.Impersonation.Username,
				}
			}
		}
	}
	return nil
}

// Convert_internalversion_StagePatch_To_v1alpha1_StagePatch converts an internal version StagePatch to a v1alpha1.StagePatch.
func Convert_internalversion_StagePatch_To_v1alpha1_StagePatch(in *StagePatch, out *v1alpha1.StagePatch, s conversion.Scope) error {
	return autoConvert_internalversion_StagePatch_To_v1alpha1_StagePatch(in, out, s)
}
