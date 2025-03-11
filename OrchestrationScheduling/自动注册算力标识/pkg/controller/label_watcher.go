package controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"register-power-resources/pkg/apis"
)

type LabelWatcher struct {
	clientset  *kubernetes.Clientset
	labelInfo  apis.LabelInfo
	labelMutex sync.RWMutex
}

func NewLabelWatcher(clientset *kubernetes.Clientset) *LabelWatcher {
	return &LabelWatcher{clientset: clientset}
}

func (w *LabelWatcher) Run() {
	labelSelector := labels.Set{"label-config": "true"}.AsSelector()
	options := metav1.ListOptions{LabelSelector: labelSelector.String()}

	// 监听带有 "label-config=true" 标签的 ConfigMap
	watcher, err := w.clientset.CoreV1().ConfigMaps("").Watch(context.Background(), options)
	if err != nil {
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
			// 当 ConfigMap 添加或修改时，更新标签信息
			w.updateLabelInfo(cm)
		case watch.Deleted:
			// 当 ConfigMap 删除时，清除标签信息
			w.labelMutex.Lock()
			w.labelInfo = apis.LabelInfo{}
			w.labelMutex.Unlock()
		}
	}
}

func (w *LabelWatcher) GetLabelInfo() apis.LabelInfo {
	w.labelMutex.RLock()
	defer w.labelMutex.RUnlock()
	return w.labelInfo
}

func (w *LabelWatcher) updateLabelInfo(cm *corev1.ConfigMap) {
	w.labelMutex.Lock()
	defer w.labelMutex.Unlock()

	w.labelInfo.CityName = cm.Data["cityName"]
	w.labelInfo.IndustryName = cm.Data["industryName"]
	w.labelInfo.EnterpriseName = cm.Data["enterpriseName"]
	w.labelInfo.ResourceType = cm.Data["resourceType"]
	w.labelInfo.ServiceType = cm.Data["serviceType"]
}
