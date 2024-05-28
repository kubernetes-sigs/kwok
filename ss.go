package main

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	if err := testNodeReady("fake-node"); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func testNodeReady(nodeName string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	for i := 0; i < 300; i++ {
		node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get node: %w", err)
		}

		if isNodeReady(node) {
			fmt.Println("Node is ready")
			return nil
		}

		fmt.Println("Waiting for node to be ready...")
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("node %s is not ready after 300 seconds", nodeName)
}

func isNodeReady(node *v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}
