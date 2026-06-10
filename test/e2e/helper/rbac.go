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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterRoleBuilder struct {
	role *rbacv1.ClusterRole
}

func NewClusterRoleBuilder(name string) *ClusterRoleBuilder {
	return &ClusterRoleBuilder{
		role: &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		},
	}
}

func (b ClusterRoleBuilder) WithRules(rules ...rbacv1.PolicyRule) *ClusterRoleBuilder {
	b.role.Rules = rules
	return &b
}

func (b ClusterRoleBuilder) Build() *rbacv1.ClusterRole {
	return b.role.DeepCopy()
}

type ClusterRoleBindingBuilder struct {
	binding *rbacv1.ClusterRoleBinding
}

func NewClusterRoleBindingBuilder(name string) *ClusterRoleBindingBuilder {
	return &ClusterRoleBindingBuilder{
		binding: &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		},
	}
}

func (b ClusterRoleBindingBuilder) WithRoleRef(roleRef rbacv1.RoleRef) *ClusterRoleBindingBuilder {
	b.binding.RoleRef = roleRef
	return &b
}

func (b ClusterRoleBindingBuilder) WithSubjects(subjects ...rbacv1.Subject) *ClusterRoleBindingBuilder {
	b.binding.Subjects = subjects
	return &b
}

func (b ClusterRoleBindingBuilder) Build() *rbacv1.ClusterRoleBinding {
	return b.binding.DeepCopy()
}
