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

package e2e

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
	"sigs.k8s.io/kwok/test/e2e/helper"

	_ "embed"
)

//go:embed impersonation.yaml
var impersonationCase []byte

// CaseImpersonation returns a feature that tests the impersonation functionality of stages.
func CaseImpersonation() *features.FeatureBuilder {
	username := "kwok-e2e-impersonated"

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: username,
		},
	}

	// Grant impersonate permission to kwok-controller so it can impersonate the target user
	impersonateCr := helper.NewClusterRoleBuilder("kwok-e2e-impersonation-impersonate").
		WithRules(rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"get", "list", "watch"},
		}, rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"users"},
			Verbs:     []string{"impersonate"},
		})

	impersonateCrb := helper.NewClusterRoleBindingBuilder("kwok-e2e-impersonation-impersonate-binding").
		WithRoleRef(rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "kwok-e2e-impersonation-impersonate",
		}).
		WithSubjects(rbacv1.Subject{
			Kind:      "ServiceAccount",
			APIGroup:  "",
			Name:      "kwok-controller",
			Namespace: "kube-system",
		})

	// Grant status patch permission to the impersonated user
	statusCr := helper.NewClusterRoleBuilder("kwok-e2e-impersonation-status-patch").
		WithRules(rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"namespaces/status"},
			Verbs:     []string{"patch", "update"},
		})

	statusCrb := helper.NewClusterRoleBindingBuilder("kwok-e2e-impersonation-status-patch-binding").
		WithRoleRef(rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "kwok-e2e-impersonation-status-patch",
		}).
		WithSubjects(rbacv1.Subject{
			Kind:     "User",
			APIGroup: rbacv1.GroupName,
			Name:     username,
		})

	b := features.New("Stage Impersonation").
		Assess("grant permissions", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			// Grant impersonate permission to kwok-controller
			if err = client.Create(ctx, impersonateCr.Build()); err != nil && !apierrors.IsAlreadyExists(err) {
				t.Fatal(err)
			}
			if err = client.Create(ctx, impersonateCrb.Build()); err != nil && !apierrors.IsAlreadyExists(err) {
				t.Fatal(err)
			}
			// Grant status patch permission to the impersonated user
			if err = client.Create(ctx, statusCr.Build()); err != nil && !apierrors.IsAlreadyExists(err) {
				t.Fatal(err)
			}
			if err = client.Create(ctx, statusCrb.Build()); err != nil && !apierrors.IsAlreadyExists(err) {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("create stages", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			err = v1alpha1.AddToScheme(client.GetScheme())
			if err != nil {
				t.Fatal(err)
			}

			var s *v1alpha1.Stage
			err = yaml.Unmarshal(impersonationCase, &s)
			if err != nil {
				t.Fatal(err)
			}

			err = client.Create(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			// Wait a bit to ensure the stage controllers have started and are watching for changes.
			log.FromContext(ctx).Info("waiting for stage controllers to start")
			time.Sleep(5 * time.Second)
			return ctx
		}).
		Assess("create namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			if err = client.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("verify condition patched", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			logger := log.FromContext(ctx)
			err = wait.For(func(ctx context.Context) (bool, error) {
				var n corev1.Namespace
				if err = client.Get(ctx, ns.Name, "", &n); err != nil {
					logger.Error("failed to get namespace", err, "namespace", ns.Name)
					return false, nil
				}
				for _, cond := range n.Status.Conditions {
					if cond.Type == "kwok.x-k8s.io/impersonated" && cond.Status == corev1.ConditionTrue {
						return true, nil
					}
				}
				logger.Info("waiting for impersonated condition", "namespace", ns.Name)
				return false, nil
			},
				wait.WithContext(ctx),
				wait.WithTimeout(time.Minute),
			)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("revoke permissions", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			// Revoke impersonate permission
			if err = client.Delete(ctx, impersonateCrb.Build()); err != nil && !apierrors.IsNotFound(err) {
				t.Fatal(err)
			}
			if err = client.Delete(ctx, impersonateCr.Build()); err != nil && !apierrors.IsNotFound(err) {
				t.Fatal(err)
			}
			// Revoke status patch permission
			if err = client.Delete(ctx, statusCr.Build()); err != nil && !apierrors.IsNotFound(err) {
				t.Fatal(err)
			}
			if err = client.Delete(ctx, statusCrb.Build()); err != nil && !apierrors.IsNotFound(err) {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("delete namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			if err = client.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
				t.Fatal(err)
			}
			return ctx
		})

	return b
}
