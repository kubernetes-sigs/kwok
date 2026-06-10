/*
Copyright 2023 The Kubernetes Authors.

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
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/log"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
)

// nodeIsReady returns a function that checks if a node is ready
func nodeIsReady(name string) func(obj k8s.Object) bool {
	return func(obj k8s.Object) bool {
		node := obj.(*corev1.Node)
		if node.Name != name {
			return false
		}
		cond, ok := utilsslices.Find(node.Status.Conditions, func(cond corev1.NodeCondition) bool {
			return cond.Type == corev1.NodeReady
		})
		if ok && cond.Status == corev1.ConditionTrue {
			return true
		}
		return false
	}
}

// CreateNode creates a node and waits for it to be ready
func CreateNode(node *corev1.Node) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		t.Log("creating node", node.Name)
		err = client.Create(ctx, node)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("waiting for node to be ready", node.Name)
		err = wait.For(
			conditions.New(client).ResourceMatch(node, nodeIsReady(node.Name)),
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("node is ready", node.Name)
		return ctx
	}
}

// DeleteNode deletes a node
func DeleteNode(node *corev1.Node) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		t.Log("deleting node", node.Name)
		err = client.Delete(ctx, node)
		if err != nil {
			t.Fatal(err)
		}

		err = wait.For(
			conditions.New(client).ResourceDeleted(node),
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

// CreatePod creates a pod and waits for it to be ready
func CreatePod(pod *corev1.Pod) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		shouldHasPodScheduled := pod.Spec.NodeName == ""

		t.Log("creating pod", log.KObj(pod))
		err = client.Create(ctx, pod)
		if err != nil {
			t.Fatal(err)
		}

		t.Log("waiting for pod to be ready", log.KObj(pod))
		err = wait.For(
			conditions.New(client).PodConditionMatch(pod, corev1.PodReady, corev1.ConditionTrue),
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}

		err = client.Get(ctx, pod.GetName(), pod.GetNamespace(), pod)
		if err != nil {
			t.Fatal(err)
		}

		if pod.Spec.NodeName == "" {
			t.Fatal("pod node name is empty", log.KObj(pod))
		}

		_, hasPodScheduled := utilsslices.Find(pod.Status.Conditions, func(cond corev1.PodCondition) bool {
			return cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionTrue
		})
		if shouldHasPodScheduled {
			if !hasPodScheduled {
				t.Fatal("pod is scheduled, but there are no PodScheduled conditions", log.KObj(pod))
			}
		} else {
			if hasPodScheduled {
				t.Fatal("Pod is not scheduled, but there are PodScheduled conditions", log.KObj(pod))
			}
		}

		if pod.Status.PodIP != "" {
			if pod.Spec.HostNetwork {
				if pod.Status.PodIP != pod.Status.HostIP {
					t.Errorf("pod ip %q is not equal to host ip %q: %s", pod.Status.PodIP, pod.Status.HostIP, log.KObj(pod))
				}
			} else {
				if pod.Status.PodIP == pod.Status.HostIP {
					t.Errorf("pod ip %q is equal to host ip %q: %s", pod.Status.PodIP, pod.Status.HostIP, log.KObj(pod))
				}

				var node corev1.Node
				err = client.Get(ctx, pod.Spec.NodeName, "", &node)
				if err != nil {
					t.Fatal(err)
				}

				if node.Spec.PodCIDR != "" {
					_, ipnet, err := net.ParseCIDR(node.Spec.PodCIDR)
					if err != nil {
						t.Errorf("failed to parse pod cidr %q in %q", node.Spec.PodCIDR, node.Name)
					}

					ip := net.ParseIP(pod.Status.PodIP)
					if !ipnet.Contains(ip) {
						t.Errorf("pod ip %q is not in pod cidr %q in %q: %s", pod.Status.PodIP, node.Spec.PodCIDR, node.Name, log.KObj(pod))
					}
				}
			}
		}

		t.Log("pod is ready", log.KObj(pod))
		return ctx
	}
}

// DeletePod deletes a pod
func DeletePod(pod *corev1.Pod) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		t.Log("deleting pod", log.KObj(pod))
		err = client.Delete(ctx, pod)
		if err != nil {
			t.Fatal(err)
		}

		err = wait.For(
			conditions.New(client).ResourceDeleted(pod),
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

// WaitForAllNodesReady waits for all nodes to be ready
func WaitForAllNodesReady() env.Func {
	return func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}

		logger := log.FromContext(ctx)

		var list corev1.NodeList
		err = wait.For(
			func(ctx context.Context) (done bool, err error) {
				if err = client.List(ctx, &list); err != nil {
					logger.Error("failed to list nodes",
						"err", err,
					)
					return false, nil
				}

				metaList, err := meta.ExtractList(&list)
				if err != nil {
					logger.Error("failed to extract list",
						"err", err,
					)
					return false, nil
				}
				if len(metaList) == 0 {
					logger.Error("no node found")
					return false, nil
				}

				notReady := []string{}
				for _, obj := range metaList {
					node := obj.(*corev1.Node)
					cond, ok := utilsslices.Find(node.Status.Conditions, func(cond corev1.NodeCondition) bool {
						return cond.Type == corev1.NodeReady
					})
					if !ok || cond.Status != corev1.ConditionTrue {
						notReady = append(notReady, node.Name)
					}
				}
				if len(notReady) != 0 {
					logger.Error("not ready nodes",
						"err", notReady,
					)
					return false, nil
				}

				return true, nil
			},
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}

// WaitForAllPodsReady waits for all pods to be ready
func WaitForAllPodsReady() env.Func {
	return func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}

		logger := log.FromContext(ctx)

		var list corev1.PodList
		err = wait.For(
			func(ctx context.Context) (done bool, err error) {
				if err = client.List(ctx, &list); err != nil {
					logger.Error("failed to list pods",
						"err", err,
					)
					return false, nil
				}

				metaList, err := meta.ExtractList(&list)
				if err != nil {
					logger.Error("failed to extract list",
						"err", err,
					)
					return false, nil
				}
				if len(metaList) == 0 {
					logger.Error("no pod found")
					return false, nil
				}

				notReady := []string{}
				for _, obj := range metaList {
					pod := obj.(*corev1.Pod)
					// On Kind, ignore pods in kube-system and local-path-storage namespaces
					if pod.Namespace == "kube-system" || pod.Namespace == "local-path-storage" {
						continue
					}
					if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
						notReady = append(notReady, log.KObj(pod).String())
					}
				}
				if len(notReady) != 0 {
					logger.Error("not ready pods",
						"err", notReady,
					)
					return false, nil
				}

				return true, nil
			},
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}

// CreatePVC creates a PVC and waits for it to be Bound.
func CreatePVC(pvc *corev1.PersistentVolumeClaim) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		t.Log("creating pvc", log.KObj(pvc))
		err = client.Create(ctx, pvc)
		if err != nil {
			t.Fatal(err)
		}

		t.Log("waiting for pvc to be bound", log.KObj(pvc))
		err = wait.For(
			conditions.New(client).ResourceMatch(pvc, func(obj k8s.Object) bool {
				pvc := obj.(*corev1.PersistentVolumeClaim)
				return pvc.Status.Phase == corev1.ClaimBound
			}),
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}

		err = client.Get(ctx, pvc.GetName(), pvc.GetNamespace(), pvc)
		if err != nil {
			t.Fatal(err)
		}

		t.Log("pvc is bound", log.KObj(pvc), "volumeName", pvc.Spec.VolumeName)
		return ctx
	}
}

// DeletePVC deletes a PVC and waits for it to be fully removed.
func DeletePVC(pvc *corev1.PersistentVolumeClaim) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		t.Log("deleting pvc", log.KObj(pvc))
		err = client.Delete(ctx, pvc)
		if err != nil {
			t.Fatal(err)
		}

		err = wait.For(
			conditions.New(client).ResourceDeleted(pvc),
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}

		t.Log("pvc is deleted", log.KObj(pvc))
		return ctx
	}
}

// WaitForPVBound waits for a PersistentVolume with the given name to reach the Bound phase.
func WaitForPVBound(pvName string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		logger := log.FromContext(ctx)
		pv := &corev1.PersistentVolume{}
		pv.Name = pvName

		t.Log("waiting for pv to be bound", pvName)
		err = wait.For(
			conditions.New(client).ResourceMatch(pv, func(obj k8s.Object) bool {
				v := obj.(*corev1.PersistentVolume)
				if v.Status.Phase != corev1.VolumeBound {
					logger.Info("pv not yet bound",
						"phase", v.Status.Phase,
					)
					return false
				}
				return true
			}),
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}

		t.Log("pv is bound", pvName)
		return ctx
	}
}

// WaitForPVDeleted waits for a PersistentVolume with the given name to be fully removed.
func WaitForPVDeleted(pvName string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		logger := log.FromContext(ctx)
		pv := &corev1.PersistentVolume{}
		pv.Name = pvName

		t.Log("waiting for pv to be deleted", pvName)
		err = wait.For(
			func(ctx context.Context) (bool, error) {
				if err := client.Get(ctx, pvName, "", pv); err != nil {
					if apierrors.IsNotFound(err) {
						return true, nil
					}
					logger.Error("failed to get pv",
						"err", err,
					)
					return false, nil
				}
				logger.Info("pv still exists",
					"phase", pv.Status.Phase,
				)
				return false, nil
			},
			wait.WithContext(ctx),
			wait.WithTimeout(20*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}

		t.Log("pv is deleted", pvName)
		return ctx
	}
}

func waitForServiceAccountReady(ctx context.Context, resource *resources.Resources, name, namespace string) error {
	var sa corev1.ServiceAccount

	logger := log.FromContext(ctx)

	err := wait.For(
		func(ctx context.Context) (done bool, err error) {
			err = resource.Get(ctx, name, namespace, &sa)
			if err == nil {
				return true, nil
			}
			if !apierrors.IsNotFound(err) {
				logger.Error("failed to get service account",
					"err", err,
				)
				return false, nil
			}

			err = resource.Create(ctx, &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			})
			if err == nil {
				return false, nil
			}
			if apierrors.IsAlreadyExists(err) {
				return false, nil
			}

			logger.Error("failed to create service account",
				"err", err,
			)
			return false, nil
		},
		wait.WithContext(ctx),
		wait.WithTimeout(10*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("wait for %s.%s service account ready: %w", name, namespace, err)
	}
	return nil
}

// Environment returns an environment of the test
func Environment() env.Environment {
	logger := log.NewLogger(os.Stderr, log.LevelDebug)
	cfg, err := envconf.NewFromFlags()
	if err != nil {
		logger.Error("failed to create config",
			"err", err,
		)
		os.Exit(1)
	}

	ctx := context.Background()
	ctx = log.NewContext(ctx, logger)

	testEnv, err := env.NewWithContext(ctx, cfg)
	if err != nil {
		logger.Error("failed to create environment",
			"err", err,
		)
		os.Exit(1)
	}

	return testEnv
}
