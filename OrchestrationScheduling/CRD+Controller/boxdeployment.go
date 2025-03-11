package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	boxdeploymentv1alpha1 "cncos.io/box-controller/pkg/apis/boxdeployment/v1alpha1"
	clientset "cncos.io/box-controller/pkg/box-generated/clientset/versioned"
	informers "cncos.io/box-controller/pkg/box-generated/informers/externalversions/box/v1alpha1"
	listers "cncos.io/box-controller/pkg/box-generated/listers/box/v1alpha1"
	boxdeploymentclientset "cncos.io/box-controller/pkg/boxdeployment-generated/clientset/versioned"
	boxdeploymentscheme "cncos.io/box-controller/pkg/boxdeployment-generated/clientset/versioned/scheme"
	boxdeploymentinformers "cncos.io/box-controller/pkg/boxdeployment-generated/informers/externalversions/boxdeployment/v1alpha1"
	boxdeploymentlisters "cncos.io/box-controller/pkg/boxdeployment-generated/listers/boxdeployment/v1alpha1"
)

type UpdateItem struct {
	Kind   string
	OldObj interface{}
	NewObj interface{}
}

type BoxDeploymentController struct {
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

	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewBoxDeploymentController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	boxclientset clientset.Interface,
	boxdeployclientset boxdeploymentclientset.Interface,
	podInformer corev1informers.PodInformer,
	// deploymentInformer appsinformers.DeploymentInformer,
	boxdeployInformer boxdeploymentinformers.BoxDeploymentInformer,
	boxInformer informers.BoxInformer) *BoxDeploymentController {
	logger := klog.FromContext(ctx)

	// Create event broadcaster
	// Add cncos-box-controller types to the default Kubernetes Scheme so Events can be
	// logged for cncos-box-controller types.
	utilruntime.Must(boxdeploymentscheme.AddToScheme(scheme.Scheme))
	logger.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &BoxDeploymentController{
		kubeclientset:        kubeclientset,
		boxclientset:         boxclientset,
		boxdeployclientset:   boxdeployclientset,
		boxesLister:          boxInformer.Lister(),
		boxesSynced:          boxInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "BoxDeployments"),
		recorder:             recorder,
		boxDeploymentsLister: boxdeployInformer.Lister(),
		boxDeploymentsSynced: boxdeployInformer.Informer().HasSynced,
	}

	logger.Info("Setting up event handlers")
	boxdeployInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			newDepl := new.(*boxdeploymentv1alpha1.BoxDeployment)
			klog.V(4).Infof("add boxdp: %v", newDepl.Namespace+"/"+newDepl.Name)
			controller.enqueueBoxDeployment(new, BoxDeploymentKind, "addfunc")
		},
		UpdateFunc: func(old, new interface{}) {
			// Note: 之前的逻辑这里存在数据竞争, 具体分析如下:
			// 删除 box 触发 boxdp 状态更新 -> 触发 box 更新同步函数 boxDeploymentUpdateHdlr -> 创建 box；
			// 删除 box 触发 enqueue -> 触发 syncBoxDeploymentHandler -> 创建 box
			oldDepl := old.(*boxdeploymentv1alpha1.BoxDeployment)
			newDepl := new.(*boxdeploymentv1alpha1.BoxDeployment)
			// controller.enqueueBoxDeployment(new, BoxDeploymentKind, "updatefunc")
			if !reflect.DeepEqual(oldDepl.Spec, newDepl.Spec) {
				klog.V(4).InfoS("boxdeployment spec changed", toString(newDepl.Spec), "newBox.Spec", toString(oldDepl.Spec), "oldBox.Spec")

				// controller.boxDeploymentUpdateHdlr(oldDepl, newDepl)
				controller.workqueue.Add(&UpdateItem{
					Kind:   BoxDeploymentKind,
					OldObj: oldDepl,
					NewObj: newDepl,
				})
			} else {
				klog.V(4).InfoS("just boxdeployment status changed, ignore", "boxdeployment", newDepl.Namespace+"/"+newDepl.Name)
			}
		},
		DeleteFunc: func(new interface{}) {
			// 资源删除时会通过 ownerReference 级联删除, 因此无需添加删除逻辑
			newDepl := new.(*boxdeploymentv1alpha1.BoxDeployment)
			klog.V(4).Infof("delete boxdp: %v", newDepl.Namespace+"/"+newDepl.Name)
		},
	})

	// Set up an event handler for when Box resources change
	boxInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			newDepl := new.(*boxv1alpha1.Box)
			klog.V(4).Infof("add box: %v", newDepl.Namespace+"/"+newDepl.Name)
			// controller.enqueueBoxDeployment(new, BoxKind, "addfunc")
			controller.handleBoxObject(new)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newBox := newObj.(*boxv1alpha1.Box)
			oldBox := oldObj.(*boxv1alpha1.Box)
			_ = oldBox
			klog.V(4).Infof("update box: %v", newBox.Namespace+"/"+newBox.Name)
			// // Note: 更新 box 的 status 字段也会触发 UpdateFunc(进而触发 box 控制器), 因此需要忽略该情形
			// if !reflect.DeepEqual(newBox.Spec, oldBox.Spec) {
			// 	klog.V(4).InfoS("box spec changed", toString(newBox.Spec), "newBox.Spec", toString(oldBox.Spec), "oldBox.Spec")
			// 	controller.enqueueBoxDeployment(newObj, BoxKind, "updatefunc")
			// }
			controller.handleBoxObject(newBox)
		},
		DeleteFunc: func(obj interface{}) {
			// 资源删除时会通过 ownerReference 级联删除 pod , 因此无需添加 pod 删除逻辑
			newDepl := obj.(*boxv1alpha1.Box)
			klog.V(4).Infof("delete: %v", newDepl.Namespace+"/"+newDepl.Name)

			// Note: box 被 boxdeployment 控制后, 删除 box 时, 需要通知关联的 ownerReference 新启动一个 box
			controller.handleBoxObject(obj)
		},
	})

	return controller
}

