package main

import (
	"fmt"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"register-power-resources/pkg/node-reporter"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("Error getting in cluster config: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error getting clientset: %v\n", err)
		os.Exit(1)
	}

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		fmt.Println("NODE_NAME environment variable is not set")
		os.Exit(1)
	}

	reporter := node_reporter.NewNodeReporter(clientset, nodeName)

	for {
		if err := reporter.UpdateNodeLabels(); err != nil {
			fmt.Printf("Error updating node labels: %v\n", err)
			continue
		}

		fmt.Println("Successfully added labels to the node")

		// 每隔1分钟更新一次节点标签
		time.Sleep(1 * time.Minute)
	}
}
