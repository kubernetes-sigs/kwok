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
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

const (
	// kwokStorageClassName is the StorageClass object name selected by the
	// volume stages in kustomize/stage/volume/fast.
	kwokStorageClassName = "kwok-volume"

	// kwokVolumeProvisioner is a provisioner name that has no active
	// controller in the cluster, so only the kwok volume stage will
	// respond to PVCs that request it.
	kwokVolumeProvisioner = "kwok.x-k8s.io/volume"
)

func expectedProvisionedPVName(pvc *corev1.PersistentVolumeClaim) string {
	return fmt.Sprintf("pvc-%s", pvc.UID)
}

func deleteProvisionedPV(ctx context.Context, client *resources.Resources, pvc *corev1.PersistentVolumeClaim) {
	pvName := pvc.Spec.VolumeName
	if pvName == "" && pvc.UID != "" {
		pvName = expectedProvisionedPVName(pvc)
	}
	if pvName == "" {
		return
	}
	_ = client.Delete(ctx, &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: pvName}})
}

// CaseVolumeProvisioner creates a feature that exercises the complete PV/PVC
// lifecycle using the kwok volume stages:
//
//  1. A StorageClass with a kwok-specific provisioner is created.
//  2. A PVC is created → the stage applies a PV and marks the PVC Bound.
//  3. A Pod that mounts the PVC is created → the stage marks it Running.
//  4. The Pod is deleted.
//  5. The PVC is deleted → the stage transitions the PV to Released and
//     removes the PVC finalizer.  A second stage then deletes the PV.
func CaseVolumeProvisioner(nodeName, namespace string) *features.FeatureBuilder {
	scName := kwokStorageClassName
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: scName,
		},
		// Use a provisioner name that no real controller will respond to so
		// that only the kwok stage handles provisioning.
		Provisioner: kwokVolumeProvisioner,
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			m := storagev1.VolumeBindingImmediate
			return &m
		}(),
	}

	node := helper.NewNodeBuilder(nodeName).Build()

	pvc := helper.NewPersistentVolumeClaimBuilder("kwok-pvc").
		WithNamespace(namespace).
		WithStorageClassName(scName).
		WithStorage("2Gi").
		Build()

	var pvName string

	pod := helper.NewPodBuilder("kwok-pod-with-pvc").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		Build()
	// Replace the default containers with a single app container and attach
	// the PVC as a volume so that the pod-create-with-pvc stage picks it up.
	pod.Spec.InitContainers = nil
	pod.Spec.Containers = []corev1.Container{
		{Name: "app", Image: "image"},
	}
	pod.Spec.Volumes = []corev1.Volume{
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		},
	}

	return features.New("VolumeProvisioner: PVC/PV lifecycle with Pod interaction").
		// ── setup ──────────────────────────────────────────────────────────
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// Create the StorageClass (ignore AlreadyExists so the test is
			// idempotent when rerun against the same cluster).
			t.Log("creating storage class", scName)
			if err = client.Create(ctx, sc); err != nil && !apierrors.IsAlreadyExists(err) {
				t.Fatal(err)
			}

			return ctx
		}).
		Setup(helper.CreateNode(node)).
		// ── PVC → PV ───────────────────────────────────────────────────────
		Assess("pvc becomes bound and pv is created", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("creating pvc", log.KObj(pvc))
			if err = client.Create(ctx, pvc); err != nil {
				t.Fatal(err)
			}

			// Wait for PVC to be Bound.
			t.Log("waiting for pvc to be bound", log.KObj(pvc))
			err = wait.For(
				conditions.New(client).ResourceMatch(pvc, func(obj k8s.Object) bool {
					p := obj.(*corev1.PersistentVolumeClaim)
					if p.Status.Phase != corev1.ClaimBound {
						return false
					}
					return true
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pvc did not become Bound:", err)
			}

			if err = client.Get(ctx, pvc.GetName(), pvc.GetNamespace(), pvc); err != nil {
				t.Fatal(err)
			}
			pvName = pvc.Spec.VolumeName
			expectedPVName := expectedProvisionedPVName(pvc)
			if pvc.Spec.VolumeName != expectedPVName {
				t.Errorf("expected pvc.spec.volumeName=%q, got %q", expectedPVName, pvc.Spec.VolumeName)
			}
			t.Log("pvc is bound to pv", pvName)

			// Wait for PV to be created and Bound.
			t.Log("waiting for pv to be bound", pvName)
			pv := &corev1.PersistentVolume{}
			pv.Name = pvName
			err = wait.For(
				conditions.New(client).ResourceMatch(pv, func(obj k8s.Object) bool {
					v := obj.(*corev1.PersistentVolume)
					if v.Status.Phase != corev1.VolumeBound {
						return false
					}
					return true
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pv did not become Bound:", err)
			}
			t.Log("pv is bound", pvName)

			return ctx
		}).
		// ── Pod with PVC → Running ─────────────────────────────────────────
		Assess("pod with pvc becomes running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("creating pod with pvc", log.KObj(pod))
			if err = client.Create(ctx, pod); err != nil {
				t.Fatal(err)
			}

			t.Log("waiting for pod to be running", log.KObj(pod))
			err = wait.For(
				conditions.New(client).PodPhaseMatch(pod, corev1.PodRunning),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pod did not reach Running phase:", err)
			}
			t.Log("pod is running", log.KObj(pod))

			return ctx
		}).
		// ── Pod deletion ───────────────────────────────────────────────────
		Assess("pod deletion", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("deleting pod", log.KObj(pod))
			if err = client.Delete(ctx, pod); err != nil {
				t.Fatal(err)
			}
			err = wait.For(
				conditions.New(client).ResourceDeleted(pod),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pod was not deleted:", err)
			}
			t.Log("pod deleted", log.KObj(pod))

			return ctx
		}).
		// ── PVC deletion → PV Released → PV deleted ────────────────────────
		Assess("pvc deletion triggers pv cleanup", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("deleting pvc", log.KObj(pvc))
			if err = client.Delete(ctx, pvc); err != nil {
				t.Fatal(err)
			}

			// PVC should be garbage-collected after the stage removes its finalizer.
			t.Log("waiting for pvc to be deleted", log.KObj(pvc))
			err = wait.For(
				conditions.New(client).ResourceDeleted(pvc),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pvc was not deleted:", err)
			}
			t.Log("pvc deleted", log.KObj(pvc))

			// The pv-delete stage should remove the PV once it reaches Released.
			t.Log("waiting for pv to be deleted", pvName)
			pv := &corev1.PersistentVolume{}
			pv.Name = pvName
			err = wait.For(
				func(ctx context.Context) (bool, error) {
					if getErr := client.Get(ctx, pvName, "", pv); getErr != nil {
						if apierrors.IsNotFound(getErr) {
							return true, nil
						}
						t.Logf("failed to get pv %s: %v", pvName, getErr)
						return false, nil
					}
					return false, nil
				},
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pv was not deleted:", err)
			}
			t.Log("pv deleted", pvName)

			return ctx
		}).
		// ── teardown ───────────────────────────────────────────────────────
		Teardown(helper.DeleteNode(node)).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			if err = client.Delete(ctx, sc); err != nil && !apierrors.IsNotFound(err) {
				t.Log("warning: failed to delete storage class", scName, err)
			}
			return ctx
		})
}