func (c *BoxDeploymentController) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	// Start the informer factories to begin populating the informer caches
	logger.Info("Starting Box controller")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")

	// if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentsSynced, c.boxesSynced, c.podSynced); !ok {
	if ok := cache.WaitForCacheSync(ctx.Done(), c.boxesSynced, c.boxDeploymentsSynced); !ok {
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

func (c *BoxDeploymentController) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *BoxDeploymentController) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.workqueue.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		// var ok bool

		switch item := obj.(type) {
		case string:
			key = item
			if err := c.syncHandler(ctx, key); err != nil {
				// Put the item back on the workqueue to handle any transient errors.
				c.workqueue.AddRateLimited(item)
				return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
			}
		case *UpdateItem:
			if err := c.updateHandler(item.OldObj.(*boxdeploymentv1alpha1.BoxDeployment), item.NewObj.(*boxdeploymentv1alpha1.BoxDeployment)); err != nil {
				// Put the item back on the workqueue to handle any transient errors.
				c.workqueue.AddRateLimited(item)
				return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
			}
		}

		// if key, ok = obj.(string); !ok {
		// 	c.workqueue.Forget(obj)
		// 	utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		// 	return nil
		// }
		// if err := c.syncBoxDeploymentHandler(ctx, key); err != nil {
		// 	// Put the item back on the workqueue to handle any transient errors.
		// 	c.workqueue.AddRateLimited(key)
		// 	return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		// }

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

func (c *BoxDeploymentController) enqueueBoxDeployment(obj interface{}, kind string, msg string) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.V(4).Infof("msg: %v, key: %v", msg, key)
	c.workqueue.Add(key)
}

