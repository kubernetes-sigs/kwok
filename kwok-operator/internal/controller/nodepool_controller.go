/*
Copyright 2024.

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

package controller

import (
	"context"
	kwoksigsv1beta1 "github.com/lliranbabi/kwok/kwok-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

const (
	nodePoolFinalizer       = "kwok.sigs.k8s.io/finalizer"
	nodePoolControllerLabel = "kwok.x-k8s.io/controller"
	nodePoolAnnotation      = "kwok.x-k8s.io/node"
	fakeString              = "fake"
)

// NodePoolReconciler reconciles a NodePool object
type NodePoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kwok.sigs.k8s.io,resources=nodepools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.sigs.k8s.io,resources=nodepools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kwok.sigs.k8s.io,resources=nodepools/finalizers,verbs=update

func (r *NodePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling NodePool")
	nodePool := &kwoksigsv1beta1.NodePool{}
	err := r.Get(ctx, req.NamespacedName, nodePool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("nodePool resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request CR.
		log.Error(err, "Failed to get NodePool")
		return ctrl.Result{}, err
	}
	// Set reconciling status condition in the NodePool
	if nodePool.Status.Conditions == nil || len(nodePool.Status.Conditions) == 0 {
		err := r.statusConditionController(ctx, nodePool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting to reconcile the NodePool",
		})
		if err != nil {
			log.Error(err, "Failed to update NodePool status")
			return ctrl.Result{}, err
		}
		err = r.Get(ctx, req.NamespacedName, nodePool)
		if err != nil {
			log.Error(err, "Failed to get NodePool")
			return ctrl.Result{}, err
		}
	}
	// Add finalizer to the NodePool
	if !controllerutil.ContainsFinalizer(nodePool, nodePoolFinalizer) {
		log.Info("Adding Finalizer for the NodePool")
		err := r.addFinalizer(ctx, nodePool)
		if err != nil {
			log.Error(err, "Failed to add finalizer to NodePool")
			return ctrl.Result{}, err
		}
	}
	// Get nodes in the cluster with owner reference to the nodePool
	nodes, err := r.getNodes(ctx, nodePool)
	if err != nil {
		log.Error(err, "Failed to get nodes")
		return ctrl.Result{}, err
	}
	// Check if the number of nodes in the cluster is equal to the desired number of nodes
	if int32(len(nodes)) != nodePool.Spec.NodeCount {
		// Create or delete nodes in the cluster
		if int32(len(nodes)) < nodePool.Spec.NodeCount {
			err := r.statusConditionController(ctx, nodePool, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionFalse,
				Reason:  "ScalingUp",
				Message: "Scalling up the NodePool",
			})
			if err != nil {
				log.Error(err, "Failed to update NodePool status")
				return ctrl.Result{}, err
			}
			log.Info("Scalling up the NodePool... creating nodes!")
			err = r.createNodes(ctx, nodePool, nodes)
			if err != nil {
				log.Error(err, "Failed to create nodes")
				return ctrl.Result{}, err
			}
		} else {
			err := r.statusConditionController(ctx, nodePool, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionFalse,
				Reason:  "ScalingDown",
				Message: "Scalling down the NodePool",
			})
			if err != nil {
				log.Error(err, "Failed to update NodePool status")
				return ctrl.Result{}, err
			}
			log.Info("Too many nodes... deleting! ")
			err = r.deleteNodes(ctx, nodePool, nodes)
			if err != nil {
				log.Error(err, "Failed to delete nodes")
				return ctrl.Result{}, err
			}
		}
		err := r.statusConditionController(ctx, nodePool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "NodePool is ready",
		})
		if err != nil {
			log.Error(err, "Failed to update NodePool status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if observed generation is different from the generation
	if nodePool.Status.ObservedGeneration != nodePool.Generation {
		log.Info("NodeTemplate has changed")
		err := r.statusConditionController(ctx, nodePool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Updating",
			Message: "Updating the NodePool",
		})
		if err != nil {
			log.Error(err, "Failed to update NodePool status")
			return ctrl.Result{}, err
		}
		emptyNodePool := &kwoksigsv1beta1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodePool.Name,
			},
		}
		err = r.deleteNodes(ctx, emptyNodePool, nodes)
		if err != nil {
			log.Error(err, "Failed to delete nodes")
			return ctrl.Result{}, err
		}
		err = r.createNodes(ctx, nodePool, nodes)
		if err != nil {
			log.Error(err, "Failed to create nodes")
			return ctrl.Result{}, err
		}
		err = r.statusConditionController(ctx, nodePool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "NodePool is ready",
		})
		if err != nil {
			log.Error(err, "Failed to update NodePool status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if !nodePool.DeletionTimestamp.IsZero() {
		// Remove finalizer from the NodePool
		log.Info("Deleting the NodePool")
		err = r.statusConditionController(ctx, nodePool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Deleting",
			Message: "Deleting the NodePool",
		})
		if err != nil {
			log.Error(err, "Failed to update NodePool status")
			return ctrl.Result{}, err
		}
		err := r.deleteNodes(ctx, nodePool, nodes)
		if err != nil {
			log.Error(err, "Failed to delete nodes")
			return ctrl.Result{}, err
		}
		err = r.deleteFinalizer(ctx, nodePool)
		if err != nil {
			log.Error(err, "Failed to delete finalizer from NodePool")
			return ctrl.Result{}, err
		}
		err = r.statusConditionController(ctx, nodePool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Deleting",
			Message: "Deleting the NodePool",
		})
		if err != nil {
			log.Error(err, "Failed to update NodePool status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	log.Info("Reconciliation completed successfully")
	return ctrl.Result{RequeueAfter: time.Duration(60 * time.Second)}, nil
}

func (r *NodePoolReconciler) getNodes(ctx context.Context, nodePool *kwoksigsv1beta1.NodePool) ([]corev1.Node, error) {
	nodes := &corev1.NodeList{}
	err := r.List(ctx, nodes, client.InNamespace(nodePool.Namespace), client.MatchingLabels{nodePoolControllerLabel: nodePool.Name})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		return []corev1.Node{}, nil
	} else if err != nil {
		return nil, err
	}
	return nodes.Items, nil

}

// Create nodes in the cluster
func (r *NodePoolReconciler) createNodes(ctx context.Context, nodePool *kwoksigsv1beta1.NodePool, nodes []corev1.Node) error {
	nodeLabels := nodePool.Spec.NodeTemplate.Labels
	if nodeLabels == nil {
		nodeLabels = make(map[string]string)
	}
	nodeLabels[nodePoolControllerLabel] = nodePool.Name
	nodeTaint := nodePool.Spec.NodeTemplate.Spec.Taints
	if nodeTaint == nil {
		nodeTaint = make([]corev1.Taint, 0)
	}
	nodeTaint = append(nodeTaint, corev1.Taint{
		Key:    nodePoolAnnotation,
		Value:  fakeString,
		Effect: corev1.TaintEffectNoSchedule,
	})
	nodeAnnotation := nodePool.Spec.NodeTemplate.Annotations
	if nodeAnnotation == nil {
		nodeAnnotation = make(map[string]string)
	}
	nodeAnnotation[nodePoolAnnotation] = fakeString
	for i := int32(len(nodes)); i < nodePool.Spec.NodeCount; i++ {
		// Create a new node
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: nodePool.Name + "-",
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(nodePool, kwoksigsv1beta1.GroupVersion.WithKind("NodePool")),
				},
			},
			Spec: nodePool.Spec.NodeTemplate.Spec,
		}
		node.Labels = nodeLabels
		node.Spec.Taints = nodeTaint
		node.ObjectMeta.Annotations = nodeAnnotation

		err := r.Create(ctx, node)
		if err != nil {
			return err
		}
	}

	err := r.updateObservedGeneration(ctx, nodePool)
	if err != nil {
		return err
	}
	return nil
}

// Delete nodes in the cluster
func (r *NodePoolReconciler) deleteNodes(ctx context.Context, nodePool *kwoksigsv1beta1.NodePool, nodes []corev1.Node) error {
	for i := int32(len(nodes)); i > nodePool.Spec.NodeCount; i-- {
		// Delete a node
		err := r.Delete(ctx, &nodes[i-1])
		if err != nil {
			return err
		}
	}
	err := r.updateObservedGeneration(ctx, nodePool)
	if err != nil {
		return err
	}
	return nil
}

// Add the finalizer to the NodePool
func (r *NodePoolReconciler) addFinalizer(ctx context.Context, nodePool *kwoksigsv1beta1.NodePool) error {
	controllerutil.AddFinalizer(nodePool, nodePoolFinalizer)
	return r.Update(ctx, nodePool)
}

// Delete the finalizer from the NodePool
func (r *NodePoolReconciler) deleteFinalizer(ctx context.Context, nodePool *kwoksigsv1beta1.NodePool) error {
	controllerutil.RemoveFinalizer(nodePool, nodePoolFinalizer)
	return r.Update(ctx, nodePool)
}

// statusConditionController updates the status of the NodePool
func (r *NodePoolReconciler) statusConditionController(ctx context.Context, nodePool *kwoksigsv1beta1.NodePool, condition metav1.Condition) error {
	meta.SetStatusCondition(&nodePool.Status.Conditions, condition)
	return r.Status().Update(ctx, nodePool)
}

// updateObservedGeneration updates the observed generation of the NodePool
func (r *NodePoolReconciler) updateObservedGeneration(ctx context.Context, nodePool *kwoksigsv1beta1.NodePool) error {
	nodePool.Status.ObservedGeneration = nodePool.Generation
	return r.Status().Update(ctx, nodePool)
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kwoksigsv1beta1.NodePool{}).
		Complete(r)
}
