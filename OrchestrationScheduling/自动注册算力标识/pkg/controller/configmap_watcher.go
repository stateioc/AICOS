package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ConfigMapWatcher struct {
	clientset    *kubernetes.Clientset
	labelWatcher *LabelWatcher
}

func NewConfigMapWatcher(clientset *kubernetes.Clientset, labelWatcher *LabelWatcher) *ConfigMapWatcher {
	return &ConfigMapWatcher{clientset: clientset, labelWatcher: labelWatcher}
}

func (w *ConfigMapWatcher) Run() {
	labelSelector := labels.Set{"cluster_resources_register_kubeconfig": "true"}.AsSelector()
	options := metav1.ListOptions{LabelSelector: labelSelector.String()}

	// 监听带有 "cluster_resources_register_kubeconfig=true" 标签的 ConfigMap
	watcher, err := w.clientset.CoreV1().ConfigMaps("kube-system").Watch(context.Background(), options)
	if err != nil {
		fmt.Printf("watch configmap error %v\n", err)
		panic(err.Error())
	}

	// 处理 ConfigMap 事件
	for event := range watcher.ResultChan() {
		cm, ok := event.Object.(*corev1.ConfigMap)
		if !ok {
			continue
		}

		switch event.Type {
		case watch.Added, watch.Modified:
			// 当 ConfigMap 添加或修改时，启动相应的控制器实例
			w.startControllerForConfigMap(cm)
		case watch.Deleted:
			// 当 ConfigMap 删除时，停止相应的控制器实例（如果需要）
			// 您需要实现相应的逻辑来跟踪和停止控制器实例
		}
	}
}

func (w *ConfigMapWatcher) startControllerForConfigMap(cm *corev1.ConfigMap) {
	// 从 ConfigMap 中获取 kubeconfig 数据
	kubeconfigData := cm.Data["kubeconfig"]
	if kubeconfigData == "" {
		fmt.Printf("Warning: ConfigMap %s does not contain kubeconfig data\n", cm.Name)
		return
	}

	// 使用 kubeconfig 数据创建集群配置
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigData))
	if err != nil {
		fmt.Printf("Error: Failed to parse kubeconfig from ConfigMap %s: %v\n", cm.Name, err)
		return
	}

	// 创建集群的 Kubernetes 客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error: Failed to create Kubernetes client for ConfigMap %s: %v\n", cm.Name, err)
		return
	}

	// 创建并启动控制器
	controller := NewNodeResourceController(clientset)
	go controller.Run() // 使用 goroutine 并发地运行控制器
}