// CasePVCBeforePod creates a feature for scenario 3: PVC must exist before Pod.
//
// A PVC is created and waits until Bound before the Pod is created.  This
// ensures the Pod scheduler would find the volume ready immediately.  In the
// kwok fast variant the pvc-provision stage binds the PVC synchronously so
// the Pod transitions to Running right away.
func CasePVCBeforePod(nodeName, namespace string) *features.FeatureBuilder {
	pvc := helper.NewPersistentVolumeClaimBuilder("existing-pvc").
		WithNamespace(namespace).
		WithStorageClassName(kwokStorageClassName).
		WithStorage("2Gi").
		Build()

	node := helper.NewNodeBuilder(nodeName).Build()

	pod := helper.NewPodBuilder("pod-use-existing-pvc").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		Build()
	pod.Spec.InitContainers = nil
	pod.Spec.Containers = []corev1.Container{
		{Name: "app", Image: "busybox"},
	}
	pod.Spec.Volumes = []corev1.Volume{
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		},
	}

	return features.New("VolumeProvisioner: PVC created before Pod").
		Setup(helper.CreateNode(node)).
		Assess("pvc bound first, then pod running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("creating pvc", log.KObj(pvc))
			if err = client.Create(ctx, pvc); err != nil {
				t.Fatal(err)
			}

			t.Log("waiting for pvc to be bound before creating pod", log.KObj(pvc))
			err = wait.For(
				conditions.New(client).ResourceMatch(pvc, func(obj k8s.Object) bool {
					p := obj.(*corev1.PersistentVolumeClaim)
					if p.Status.Phase != corev1.ClaimBound {
						return false
					}
					return true
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pvc did not become Bound:", err)
			}
			t.Log("pvc is bound; now creating pod", log.KObj(pvc))

			if err = client.Create(ctx, pod); err != nil {
				t.Fatal(err)
			}
			t.Log("waiting for pod to be running", log.KObj(pod))
			err = wait.For(
				conditions.New(client).PodPhaseMatch(pod, corev1.PodRunning),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pod did not reach Running phase:", err)
			}
			t.Log("pod is running", log.KObj(pod))

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			_ = client.Delete(ctx, pod)
			_ = client.Delete(ctx, pvc)
			deleteProvisionedPV(ctx, client, pvc)
			return ctx
		}).
		Teardown(helper.DeleteNode(node))
}

