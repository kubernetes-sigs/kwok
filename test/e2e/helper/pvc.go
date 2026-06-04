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

package helper

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolumeClaimBuilder builds a PersistentVolumeClaim.
type PersistentVolumeClaimBuilder struct {
	pvc *corev1.PersistentVolumeClaim
}

// NewPersistentVolumeClaimBuilder returns a builder for a PVC with sensible defaults.
func NewPersistentVolumeClaimBuilder(name string) *PersistentVolumeClaimBuilder {
	storageRequest := resource.MustParse("1Gi")
	return &PersistentVolumeClaimBuilder{
		pvc: &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: storageRequest,
					},
				},
			},
		},
	}
}

// WithNamespace sets the namespace.
func (b PersistentVolumeClaimBuilder) WithNamespace(namespace string) *PersistentVolumeClaimBuilder {
	b.pvc.Namespace = namespace
	return &b
}

// WithStorageClassName sets the StorageClass name for the PVC.
func (b PersistentVolumeClaimBuilder) WithStorageClassName(name string) *PersistentVolumeClaimBuilder {
	b.pvc.Spec.StorageClassName = &name
	return &b
}

// WithStorage sets the requested storage size.
func (b PersistentVolumeClaimBuilder) WithStorage(size string) *PersistentVolumeClaimBuilder {
	b.pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(size)
	return &b
}

// WithAccessModes sets the access modes.
func (b PersistentVolumeClaimBuilder) WithAccessModes(modes ...corev1.PersistentVolumeAccessMode) *PersistentVolumeClaimBuilder {
	b.pvc.Spec.AccessModes = modes
	return &b
}

// Build returns the built PVC.
func (b PersistentVolumeClaimBuilder) Build() *corev1.PersistentVolumeClaim {
	return b.pvc.DeepCopy()
}
