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
	"encoding/json"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"sigs.k8s.io/kwok/pkg/log"
)

var (
    universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

const (
	KwokNodeLabel = "kwok.x-k8s.io/node"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (s *Server) handlePatchingPod(rw http.ResponseWriter, req *http.Request) {
	logger := log.FromContext(req.Context())
    body, err := io.ReadAll(req.Body)
    if err != nil {
        logger.Error("Failed to read request body", err)
        http.Error(rw, err.Error(), http.StatusBadRequest)
        return
    }

    var admissionReviewReq admissionv1.AdmissionReview
    _, _, err = universalDeserializer.Decode(body, nil, &admissionReviewReq)
    if err != nil {
        logger.Error("Failed to decode admission review request", err)
        http.Error(rw, err.Error(), http.StatusBadRequest)
        return
    }

    pod := corev1.Pod{}
    if err := json.Unmarshal(admissionReviewReq.Request.Object.Raw, &pod); err != nil {
        logger.Error("Failed to unmarshal pod from request", err)
        http.Error(rw, err.Error(), http.StatusBadRequest)
        return
    }

    patches := []map[string]interface{}{}
    nodeSelectorTerm := corev1.NodeSelectorTerm{
        MatchExpressions: []corev1.NodeSelectorRequirement{{
            Key:      KwokNodeLabel,
            Operator: corev1.NodeSelectorOpExists,
        }},
    }

    affinityPatch := map[string]interface{}{
        "op":   "add",
        "path": "/spec/affinity",
        "value": corev1.Affinity{
            NodeAffinity: &corev1.NodeAffinity{
                RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
                    NodeSelectorTerms: []corev1.NodeSelectorTerm{nodeSelectorTerm},
                },
            },
        },
    }

    if pod.Spec.Affinity == nil {
        patches = append(patches, affinityPatch)
    }

    toleration := corev1.Toleration{
        Key:      KwokNodeLabel,
        Operator: corev1.TolerationOpExists,
        Effect:   corev1.TaintEffectNoSchedule,
    }

    if len(pod.Spec.Tolerations) == 0 {
        patches = append(patches, map[string]interface{}{
            "op":    "add",
            "path":  "/spec/tolerations",
            "value": []corev1.Toleration{toleration},
        })
    } else {
        patches = append(patches, map[string]interface{}{
            "op":    "add",
            "path":  "/spec/tolerations/-",
            "value": toleration,
        })
    }

    patchBytes, err := json.Marshal(patches)
    if err != nil {
        logger.Error("Failed to marshal patches", err)
        http.Error(rw, err.Error(), http.StatusInternalServerError)
        return
    }

    var admissionReviewResp admissionv1.AdmissionReview
    admissionReviewResp.Response = &admissionv1.AdmissionResponse{
        UID:     admissionReviewReq.Request.UID,
        Allowed: true,
        Patch:   patchBytes,
        PatchType: func() *admissionv1.PatchType {
            pt := admissionv1.PatchTypeJSONPatch
            return &pt
        }(),
    }

    admissionReviewResp.APIVersion = "admission.k8s.io/v1"
    admissionReviewResp.Kind = "AdmissionReview"

    rw.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(rw).Encode(admissionReviewResp); err != nil {
        logger.Error("Failed to encode admission response", err)
        http.Error(rw, err.Error(), http.StatusInternalServerError)
    }
}

func (s *Server) InstallPatchingPodHandler() {
	s.restfulCont.Handle("/patch/pod", http.HandlerFunc(s.handlePatchingPod))
}