// CasePodPVCSimultaneous creates a feature for scenario 4: PVC and Pod are
// applied simultaneously (back-to-back Create calls).  The pvc-provision
// stage binds the PVC and the pod-create-with-pvc stage sets the Pod Running
// concurrently.
func CasePodPVCSimultaneous(nodeName, namespace string) *features.FeatureBuilder {
	pvc := helper.NewPersistentVolumeClaimBuilder("combined-pvc").
		WithNamespace(namespace).
		WithStorageClassName(kwokStorageClassName).
		WithStorage("1Gi").
		Build()

	node := helper.NewNodeBuilder(nodeName).Build()

	pod := helper.NewPodBuilder("pod-combined").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		Build()
	pod.Spec.InitContainers = nil
	pod.Spec.Containers = []corev1.Container{
		{Name: "app", Image: "image"},
	}
	pod.Spec.Volumes = []corev1.Volume{
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		},
	}

	return features.New("VolumeProvisioner: PVC and Pod created simultaneously").
		Setup(helper.CreateNode(node)).
		Assess("pvc and pod both reach ready state", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// Create PVC and Pod back-to-back to simulate simultaneous apply.
			t.Log("creating pvc and pod simultaneously")
			if err = client.Create(ctx, pvc); err != nil {
				t.Fatal(err)
			}
			if err = client.Create(ctx, pod); err != nil {
				t.Fatal(err)
			}

			t.Log("waiting for pvc to be bound", log.KObj(pvc))
			err = wait.For(
				conditions.New(client).ResourceMatch(pvc, func(obj k8s.Object) bool {
					p := obj.(*corev1.PersistentVolumeClaim)
					if p.Status.Phase != corev1.ClaimBound {
						return false
					}
					return true
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pvc did not become Bound:", err)
			}
			t.Log("pvc is bound", log.KObj(pvc))

			t.Log("waiting for pod to be running", log.KObj(pod))
			err = wait.For(
				conditions.New(client).PodPhaseMatch(pod, corev1.PodRunning),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pod did not reach Running phase:", err)
			}
			t.Log("pod is running", log.KObj(pod))

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			_ = client.Delete(ctx, pod)
			_ = client.Delete(ctx, pvc)
			deleteProvisionedPV(ctx, client, pvc)
			return ctx
		}).
		Teardown(helper.DeleteNode(node))
}

