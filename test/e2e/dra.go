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
	resourcev1 "k8s.io/api/resource/v1"
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
	// kwokDRADriver is the GPU driver name used by the DRA stages in
	// kustomize/stage/dra/gpu.
	kwokDRADriver = "gpu.kwok.x-k8s.io"

	// kwokDRADeviceClass is the DeviceClass object name created by
	// kustomize/stage/dra/gpu, it selects devices of kwokDRADriver.
	kwokDRADeviceClass = "gpu.kwok.x-k8s.io"

	// kwokDRAGPUAnnotation is the node annotation that opts a node into the
	// gpu-resource-slice-publish stage, its value is the number of fake GPUs.
	kwokDRAGPUAnnotation = "kwok.x-k8s.io/dra-gpu"

	// kwokDRACPUDriver is the simulated dra-driver-cpu driver name used by
	// the cpu-resource-slice-publish stage in kustomize/stage/dra/cpu.
	kwokDRACPUDriver = "dra.cpu"

	// kwokDRACPUAnnotation is the node annotation that opts a node into the
	// cpu-resource-slice-publish stage, its value is the number of fake CPUs.
	kwokDRACPUAnnotation = "kwok.x-k8s.io/dra-cpu"

	// kwokDRACPUNUMAAnnotation is the node annotation that sets the number of
	// NUMA nodes the fake CPUs are spread across.
	kwokDRACPUNUMAAnnotation = "kwok.x-k8s.io/dra-cpu-numa"

	// kwokDRANvidiaGPUDriver is the simulated dra-driver-nvidia-gpu driver name
	// used by the nvidia-gpu-resource-slice-publish stage in
	// kustomize/stage/dra/nvidia-gpu.
	kwokDRANvidiaGPUDriver = "gpu.nvidia.com"

	// kwokDRANvidiaGPUAnnotation is the node annotation that opts a node into
	// the nvidia-gpu-resource-slice-publish stage, its value is the number of
	// fake GPUs.
	kwokDRANvidiaGPUAnnotation = "kwok.x-k8s.io/dra-nvidia-gpu"

	// kwokDRAGoogleTPUDriver is the simulated dra-driver-google-tpu driver name
	// used by the google-tpu-resource-slice-publish stage in
	// kustomize/stage/dra/google-tpu.
	kwokDRAGoogleTPUDriver = "tpu.google.com"

	// kwokDRAGoogleTPUAnnotation is the node annotation that opts a node into
	// the google-tpu-resource-slice-publish stage, its value is the number of
	// fake TPU chips.
	kwokDRAGoogleTPUAnnotation = "kwok.x-k8s.io/dra-google-tpu"
)

