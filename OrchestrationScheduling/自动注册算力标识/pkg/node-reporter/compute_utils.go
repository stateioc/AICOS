package node_reporter

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getAllocatableCPU(node *corev1.Node) *resource.Quantity {
	cpu := node.Status.Allocatable[corev1.ResourceCPU]
	return &cpu
}

func getAllocatableMemory(node *corev1.Node) *resource.Quantity {
	memory := node.Status.Allocatable[corev1.ResourceMemory]
	return &memory
}

func getAllocatableResources(clientset *kubernetes.Clientset, node *corev1.Node) (availableCPU float64, availableMemory float64, err error) {
	allocatableCPU := getAllocatableCPU(node)
	allocatableMemory := getAllocatableMemory(node)

	allocatableCPUQuantity := allocatableCPU.DeepCopy()
	allocatableMemoryQuantity := allocatableMemory.DeepCopy()

	return float64(allocatableCPUQuantity.MilliValue()) / 1000, float64(allocatableMemoryQuantity.Value()) / (1024 * 1024 * 1024), nil
}

func getAvailableResources(clientset *kubernetes.Clientset, node *corev1.Node) (availableCPU float64, availableMemory float64, err error) {
	allocatableCPU := getAllocatableCPU(node)
	allocatableMemory := getAllocatableMemory(node)

	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("error listing pods: %v", err)
	}

	usedCPU := resource.NewQuantity(0, resource.DecimalSI)
	usedMemory := resource.NewQuantity(0, resource.BinarySI)

	for _, pod := range pods.Items {
		if pod.Spec.NodeName != node.Name {
			continue
		}

		for _, container := range pod.Spec.Containers {
			usedCPU.Add(container.Resources.Requests[corev1.ResourceCPU])
			usedMemory.Add(container.Resources.Requests[corev1.ResourceMemory])
		}
	}

	availableCPUQuantity := allocatableCPU.DeepCopy()
	availableCPUQuantity.Sub(*usedCPU)
	availableCPU = float64(availableCPUQuantity.MilliValue()) / 1000

	availableMemoryQuantity := allocatableMemory.DeepCopy()
	availableMemoryQuantity.Sub(*usedMemory)
	availableMemory = float64(availableMemoryQuantity.Value()) / (1024 * 1024 * 1024)

	return availableCPU, availableMemory, nil
}