// CasePodFirstThenPVC creates a feature for scenario 5: Pod is created before
// its PVC exists, then the PVC is created and triggers dynamic PV provisioning.
//
// In the kwok fast variant the pod-create-with-pvc stage fires immediately
// when the Pod is created (it only checks that a PVC volume is declared, not
// that the PVC is already Bound).  The test therefore verifies:
//   - Pod → Running (stage fires before PVC exists)
//   - PVC created → PVC Bound + PV created (pvc-provision fires)
func CasePodFirstThenPVC(nodeName, namespace string) *features.FeatureBuilder {
	pvc := helper.NewPersistentVolumeClaimBuilder("auto-pvc").
		WithNamespace(namespace).
		WithStorageClassName(kwokStorageClassName).
		WithStorage("5Gi").
		Build()

	var pvName string

	node := helper.NewNodeBuilder(nodeName).Build()

	pod := helper.NewPodBuilder("pod-waiting-pvc").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		Build()
	pod.Spec.InitContainers = nil
	pod.Spec.Containers = []corev1.Container{
		{Name: "app", Image: "image"},
	}
	pod.Spec.Volumes = []corev1.Volume{
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		},
	}

	return features.New("VolumeProvisioner: Pod created before PVC").
		Setup(helper.CreateNode(node)).
		// ── Pod first (fast variant: goes Running immediately) ─────────────
		Assess("pod created before pvc becomes running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("creating pod before pvc exists", log.KObj(pod))
			if err = client.Create(ctx, pod); err != nil {
				t.Fatal(err)
			}

			// In the fast variant the stage does not check PVC existence, so
			// the pod goes Running immediately.
			t.Log("waiting for pod to be running", log.KObj(pod))
			err = wait.For(
				conditions.New(client).PodPhaseMatch(pod, corev1.PodRunning),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pod did not reach Running phase:", err)
			}
			t.Log("pod is running before pvc was created", log.KObj(pod))

			return ctx
		}).
		// ── PVC created after Pod → dynamic PV provisioned ─────────────────
		Assess("pvc created after pod triggers dynamic pv provisioning", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("creating pvc after pod is already running", log.KObj(pvc))
			if err = client.Create(ctx, pvc); err != nil {
				t.Fatal(err)
			}

			t.Log("waiting for pvc to be bound", log.KObj(pvc))
			err = wait.For(
				conditions.New(client).ResourceMatch(pvc, func(obj k8s.Object) bool {
					p := obj.(*corev1.PersistentVolumeClaim)
					if p.Status.Phase != corev1.ClaimBound {
						return false
					}
					return true
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pvc did not become Bound:", err)
			}
			if err = client.Get(ctx, pvc.GetName(), pvc.GetNamespace(), pvc); err != nil {
				t.Fatal(err)
			}
			pvName = pvc.Spec.VolumeName
			expectedPVName := expectedProvisionedPVName(pvc)
			if pvName != expectedPVName {
				t.Errorf("expected pvc.spec.volumeName=%q, got %q", expectedPVName, pvName)
			}
			t.Log("pvc is bound and pv was provisioned", pvName)

			// Verify the PV was created and is Bound.
			pv := &corev1.PersistentVolume{}
			pv.Name = pvName
			err = wait.For(
				conditions.New(client).ResourceMatch(pv, func(obj k8s.Object) bool {
					v := obj.(*corev1.PersistentVolume)
					if v.Status.Phase != corev1.VolumeBound {
						return false
					}
					return true
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("pv did not become Bound:", err)
			}
			t.Log("pv is bound", pvName)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			_ = client.Delete(ctx, pod)
			_ = client.Delete(ctx, pvc)
			deleteProvisionedPV(ctx, client, pvc)
			return ctx
		}).
		Teardown(helper.DeleteNode(node))
}
