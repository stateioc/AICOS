/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	clientset "cncos.io/box-controller/pkg/box-generated/clientset/versioned"
	boxscheme "cncos.io/box-controller/pkg/box-generated/clientset/versioned/scheme"
	informers "cncos.io/box-controller/pkg/box-generated/informers/externalversions/box/v1alpha1"
	listers "cncos.io/box-controller/pkg/box-generated/listers/box/v1alpha1"
	boxdeploymentclientset "cncos.io/box-controller/pkg/boxdeployment-generated/clientset/versioned"
	boxdeploymentinformers "cncos.io/box-controller/pkg/boxdeployment-generated/informers/externalversions/boxdeployment/v1alpha1"
	boxdeploymentlisters "cncos.io/box-controller/pkg/boxdeployment-generated/listers/boxdeployment/v1alpha1"
)

const controllerAgentName = "cncos-box-controller"

const (
	PodKind           = "Pod"
	DeploymentKind    = "Deployment"
	BoxKind           = "Box"
	BoxDeploymentKind = "BoxDeployment"
)

const (
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Box fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Box"
	// MessageResourceSynced is the message used for an Event fired when a Box
	// is synced successfully
	MessageResourceSynced = "Resource synced successfully"
)

type UpdateStatusItem struct {
	Key    string
	OldObj interface{}
	NewObj interface{}
}

// Controller is the controller implementation for Box resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// boxclientset is a clientset for our own API group
	boxclientset       clientset.Interface
	boxdeployclientset boxdeploymentclientset.Interface

	// deploymentsLister appslisters.DeploymentLister
	// deploymentsSynced cache.InformerSynced
	boxesLister listers.BoxLister
	boxesSynced cache.InformerSynced

	boxDeploymentsLister boxdeploymentlisters.BoxDeploymentLister
	boxDeploymentsSynced cache.InformerSynced
	// pod
	podLister v1.PodLister
	podSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	boxclientset clientset.Interface,
	boxdeployclientset boxdeploymentclientset.Interface,
	podInformer corev1informers.PodInformer,
	// deploymentInformer appsinformers.DeploymentInformer,
	boxdeployInformer boxdeploymentinformers.BoxDeploymentInformer,
	boxInformer informers.BoxInformer) *Controller {
	logger := klog.FromContext(ctx)

	utilruntime.Must(boxscheme.AddToScheme(scheme.Scheme))
	logger.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		// deploymentsLister:    deploymentInformer.Lister(),
		// deploymentsSynced:    deploymentInformer.Informer().HasSynced,
		kubeclientset:        kubeclientset,
		boxclientset:         boxclientset,
		boxdeployclientset:   boxdeployclientset,
		podLister:            podInformer.Lister(),
		podSynced:            podInformer.Informer().HasSynced,
		boxesLister:          boxInformer.Lister(),
		boxesSynced:          boxInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Boxes"),
		recorder:             recorder,
		boxDeploymentsLister: boxdeployInformer.Lister(),
		boxDeploymentsSynced: boxdeployInformer.Informer().HasSynced,
	}

	logger.Info("Setting up event handlers")

	// Set up an event handler for when Box resources change
	boxInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			newDepl := new.(*boxv1alpha1.Box)
			klog.V(4).Infof("add box: %v", newDepl.Namespace+"/"+newDepl.Name)
			controller.enqueueBox(new, BoxKind, "addfunc")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newBox := newObj.(*boxv1alpha1.Box)
			oldBox := oldObj.(*boxv1alpha1.Box)
			// Note: 更新 box 的 status 字段也会触发 UpdateFunc(进而触发 box 控制器), 因此需要忽略该情形
			if !reflect.DeepEqual(newBox.Spec, oldBox.Spec) {
				klog.V(4).InfoS("box spec changed", toString(newBox.Spec), "newBox.Spec", toString(oldBox.Spec), "oldBox.Spec")
				controller.enqueueBox(newObj, BoxKind, "updatefunc")
			} else {
				klog.V(4).Infof("just update box: %v status", newBox.Namespace+"/"+newBox.Name)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// 资源删除时会通过 ownerReference 级联删除 pod , 因此无需添加 pod 删除逻辑
			newDepl := obj.(*boxv1alpha1.Box)
			klog.V(4).Infof("delete: %v", newDepl.Namespace+"/"+newDepl.Name)
		},
	})

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newPod := obj.(*corev1.Pod)
			klog.V(4).Infof("gid: %v, add pod %v", GetGID(), newPod.Namespace+"/"+newPod.Name)
			controller.handlePodObject(obj, true)
		},
		UpdateFunc: func(old, new interface{}) {
			newPod := new.(*corev1.Pod)
			oldPod := old.(*corev1.Pod)
			// klog.V(4).Infof("update pod func: %v", newPod.Namespace+"/"+newPod.Name)
			if newPod.ResourceVersion == oldPod.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			klog.V(4).Infof("gid: %v, update pod %v, %v, %v", GetGID(), newPod.Namespace+"/"+newPod.Name, newPod.ResourceVersion, oldPod.ResourceVersion)
			// 去掉这个试一试
			controller.handlePodObject(new, true)
		},
		DeleteFunc: func(obj interface{}) {
			delpod := obj.(*corev1.Pod)
			klog.V(4).Infof("gid: %v, delete pod %v", GetGID(), delpod.Namespace+"/"+delpod.Name)
			controller.handlePodObject(obj, false)
		},
	})

	return controller
}