func convertPodSpec2BoxSpecV2(podSpec *corev1.PodSpec) *boxv1alpha1.BoxSpecV2 {
	buf, _ := json.Marshal(podSpec)
	boxSpecv2 := &boxv1alpha1.BoxSpecV2{}
	if err := json.Unmarshal([]byte(buf), boxSpecv2); err != nil {
		klog.V(2).InfoS("convertPodSpec2BoxSpecV2 Unmarshal err", "err", err)
		// return nil, err

		return &boxv1alpha1.BoxSpecV2{
			Volumes:                       podSpec.Volumes,
			InitContainers:                podSpec.InitContainers,
			Containers:                    podSpec.Containers,
			EphemeralContainers:           podSpec.EphemeralContainers,
			RestartPolicy:                 podSpec.RestartPolicy,
			TerminationGracePeriodSeconds: podSpec.TerminationGracePeriodSeconds,
			ActiveDeadlineSeconds:         podSpec.ActiveDeadlineSeconds,
			DNSPolicy:                     podSpec.DNSPolicy,
			NodeSelector:                  podSpec.NodeSelector,

			ServiceAccountName:           podSpec.ServiceAccountName,
			DeprecatedServiceAccount:     podSpec.DeprecatedServiceAccount,
			AutomountServiceAccountToken: podSpec.AutomountServiceAccountToken,

			NodeName:                  podSpec.NodeName,
			HostNetwork:               podSpec.HostNetwork,
			HostPID:                   podSpec.HostPID,
			HostIPC:                   podSpec.HostIPC,
			ShareProcessNamespace:     podSpec.ShareProcessNamespace,
			SecurityContext:           podSpec.SecurityContext,
			ImagePullSecrets:          podSpec.ImagePullSecrets,
			Hostname:                  podSpec.Hostname,
			Subdomain:                 podSpec.Subdomain,
			Affinity:                  podSpec.Affinity,
			SchedulerName:             podSpec.SchedulerName,
			Tolerations:               podSpec.Tolerations,
			HostAliases:               podSpec.HostAliases,
			PriorityClassName:         podSpec.PriorityClassName,
			Priority:                  podSpec.Priority,
			DNSConfig:                 podSpec.DNSConfig,
			ReadinessGates:            podSpec.ReadinessGates,
			RuntimeClassName:          podSpec.RuntimeClassName,
			EnableServiceLinks:        podSpec.EnableServiceLinks,
			PreemptionPolicy:          podSpec.PreemptionPolicy,
			Overhead:                  podSpec.Overhead,
			TopologySpreadConstraints: podSpec.TopologySpreadConstraints,
			SetHostnameAsFQDN:         podSpec.SetHostnameAsFQDN,
			OS:                        podSpec.OS,

			HostUsers: podSpec.HostUsers,

			SchedulingGates: podSpec.SchedulingGates,
			ResourceClaims:  podSpec.ResourceClaims,
		}
	}
	return boxSpecv2
}