// CaseDRA creates a feature that exercises the DRA simulation provided by the
// kwok DRA stages:
//
//  1. A node annotated with fake GPU and CPU counts is created → the stages
//     publish ResourceSlices for the node.
//  2. A ResourceClaim requesting a device of the kwok DeviceClass and a Pod
//     referencing it are created → the kube-scheduler allocates the claim and
//     the pod becomes Running.
//  3. The Pod is deleted → the claim reservation is released.
//  4. The Node is deleted → the ResourceSlices are garbage collected.
func CaseDRA(nodeName, namespace string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder(nodeName).
		WithAnnotation(kwokDRAGPUAnnotation, "2").
		WithAnnotation(kwokDRACPUAnnotation, "4").
		WithAnnotation(kwokDRACPUNUMAAnnotation, "2").
		WithAnnotation(kwokDRANvidiaGPUAnnotation, "1").
		WithAnnotation(kwokDRAGoogleTPUAnnotation, "4").
		Build()

	sliceName := nodeName + "-" + kwokDRADriver
	cpuSliceName := nodeName + "-" + kwokDRACPUDriver
	nvidiaSliceName := nodeName + "-" + kwokDRANvidiaGPUDriver
	tpuSliceName := nodeName + "-" + kwokDRAGoogleTPUDriver

	claim := &resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kwok-gpu-claim",
			Namespace: namespace,
		},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{
					{
						Name: "gpu",
						Exactly: &resourcev1.ExactDeviceRequest{
							DeviceClassName: kwokDRADeviceClass,
						},
					},
				},
			},
		},
	}

	pod := helper.NewPodBuilder("kwok-pod-with-claim").
		WithNamespace(namespace).
		Build()
	// Replace the default containers with a single app container referencing
	// the ResourceClaim, and let the kube-scheduler schedule it onto the fake
	// node so that the DRA plugin allocates the claim.
	pod.Spec.InitContainers = nil
	pod.Spec.Containers = []corev1.Container{
		{
			Name:  "app",
			Image: "image",
			Resources: corev1.ResourceRequirements{
				Claims: []corev1.ResourceClaim{
					{Name: "gpu"},
				},
			},
		},
	}
	pod.Spec.NodeSelector = map[string]string{
		"type": "kwok",
	}
	pod.Spec.ResourceClaims = []corev1.PodResourceClaim{
		{
			Name:              "gpu",
			ResourceClaimName: &claim.Name,
		},
	}

	return features.New("DRA: ResourceSlice publishing and ResourceClaim allocation").
		// ── setup ──────────────────────────────────────────────────────────
		Setup(helper.CreateNode(node)).
		// ── Node → ResourceSlice ───────────────────────────────────────────
		Assess("resource slice is published for the node", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("waiting for resource slice to be published", sliceName)
			slice := &resourcev1.ResourceSlice{}
			slice.Name = sliceName
			err = wait.For(
				conditions.New(client).ResourceMatch(slice, func(obj k8s.Object) bool {
					s := obj.(*resourcev1.ResourceSlice)
					if s.Spec.Driver != kwokDRADriver {
						return false
					}
					if s.Spec.NodeName == nil || *s.Spec.NodeName != nodeName {
						return false
					}
					return len(s.Spec.Devices) == 2
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("resource slice was not published:", err)
			}
			t.Log("resource slice is published", sliceName)

			t.Log("waiting for cpu resource slice to be published", cpuSliceName)
			cpuSlice := &resourcev1.ResourceSlice{}
			cpuSlice.Name = cpuSliceName
			err = wait.For(
				conditions.New(client).ResourceMatch(cpuSlice, func(obj k8s.Object) bool {
					s := obj.(*resourcev1.ResourceSlice)
					if s.Spec.Driver != kwokDRACPUDriver {
						return false
					}
					// 4 CPUs across 2 NUMA nodes in the default grouped mode:
					// one device per NUMA node with 2 consumable CPUs each.
					if len(s.Spec.Devices) != 2 {
						return false
					}
					d := s.Spec.Devices[1]
					numa, ok := d.Attributes["dra.cpu/numaNodeID"]
					if !ok || numa.IntValue == nil || *numa.IntValue != 1 {
						return false
					}
					capacity, ok := d.Capacity["dra.cpu/cpu"]
					return ok && capacity.Value.CmpInt64(2) == 0
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("cpu resource slice was not published:", err)
			}
			t.Log("cpu resource slice is published", cpuSliceName)

			t.Log("waiting for nvidia gpu resource slice to be published", nvidiaSliceName)
			nvidiaSlice := &resourcev1.ResourceSlice{}
			nvidiaSlice.Name = nvidiaSliceName
			err = wait.For(
				conditions.New(client).ResourceMatch(nvidiaSlice, func(obj k8s.Object) bool {
					s := obj.(*resourcev1.ResourceSlice)
					if s.Spec.Driver != kwokDRANvidiaGPUDriver {
						return false
					}
					if len(s.Spec.Devices) != 1 {
						return false
					}
					typ, ok := s.Spec.Devices[0].Attributes["type"]
					return ok && typ.StringValue != nil && *typ.StringValue == "gpu"
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("nvidia gpu resource slice was not published:", err)
			}
			t.Log("nvidia gpu resource slice is published", nvidiaSliceName)

			t.Log("waiting for google tpu resource slice to be published", tpuSliceName)
			tpuSlice := &resourcev1.ResourceSlice{}
			tpuSlice.Name = tpuSliceName
			err = wait.For(
				conditions.New(client).ResourceMatch(tpuSlice, func(obj k8s.Object) bool {
					s := obj.(*resourcev1.ResourceSlice)
					if s.Spec.Driver != kwokDRAGoogleTPUDriver {
						return false
					}
					if len(s.Spec.Devices) != 4 {
						return false
					}
					gen, ok := s.Spec.Devices[0].Attributes["tpuGen"]
					return ok && gen.StringValue != nil && *gen.StringValue == "v4"
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("google tpu resource slice was not published:", err)
			}
			t.Log("google tpu resource slice is published", tpuSliceName)

			return ctx
		}).
		// ── Pod with ResourceClaim → Running ───────────────────────────────
		Assess("pod with resource claim becomes running and claim is allocated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("creating resource claim", log.KObj(claim))
			if err = client.Create(ctx, claim); err != nil {
				t.Fatal(err)
			}

			t.Log("creating pod with resource claim", log.KObj(pod))
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

			if err = client.Get(ctx, claim.Name, claim.Namespace, claim); err != nil {
				t.Fatal(err)
			}
			if claim.Status.Allocation == nil {
				t.Fatal("resource claim is not allocated")
			}
			results := claim.Status.Allocation.Devices.Results
			if len(results) != 1 || results[0].Driver != kwokDRADriver {
				t.Fatalf("unexpected allocation results: %+v", results)
			}
			if len(claim.Status.ReservedFor) != 1 || claim.Status.ReservedFor[0].UID != pod.UID {
				t.Fatalf("resource claim is not reserved for the pod: %+v", claim.Status.ReservedFor)
			}
			t.Log("resource claim is allocated and reserved", log.KObj(claim))

			return ctx
		}).
		// ── Pod deletion → claim released ──────────────────────────────────
		Assess("pod deletion releases the claim reservation", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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

			t.Log("waiting for claim reservation to be released", log.KObj(claim))
			err = wait.For(
				conditions.New(client).ResourceMatch(claim, func(obj k8s.Object) bool {
					c := obj.(*resourcev1.ResourceClaim)
					return len(c.Status.ReservedFor) == 0
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("claim reservation was not released:", err)
			}
			t.Log("claim reservation released", log.KObj(claim))

			t.Log("deleting resource claim", log.KObj(claim))
			if err = client.Delete(ctx, claim); err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		// ── Node deletion → ResourceSlice garbage collected ────────────────
		Assess("node deletion garbage collects the resource slice", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("deleting node", nodeName)
			if err = client.Delete(ctx, node); err != nil {
				t.Fatal(err)
			}
			err = wait.For(
				conditions.New(client).ResourceDeleted(node),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("node was not deleted:", err)
			}

			t.Log("waiting for resource slice to be garbage collected", sliceName)
			slice := &resourcev1.ResourceSlice{}
			slice.Name = sliceName
			err = wait.For(
				conditions.New(client).ResourceDeleted(slice),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("resource slice was not garbage collected:", err)
			}
			t.Log("resource slice garbage collected", sliceName)

			t.Log("waiting for cpu resource slice to be garbage collected", cpuSliceName)
			cpuSlice := &resourcev1.ResourceSlice{}
			cpuSlice.Name = cpuSliceName
			err = wait.For(
				conditions.New(client).ResourceDeleted(cpuSlice),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("cpu resource slice was not garbage collected:", err)
			}
			t.Log("cpu resource slice garbage collected", cpuSliceName)

			t.Log("waiting for nvidia gpu resource slice to be garbage collected", nvidiaSliceName)
			nvidiaSlice := &resourcev1.ResourceSlice{}
			nvidiaSlice.Name = nvidiaSliceName
			err = wait.For(
				conditions.New(client).ResourceDeleted(nvidiaSlice),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("nvidia gpu resource slice was not garbage collected:", err)
			}
			t.Log("nvidia gpu resource slice garbage collected", nvidiaSliceName)

			t.Log("waiting for google tpu resource slice to be garbage collected", tpuSliceName)
			tpuSlice := &resourcev1.ResourceSlice{}
			tpuSlice.Name = tpuSliceName
			err = wait.For(
				conditions.New(client).ResourceDeleted(tpuSlice),
				wait.WithContext(ctx),
				wait.WithTimeout(3*time.Minute),
			)
			if err != nil {
				t.Fatal("google tpu resource slice was not garbage collected:", err)
			}
			t.Log("google tpu resource slice garbage collected", tpuSliceName)

			return ctx
		}).
		// ── teardown ───────────────────────────────────────────────────────
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			// Best-effort cleanup in case an assessment failed midway.
			if err = client.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
				t.Log("warning: failed to delete pod", log.KObj(pod), err)
			}
			if err = client.Delete(ctx, claim); err != nil && !apierrors.IsNotFound(err) {
				t.Log("warning: failed to delete resource claim", log.KObj(claim), err)
			}
			if err = client.Delete(ctx, node); err != nil && !apierrors.IsNotFound(err) {
				t.Log("warning: failed to delete node", nodeName, err)
			}
			return ctx
		})
}
