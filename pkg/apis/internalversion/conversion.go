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

	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
)

func Convert_v1alpha1_KwokctlConfigurationOptions_To_internalversion_KwokctlConfigurationOptions(in *v1alpha1.KwokctlConfigurationOptions, out *KwokctlConfigurationOptions, s conversion.Scope) error {
	return autoConvert_v1alpha1_KwokctlConfigurationOptions_To_internalversion_KwokctlConfigurationOptions(in, out, s)
}

func ConvertToV1alpha1KwokctlConfiguration(in *KwokctlConfiguration) (*v1alpha1.KwokctlConfiguration, error) {
	var out v1alpha1.KwokctlConfiguration
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.KwokctlConfigurationKind
	err := Convert_internalversion_KwokctlConfiguration_To_v1alpha1_KwokctlConfiguration(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func ConvertToInternalVersionKwokctlConfiguration(in *v1alpha1.KwokctlConfiguration) (*KwokctlConfiguration, error) {
	var out KwokctlConfiguration
	err := Convert_v1alpha1_KwokctlConfiguration_To_internalversion_KwokctlConfiguration(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func ConvertToV1alpha1KwokConfiguration(in *KwokConfiguration) (*v1alpha1.KwokConfiguration, error) {
	var out v1alpha1.KwokConfiguration
	out.APIVersion = v1alpha1.GroupVersion.String()
	out.Kind = v1alpha1.KwokConfigurationKind
	err := Convert_internalversion_KwokConfiguration_To_v1alpha1_KwokConfiguration(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func ConvertToInternalVersionKwokConfiguration(in *v1alpha1.KwokConfiguration) (*KwokConfiguration, error) {
	var out KwokConfiguration
	err := Convert_v1alpha1_KwokConfiguration_To_internalversion_KwokConfiguration(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

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

func ConvertToInternalVersionStage(in *v1alpha1.Stage) (*Stage, error) {
	var out Stage
	err := Convert_v1alpha1_Stage_To_internalversion_Stage(in, &out, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