func (c *BoxDeploymentController) newBox(boxdpObj *boxdeploymentv1alpha1.BoxDeployment, msg string) *boxv1alpha1.Box {
	// labels := map[string]string{
	// 	"controller": boxdpObj.Name,
	// }
	boxName := boxdpObj.Name + "-" + randomString(6)
	klog.V(4).Infof("gid: %v, new box: %v, msg: %v", GetGID(), boxName, msg)
	boxSpecv2 := convertPodSpec2BoxSpecV2(&boxdpObj.Spec.Template.Spec)
	// if err != nil {
	// 	klog.V(2).InfoS("convertPodSpec2BoxSpecV2 Unmarshal err", "err", err)
	// 	return nil
	// }
	ret := &boxv1alpha1.Box{
		ObjectMeta: boxdpObj.Spec.Template.ObjectMeta,
		Spec:       *boxSpecv2,
	}
	if ret.ObjectMeta.Labels == nil {
		ret.ObjectMeta.Labels = make(map[string]string)
	}
	// Note: 不清楚为啥不能加上 controller 字段, 会导致 boxdeployment  spec.template.metadata.labels 莫名其妙的出现 controller 字段
	// ret.ObjectMeta.Labels["controller"] = boxdpObj.Name
	ret.ObjectMeta.Name = boxName
	ret.ObjectMeta.Namespace = boxdpObj.Namespace
	ret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(boxdpObj, boxdeploymentv1alpha1.SchemeGroupVersion.WithKind(BoxDeploymentKind)),
	}

	// ret := &boxv1alpha1.Box{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Labels:    labels,
	// 		Name:      boxName,
	// 		Namespace: boxdpObj.Namespace,
	// 		OwnerReferences: []metav1.OwnerReference{
	// 			*metav1.NewControllerRef(boxdpObj, boxdeploymentv1alpha1.SchemeGroupVersion.WithKind(BoxDeploymentKind)),
	// 		},
	// 	},
	// 	// Spec: boxv1alpha1.BoxSpec{
	// 	Spec: boxv1alpha1.BoxSpecV2{
	// 		// BoxName:         boxName,
	// 		// Containers: []boxv1alpha1.Container{
	// 		// Containers: []corev1.Container{
	// 		// 	{
	// 		// 		Name:            boxName,
	// 		// 		Image:           boxdpObj.Spec.Template.Spec.Containers[0].Image,
	// 		// 		ImagePullPolicy: corev1.PullPolicy(boxdpObj.Spec.ImagePullPolicy),
	// 		// 	},
	// 		// },
	// 		Containers: boxdpObj.Spec.Template.Spec.Containers,
	// 	},
	// }

	return ret
}

func (c *BoxDeploymentController) syncHandler(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "resourceName", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the boxdpObj resource with this namespace/name
	boxdpObj, err := c.boxDeploymentsLister.BoxDeployments(namespace).Get(name)
	if err != nil {
		// The Box resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("Box '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	boxDpName := boxdpObj.Name
	if boxDpName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: box name must be specified", key))
		return nil
	}

	// 判断 box 资源是否存在，是否和 replicas 个数相等
	boxList, err := c.ListBoxByOwnerRerference(ctx, boxdpObj, false)
	logger.V(4).Info("owner list info", "length", len(boxList), "list", boxList)
	if err != nil {
		klog.V(2).Info(err)
		return err
	}
	// 默认 replicas 为 1
	replicas := 1
	if boxdpObj.Spec.Replicas != nil {
		replicas = int(*boxdpObj.Spec.Replicas)
	}

	// Note: 需要更新 boxdeployment status, 所以不能提前 return
	// if len(boxList) == replicas {
	// 	return nil
	// }
	if len(boxList) < replicas {
		for i := 0; i < replicas-len(boxList); i++ {
			_, err = c.boxclientset.CncosV1alpha1().Boxes(boxdpObj.Namespace).Create(context.TODO(), c.newBox(boxdpObj, "for create box"), metav1.CreateOptions{})
			if err != nil {
				logger.V(4).Info("create box err ", err)
				return err
			}
		}
	}
	if len(boxList) > replicas {
		for i := 0; i < len(boxList)-replicas; i++ {
			item := boxList[i]
			klog.V(4).InfoS("delete box", "name", item.Name, "ns", item.Namespace)
			if err := c.boxclientset.CncosV1alpha1().Boxes(boxdpObj.Namespace).Delete(context.Background(), item.Name, metav1.DeleteOptions{}); err != nil {
				logger.V(4).Info("delete box err ", err)
				return err
			}
		}
	}

	// TODO: update boxdeployment status
	if err = c.updateStatus(boxdpObj, int32(replicas)); err != nil {
		return err
	}
	c.recorder.Event(boxdpObj, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *BoxDeploymentController) updateStatus(boxdpObj *boxdeploymentv1alpha1.BoxDeployment, availableReplicas int32) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	klog.V(4).InfoS("updateBoxDeploymentStatus", "availableReplicas", availableReplicas, "Status.AvailableReplicas", boxdpObj.Status.AvailableReplicas)
	if availableReplicas == boxdpObj.Status.AvailableReplicas {
		return nil
	}
	boxdpCopy := boxdpObj.DeepCopy()
	boxdpCopy.Status.AvailableReplicas = availableReplicas
	// boxdpCopy.Status.UpdateTimestamp = time.Now().String()
	_, err := c.boxdeployclientset.CncosV1alpha1().BoxDeployments(boxdpObj.Namespace).UpdateStatus(context.TODO(), boxdpCopy, metav1.UpdateOptions{})
	klog.V(4).InfoS("updateBoxDeploymentStatus", toString(boxdpObj.Spec), "boxdpCopy.Spec", "err", err)
	return err
}