func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	// Start the informer factories to begin populating the informer caches
	logger.Info("Starting Box controller")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")

	// if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentsSynced, c.boxesSynced, c.podSynced); !ok {
	if ok := cache.WaitForCacheSync(ctx.Done(), c.boxesSynced, c.podSynced, c.boxDeploymentsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers", "count", workers)
	// Launch two workers to process Box resources
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	logger.Info("Started workers")
	// TODO: 搞不懂这里的 <-ctx.Done() 为啥不能注释掉
	<-ctx.Done()
	logger.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.workqueue.Get()
	logger := klog.FromContext(ctx)
	// klog.V(4).Info("call processNextWorkItem", toString(obj), shutdown)

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string

		switch item := obj.(type) {
		case string:
			key = item
			if err := c.syncHandler(ctx, key, true); err != nil {
				// Put the item back on the workqueue to handle any transient errors.
				c.workqueue.AddRateLimited(item)
				return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
			}
		case *UpdateStatusItem:
			key = item.Key
			// 只更新 box 状态, 不需要更新 pod
			if err := c.syncHandler(ctx, item.Key, false); err != nil {
				// Put the item back on the workqueue to handle any transient errors.
				c.workqueue.AddRateLimited(item)
				return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
			}
		}

		c.workqueue.Forget(obj)
		logger.Info("Successfully synced", "resourceName", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(ctx context.Context, key string, updatePod bool) error {
	// Convert the namespace/name string into a distinct namespace and name
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "resourceName", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the boxObj resource with this namespace/name
	boxObj, err := c.boxesLister.Boxes(namespace).Get(name)
	if err != nil {
		// The Box resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("Box '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	podName := boxObj.Name
	if podName == "" {
		utilruntime.HandleError(fmt.Errorf("%s: box name must be specified", key))
		return nil
	}

	newCreatePod := false
	pod, err := c.podLister.Pods(boxObj.Namespace).Get(podName)
	if errors.IsNotFound(err) {
		klog.V(4).Info("createObj ", toString(boxObj.Spec), "boxObj.Spec")

		// bugfix: 注意里面的 err 不要覆盖外层的 err
		var tmpPod *corev1.Pod
		tmpPod, err = c.newPodV2(boxObj, "create podv2")
		if err != nil {
			return err
		}

		pod, err = c.kubeclientset.CoreV1().Pods(boxObj.Namespace).Create(context.TODO(), tmpPod, metav1.CreateOptions{})
		newCreatePod = true
	}

	if err != nil {
		klog.V(4).Info("create pod err ", err)
		return err
	}

	if !metav1.IsControlledBy(pod, boxObj) {
		msg := fmt.Sprintf(MessageResourceExists, pod.Name)
		c.recorder.Event(boxObj, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf("%s", msg)
	}

	logger.V(4).Info("sync", "updatePod", updatePod, "newCreatePod", newCreatePod, " boxObj.Spec", boxObj.Spec, "boxObj.Status", boxObj.Status)

	// Note: 新创建的 pod, 就无需再次更新
	if !newCreatePod && updatePod {
		tmpPod, err := c.newPodV2(boxObj, "update podv2")
		if err != nil {
			return err
		}

		// Note: patch 时, 只保留 spec 字段即可
		tmpPodCopy := corev1.Pod{
			Spec: tmpPod.Spec,
		}
		patchdata, _ := json.Marshal(tmpPodCopy)
		klog.V(4).InfoS("patch pod", "json", string(patchdata))
		// Note: 试了多种 patch 类型, StrategicMergePatchType 目前未发现问题.
		// MergePatchType 需要每次都提交所有完整的字段
		pod, err = c.kubeclientset.CoreV1().Pods(boxObj.Namespace).Patch(context.TODO(), boxObj.Name, types.StrategicMergePatchType, []byte(patchdata), metav1.PatchOptions{})
		// klog.V(4).InfoS("patch result", toString(pod), "pod", "err", err)
		// Note: 更新 pod 时如果有禁止修改的字段, 则需要重建 pod
		// TODO: 重建 pod 的逻辑容易出问题, 先代办, 看看后面怎么优化解决
		if err != nil && strings.Contains(err.Error(), "Forbidden: pod updates may not change fields other than") {
			// klog.V(2).InfoS("patch foridden, try recreate", toString(pod), "pod", "err", fmt.Sprintf("V: %#v, T: %T", err, err))
			klog.V(2).InfoS("patch foridden, try recreate", toString(pod), "pod", "err", err, "IsForbidden", errors.IsForbidden(err), "err2", errors.IsInvalid(err))
			pod, err = c.RecreatePod(boxObj)
		}
	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return fmt.Errorf("update/recreate err, %v", err)
	}

	// Finally, we update the status block of the Box resource to reflect the
	// current state of the world
	err = c.updateBoxStatus(boxObj, pod)
	if err != nil {
		return fmt.Errorf("updateBoxStatus err, %v", err)
	}

	c.recorder.Event(boxObj, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) updateBoxStatus(boxObj *boxv1alpha1.Box, pod *corev1.Pod) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	if pod == nil {
		return fmt.Errorf("unknow error, pod is <nil>")
	}
	boxCopy := boxObj.DeepCopy()
	boxCopy.Status = boxv1alpha1.BoxStatusV2(pod.Status)
	// boxCopy.Status.Phase = string(pod.Status.Phase)
	// boxCopy.Status.Message = pod.Status.Message
	// boxCopy.Status.Reason = pod.Status.Reason
	// boxCopy.Status.HostIP = pod.Status.HostIP
	// boxCopy.Status.BoxIP = pod.Status.PodIP
	// boxCopy.Status.UpdateTimestamp = time.Now().String()
	_, err := c.boxclientset.CncosV1alpha1().Boxes(boxObj.Namespace).UpdateStatus(context.TODO(), boxCopy, metav1.UpdateOptions{})
	klog.V(4).InfoS("updateBoxStatus", toString(boxObj.Spec), "boxObj.Spec", "err", err)
	return err
}

func (c *Controller) enqueueBox(obj interface{}, kind string, msg string) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	// key = fmt.Sprintf("%v:%v", kind, key)
	klog.V(4).Infof("enqueue msg: %v, key: %v", msg, key)
	c.workqueue.Add(key)
}

func (c *Controller) handlePodObject(obj interface{}, justUpdateBoxStatus bool) {
	var object metav1.Object
	var ok bool
	logger := klog.FromContext(context.Background())
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		logger.V(4).Info("Recovered deleted object", "resourceName", object.GetName())
	}

	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Box, we should not do anything more
		// with it.
		if ownerRef.Kind != "Box" {
			return
		}

		logger.V(4).Info("Processing pod object", "object", klog.KObj(object))
		Box, err := c.boxesLister.Boxes(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			logger.V(4).Info("Ignore orphaned object", "object", klog.KObj(object), "Box", ownerRef.Name)
			return
		}

		if justUpdateBoxStatus {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				utilruntime.HandleError(err)
				return
			}
			c.workqueue.Add(&UpdateStatusItem{
				Key:    key,
				OldObj: Box,
				NewObj: Box,
			})
		} else {
			c.enqueueBox(Box, BoxKind, "box-handleobj")
		}
		return
	}
}

