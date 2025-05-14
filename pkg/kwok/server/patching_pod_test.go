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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestHandlePatchingPod(t *testing.T) {
	tests := []struct {
		name           string
		pod            *corev1.Pod
		contentType    string
		expectedStatus int
		expectedPatch  bool
		notAPod        bool
	}{
		{
			name: "basic pod with no affinity or tolerations",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx",
						},
					},
				},
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedPatch:  true,
		},
		{
			name: "pod with existing node affinity but no required during scheduling",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-with-affinity",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx",
						},
					},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 1,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedPatch:  true,
		},
		{
			name: "pod with existing tolerations",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-with-tolerations",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx",
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "node-role.kubernetes.io/master",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedPatch:  true,
		},
		{
			name: "pod with complete node affinity",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-with-complete-affinity",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx",
						},
					},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedPatch:  true,
		},
		{
			name: "not a pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedPatch:  false,
			notAPod:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{}
			recorder := httptest.NewRecorder()

			podBytes, err := json.Marshal(tc.pod)
			require.NoError(t, err)

			kind := "Pod"
			if tc.notAPod {
				kind = "ConfigMap"
			}

			admissionReviewReq := admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AdmissionReview",
					APIVersion: "admission.k8s.io/v1",
				},
				Request: &admissionv1.AdmissionRequest{
					UID: types.UID("test-uid"),
					Kind: metav1.GroupVersionKind{
						Kind: kind,
					},
					Object: runtime.RawExtension{
						Raw: podBytes,
					},
				},
			}

			reqBytes, err := json.Marshal(admissionReviewReq)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/patch/pod", bytes.NewBuffer(reqBytes))
			req.Header.Set("Content-Type", tc.contentType)
			req = req.WithContext(context.Background())

			server.handlePatchingPod(recorder, req)

			resp := recorder.Result()
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if tc.expectedStatus == http.StatusOK {
				var admissionReviewResp admissionv1.AdmissionReview
				err = json.NewDecoder(resp.Body).Decode(&admissionReviewResp)
				require.NoError(t, err)

				assert.NotNil(t, admissionReviewResp.Response)
				assert.Equal(t, admissionReviewReq.Request.UID, admissionReviewResp.Response.UID)
				assert.True(t, admissionReviewResp.Response.Allowed)

				if tc.expectedPatch && !tc.notAPod {
					assert.NotNil(t, admissionReviewResp.Response.Patch)
					assert.NotNil(t, admissionReviewResp.Response.PatchType)
					assert.Equal(t, admissionv1.PatchTypeJSONPatch, *admissionReviewResp.Response.PatchType)

					var patches []map[string]interface{}
					err = json.Unmarshal(admissionReviewResp.Response.Patch, &patches)
					require.NoError(t, err)
					assert.NotEmpty(t, patches)

					foundAffinity := false
					foundToleration := false

					for _, patch := range patches {
						op, _ := patch["op"].(string)
						path, _ := patch["path"].(string)

						if op == "add" {
							if path == "/spec/affinity" {
								foundAffinity = true
							}
							if path == "/spec/tolerations" || path == "/spec/tolerations/-" {
								foundToleration = true
							}
						}
					}

					if tc.pod.Spec.Affinity == nil {
						assert.True(t, foundAffinity, "Patch should contain node affinity")
					}
					assert.True(t, foundToleration, "Patch should contain tolerations")
				} else if !tc.notAPod {
					// If it's a pod but not expecting a patch
					assert.Empty(t, admissionReviewResp.Response.Patch)
				}
			}
		})
	}
}