// 删除旧资源时需要根据 uid 进行过滤
// Note2: 资源的 uid 是固定值, 所以根据 uid 进行过滤无效
func (c *BoxDeploymentController) ListBoxByOwnerRerference(ctx context.Context, boxdpObj *boxdeploymentv1alpha1.BoxDeployment, checkUid bool) ([]*boxv1alpha1.Box, error) {
	logger := klog.FromContext(context.Background())
	boxList, err := c.boxesLister.Boxes(boxdpObj.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var ret []*boxv1alpha1.Box
	for _, item := range boxList {
		item := item
		for _, owner := range item.OwnerReferences {
			logger.V(4).Info("diffInfo", owner.APIVersion, boxdpObj.APIVersion,
				owner.Kind, boxdpObj.Kind, owner.Name, boxdpObj.Name, owner.UID, boxdpObj.UID,
			)
			// Note: 有时 boxdpObj 的 apiversion 和 kind 字段为空, 所以需要和固定值进行比较
			if owner.APIVersion == "cncos.io/v1alpha1" && owner.Kind == BoxDeploymentKind && owner.Name == boxdpObj.Name {
				ret = append(ret, item)
			}
		}
	}

	logger.V(4).Info("owner list info", "length", len(ret), "list", ret)
	return ret, err
}

func (c *BoxDeploymentController) updateHandler(oldObj, newObj *boxdeploymentv1alpha1.BoxDeployment) error {
	logger := klog.FromContext(context.Background())
	oldSpec := oldObj.Spec.DeepCopy()
	newSpec := newObj.Spec.DeepCopy()
	// oldReplicas := 0
	// if oldSpec.Replicas != nil {
	// 	oldReplicas = int(*oldSpec.Replicas)
	// }
	newReplicas := 0
	if newSpec.Replicas != nil {
		newReplicas = int(*newSpec.Replicas)
	}
	oldSpec.Replicas = nil
	newSpec.Replicas = nil

	deeqEqual := reflect.DeepEqual(oldSpec, newSpec)
	logger.V(4).Info("check boxdeployment.spec", "DeepEqual", deeqEqual)
	if deeqEqual {
		// 只调整副本数
		// // Spec 相同, 副本数也相同, 说明只是状态变化
		// if oldReplicas == newReplicas {
		// 	klog.V(4).InfoS("just boxdeployment status changed, ignore")
		// 	return nil
		// }
		// logger.V(4).Info("uid diff", "oldUid", oldObj.UID, "newUid", newObj.UID)
		// logger.V(4).Info("oldOjb debug", toString(oldObj), "oldOjb")
		boxList, err := c.ListBoxByOwnerRerference(context.Background(), oldObj, false)
		logger.V(4).Info("boxdeployment spec not changed, just update replicas.", "length", len(boxList), "list", boxList)
		if err != nil {
			klog.V(2).Info(err)
			return err
		}
		if len(boxList) < newReplicas {
			for i := 0; i < newReplicas-len(boxList); i++ {
				_, err = c.boxclientset.CncosV1alpha1().Boxes(oldObj.Namespace).Create(context.TODO(), c.newBox(oldObj, "for create box"), metav1.CreateOptions{})
				if err != nil {
					logger.V(4).Info("create box err ", err)
					return err
				}
			}
		} else if len(boxList) > newReplicas {
			for i := 0; i < len(boxList)-newReplicas; i++ {
				item := boxList[i]
				klog.V(4).InfoS("delete box", "name", item.Name, "ns", item.Namespace)
				if err := c.boxclientset.CncosV1alpha1().Boxes(oldObj.Namespace).Delete(context.Background(), item.Name, metav1.DeleteOptions{}); err != nil {
					logger.V(4).Info("delete box err ", err)
					return err
				}
			}
		} else {
			logger.V(4).Info("boxdeployment not changed", "boxdpName", oldObj.Namespace+"/"+oldObj.Name)
		}
		return nil
	}

	klog.V(4).InfoS("boxdeploy spec changed",
		"ObjectMeta-label-deepequal", reflect.DeepEqual(oldSpec.Template.ObjectMeta.Labels, newSpec.Template.ObjectMeta.Labels),
		toString(newSpec), "newSpec", toString(oldSpec), "oldSpec",
		toString(oldSpec.Template.ObjectMeta.Labels), "oldDepl.Spec.Template.ObjectMeta.Labels",
		toString(newSpec.Template.ObjectMeta.Labels), "newSpec.Template.ObjectMeta.Labels")

	// 待删除的旧的 box
	boxList, err := c.ListBoxByOwnerRerference(context.Background(), oldObj, true)
	logger.V(4).Info("boxdp changed. old list info", "length", len(boxList), "list", boxList)
	if err != nil {
		klog.V(2).Info(err)
		return err
	}

	// 创建新的 box
	for i := 0; i < newReplicas; i++ {
		if _, err := c.boxclientset.CncosV1alpha1().Boxes(oldObj.Namespace).Create(context.TODO(), c.newBox(newObj, "for create box"), metav1.CreateOptions{}); err != nil {
			logger.V(4).Info("create box err ", err)
			return err
		}
	}

	for _, item := range boxList {
		klog.V(4).InfoS("delete box", "name", item.Name, "ns", item.Namespace)
		if err := c.boxclientset.CncosV1alpha1().Boxes(oldObj.Namespace).Delete(context.Background(), item.Name, metav1.DeleteOptions{}); err != nil {
			logger.V(4).Info("delete box err ", err)
			return err
		}
	}
	return nil
}

func (c *BoxDeploymentController) handleBoxObject(obj interface{}) {
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

	logger.V(4).Info("Processing box object", "object", klog.KObj(object))
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != BoxDeploymentKind {
			return
		}

		boxdp, err := c.boxDeploymentsLister.BoxDeployments(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			logger.V(4).Info("Ignore orphaned object", "object", klog.KObj(object), "BoxDeployment", ownerRef.Name)
			return
		}

		// Note: enqueueBoxDeployment 会触发 syncBoxDeploymentHandler 从而更新 status 状态, 因此此处的更新状态逻辑可以去掉

		// // // Note: status 字段可用副本数先减去 1
		// // // 由于只有 box 删除才会触发 BoxKind 的 handleObject 调用, 因此该逻辑是合理的. 如果后续其他类型的 box 操作也会触发 handleObject, 此处的逻辑需要同步更改
		// // newAvailableReplicas := boxdp.Status.AvailableReplicas - 1
		// // if newAvailableReplicas < 0 {
		// // 	newAvailableReplicas = 0
		// // }
		// if ownerRef, err := c.ListBoxByOwnerRerference(context.TODO(), boxdp, false); err == nil {
		// 	newAvailableReplicas := len(ownerRef)
		// 	c.updateBoxDeploymentStatus(boxdp, int32(newAvailableReplicas))
		// }

		c.enqueueBoxDeployment(boxdp, BoxDeploymentKind, "boxdp-handleobj")
		return
	}
}