func (c *Controller) newPodV2(boxObj *boxv1alpha1.Box, msg string) (*corev1.Pod, error) {
	buf, _ := json.Marshal(boxObj)
	pod := &corev1.Pod{}
	if err := json.Unmarshal([]byte(buf), pod); err != nil {
		klog.V(2).InfoS("new podv2 Unmarshal err", "err", err)
		return nil, err
	}

	pod.APIVersion = "v1"
	pod.Kind = PodKind
	labels := boxObj.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	// labels["controller"] = boxObj.Name
	pod.ObjectMeta = metav1.ObjectMeta{
		Labels:      labels,
		Annotations: boxObj.Annotations,
		Name:        boxObj.Name,
		Namespace:   boxObj.Namespace,
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(boxObj, boxv1alpha1.SchemeGroupVersion.WithKind("Box")),
		},
	}
	py, _ := yaml.Marshal(pod)
	klog.V(4).InfoS("new podv2", "yaml", string(py))
	return pod, nil
}

func (c *Controller) RecreatePod(boxObj *boxv1alpha1.Box) (*corev1.Pod, error) {
	// Note: 重建 pod 需要立即删除, 否则在漫长的等待 Terminating 过程中会可能导致一系列的问题
	sec := int64(0)
	err := c.kubeclientset.CoreV1().Pods(boxObj.Namespace).Delete(context.Background(), boxObj.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &sec,
	})
	if err != nil {
		return nil, fmt.Errorf("delete pod <%s/%s> err: %v", boxObj.Namespace, boxObj.Name, err)
	}

	tmpPod, err := c.newPodV2(boxObj, "reCreate podv2")
	if err != nil {
		return nil, err
	}
	pod, err := c.kubeclientset.CoreV1().Pods(boxObj.Namespace).Create(context.TODO(), tmpPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("recreate pod <%s/%s> err: %v", boxObj.Namespace, boxObj.Name, err)
	}
	return pod, nil
}
