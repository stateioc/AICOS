package box_statefulset

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	boxstatesetv1alpha1 "cncos.io/box-controller/pkg/apis/boxstatefulset/v1alpha1"
	boxclientset "cncos.io/box-controller/pkg/box-generated/clientset/versioned"
	boxinformers "cncos.io/box-controller/pkg/box-generated/informers/externalversions/box/v1alpha1"
	boxlisters "cncos.io/box-controller/pkg/box-generated/listers/box/v1alpha1"
	boxstatefulsetclientset "cncos.io/box-controller/pkg/boxstatefulset-generated/clientset/versioned"
	boxstatefulsetscheme "cncos.io/box-controller/pkg/boxstatefulset-generated/clientset/versioned/scheme"
	boxstatefulsetinformers "cncos.io/box-controller/pkg/boxstatefulset-generated/informers/externalversions/boxstatefulset/v1alpha1"
	boxstatefulsetlisters "cncos.io/box-controller/pkg/boxstatefulset-generated/listers/boxstatefulset/v1alpha1"
	"cncos.io/box-controller/pkg/history"
	boxutil "cncos.io/box-controller/pkg/util/box"
)

type BoxStatefulSetController struct {
	// kubeClientSet is a standard kubernetes clientset
	kubeClientSet kubernetes.Interface
	control       BoxStatefulSetControlInterface
	// boxClientSet is a clientset for our own API group
	boxClientSet            boxclientset.Interface
	boxStatefulSetClientSet boxstatefulsetclientset.Interface

	boxesLister          boxlisters.BoxLister
	boxStatefulSetLister boxstatefulsetlisters.BoxStatefulSetLister

	boxesSynced          cache.InformerSynced
	boxStatefulSetSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewBoxStatefulSetController(
	ctx context.Context,
	kubeClientSet kubernetes.Interface,
	boxClientSet boxclientset.Interface,
	boxStatefulSetClientSet boxstatefulsetclientset.Interface,
	boxStatefulSetInformer boxstatefulsetinformers.BoxStatefulSetInformer,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	revInformer appsinformers.ControllerRevisionInformer,
	boxInformer boxinformers.BoxInformer) *BoxStatefulSetController {
	logger := klog.FromContext(ctx)

	// Create event broadcaster
	// Add cncos-box-controller types to the default Kubernetes Scheme so Events can be
	// logged for cncos-box-controller types.
	utilruntime.Must(boxstatefulsetscheme.AddToScheme(scheme.Scheme))
	logger.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "box-stateful-set-controller"})

	controller := &BoxStatefulSetController{
		kubeClientSet:           kubeClientSet,
		boxClientSet:            boxClientSet,
		boxStatefulSetClientSet: boxStatefulSetClientSet,
		boxesLister:             boxInformer.Lister(),
		boxesSynced:             boxInformer.Informer().HasSynced,
		workqueue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "BoxDeployments"),
		recorder:                recorder,
		boxStatefulSetLister:    boxStatefulSetInformer.Lister(),
		boxStatefulSetSynced:    boxStatefulSetInformer.Informer().HasSynced,
		control: NewDefaultBoxStatefulSetControl(
			NewStatefulBoxControl(kubeClientSet, boxClientSet, boxInformer.Lister(), pvcInformer.Lister(), recorder),
			NewRealBoxStatefulSetStatusUpdater(boxStatefulSetClientSet),
			history.NewHistory(kubeClientSet, revInformer.Lister()),
			recorder),
	}

	logger.Info("Setting up event handlers")
	boxStatefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addBoxStatefulSet,
		UpdateFunc: controller.updateBoxStatefulSet,
		DeleteFunc: controller.deleteBoxStatefulSet,
	})

	// Set up an event handler for when Box resources change
	boxInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addBox,
		UpdateFunc: controller.updateBox,
		DeleteFunc: controller.deleteBox,
	})

	return controller
}

func (c *BoxStatefulSetController) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	// Start the informer factories to begin populating the informer caches
	logger.Info("Starting BoxStatefulSet Controller")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")

	// if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentsSynced, c.boxesSynced, c.podSynced); !ok {
	if ok := cache.WaitForCacheSync(ctx.Done(), c.boxesSynced, c.boxStatefulSetSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers", "count", workers)
	// Launch two workers to process Box resources
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")

	return nil
}

func (c *BoxStatefulSetController) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *BoxStatefulSetController) processNextWorkItem(ctx context.Context) bool {
	key, quit := c.workqueue.Get()
	if quit {
		return false
	}
	defer c.workqueue.Done(key)
	if err := c.sync(ctx, key.(string)); err != nil {
		utilruntime.HandleError(fmt.Errorf("error syncing BoxStatefulSet %v, requeuing: %v", key.(string), err))
		c.workqueue.AddRateLimited(key)
	} else {
		c.workqueue.Forget(key)
	}
	return true
}

func (c *BoxStatefulSetController) sync(ctx context.Context, key string) error {
	startTime := time.Now()
	klog.V(4).Info("start syncing boxstatefulset", "key", key)
	defer func() {
		klog.V(4).Info("Finished syncing boxstatefulset", "key", key, "time", time.Since(startTime))
	}()
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	boxSts, err := c.boxStatefulSetClientSet.CncosV1alpha1().BoxStatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		klog.Info("BoxStatefulSet has been deleted", "key", key)
		return nil
	}
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to retrieve BoxStatefulSet %v from store: %v", key, err))
		return err
	}
	selector, err := metav1.LabelSelectorAsSelector(boxSts.Spec.Selector)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error converting BoxStatefulSet %v selector: %v", key, err))
		// This is a non-transient error, so don't retry.
		return nil
	}
	boxes, err := c.getBoxesForBoxSts(boxSts, selector)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error to get boxes by BoxStatefulSet %v selector: %v", key, err))
		return err
	}
	if err := c.syncBoxStatefulSet(ctx, boxSts, boxes); err != nil {
		utilruntime.HandleError(fmt.Errorf("error to sync BoxStatefulSet %v err: %v", key, err))
		return err
	}
	return nil
}

func (c *BoxStatefulSetController) syncBoxStatefulSet(ctx context.Context, boxSts *boxstatesetv1alpha1.BoxStatefulSet, boxes []*boxv1alpha1.Box) error {
	var status *boxstatesetv1alpha1.BoxStatefulSetStatus
	var err error
	status, err = c.control.UpdateBoxStatefulSet(ctx, boxSts, boxes)
	if err != nil {
		klog.Error(err, "UpdateBoxStatefulSet failed")
		return err
	}
	klog.V(4).Info("Successfully synced BoxStatefulSet", klog.KObj(boxSts))
	// One more sync to handle the clock skew. This is also helping in requeuing right after status update
	if boxSts.Spec.MinReadySeconds > 0 && status != nil && status.AvailableReplicas != *boxSts.Spec.Replicas {
		c.enqueueBoxStsAfter(boxSts, "", time.Duration(boxSts.Spec.MinReadySeconds)*time.Second)
	}

	return nil
}

func (c *BoxStatefulSetController) getBoxesForBoxSts(boxSts *boxstatesetv1alpha1.BoxStatefulSet, selector labels.Selector) ([]*boxv1alpha1.Box, error) {
	boxes, err := c.boxesLister.Boxes(boxSts.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	boxList := make([]*boxv1alpha1.Box, 0)
	for _, item := range boxes {
		controllerRef := metav1.GetControllerOfNoCopy(item)
		if controllerRef == nil {
			continue
		}
		if controllerRef.Kind != boxstatesetv1alpha1.BoxStatefulSetKind {
			continue
		}
		if controllerRef.UID != boxSts.UID {
			continue
		}
		boxList = append(boxList, item)
	}
	return boxList, nil
}

func (c *BoxStatefulSetController) addBoxStatefulSet(obj interface{}) {
	newBoxStatefulSet := obj.(*boxstatesetv1alpha1.BoxStatefulSet)
	klog.V(4).Infof("add BoxStatefulSet: %s/%s", newBoxStatefulSet.Namespace, newBoxStatefulSet.Name)
	c.enqueueBoxStatefulSet(newBoxStatefulSet, "addBoxStatefulSet")
}

func (c *BoxStatefulSetController) updateBoxStatefulSet(old, cur interface{}) {
	newBoxStatefulSet := cur.(*boxstatesetv1alpha1.BoxStatefulSet)
	oldBoxStatefulSet := old.(*boxstatesetv1alpha1.BoxStatefulSet)
	if reflect.DeepEqual(oldBoxStatefulSet.Spec, newBoxStatefulSet.Spec) {
		klog.V(4).Infof("just BoxStatefulSet(%s/%s) status changed, ignore", newBoxStatefulSet.Namespace, newBoxStatefulSet.Name)
		return
	}
	c.enqueueBoxStatefulSet(newBoxStatefulSet, "updateBoxStatefulSet")
}

func (c *BoxStatefulSetController) deleteBoxStatefulSet(obj interface{}) {
	// 资源删除时会通过 ownerReference 级联删除, 因此无需添加删除逻辑
	newBoxStatefulSet := obj.(*boxstatesetv1alpha1.BoxStatefulSet)
	klog.V(4).Infof("delete BoxStatefulSet: %s/%s", newBoxStatefulSet.Namespace, newBoxStatefulSet.Name)
}

func (c *BoxStatefulSetController) addBox(obj interface{}) {
	box := obj.(*boxv1alpha1.Box)
	if box.DeletionTimestamp != nil {
		c.deleteBox(box)
		return
	}
	// If it has a ControllerRef, that's all that matters.
	controllerRef := metav1.GetControllerOf(box)
	if controllerRef == nil {
		return
	}
	boxSts := c.resolveControllerRef(context.TODO(), box.Namespace, controllerRef)
	if boxSts == nil {
		return
	}
	c.enqueueBoxStatefulSet(boxSts, "addBox")
	return
}

func (c *BoxStatefulSetController) updateBox(old, cur interface{}) {
	newBox := cur.(*boxv1alpha1.Box)
	oldBox := old.(*boxv1alpha1.Box)
	if newBox.ResourceVersion == oldBox.ResourceVersion {
		// In the event of a re-list we may receive update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}
	curControllerRef := metav1.GetControllerOf(newBox)
	oldControllerRef := metav1.GetControllerOf(oldBox)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if boxSts := c.resolveControllerRef(context.TODO(), oldBox.Namespace, oldControllerRef); boxSts != nil {
			c.enqueueBoxStatefulSet(boxSts, "updateBox")
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef == nil {
		return
	}

	boxSts := c.resolveControllerRef(context.TODO(), newBox.Namespace, curControllerRef)
	if boxSts == nil {
		return
	}
	klog.V(4).Infof("Box(%s/%s) updated", newBox.Namespace, newBox.Name)
	c.enqueueBoxStatefulSet(boxSts, "updateBox")

	if !boxutil.IsBoxReady(oldBox) && boxutil.IsBoxReady(newBox) && boxSts.Spec.MinReadySeconds > 0 {
		klog.V(2).Info("BoxStatefulSet will be enqueued after minReadySeconds for availability check", "BoxStatefulSet", klog.KObj(boxSts), "minReadySeconds", boxSts.Spec.MinReadySeconds)
		// Add a second to avoid milliseconds skew in AddAfter.
		// See https://github.com/kubernetes/kubernetes/issues/39785#issuecomment-279959133 for more info.
		c.enqueueBoxStsAfter(boxSts, "availabilityCheck", (time.Duration(boxSts.Spec.MinReadySeconds)*time.Second)+time.Second)
	}
}

func (c *BoxStatefulSetController) deleteBox(obj interface{}) {
	box, ok := obj.(*boxv1alpha1.Box)

	// When a delete is dropped, the relist will notice a box in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		box, ok = tombstone.Obj.(*boxv1alpha1.Box)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a box %+v", obj))
			return
		}
	}

	controllerRef := metav1.GetControllerOf(box)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	boxSts := c.resolveControllerRef(context.TODO(), box.Namespace, controllerRef)
	if boxSts == nil {
		return
	}
	klog.V(4).Info("Box deleted.", "box", klog.KObj(box), "caller", utilruntime.GetCaller())
	c.enqueueBoxStatefulSet(boxSts, "deleteBox")
}

func (c *BoxStatefulSetController) enqueueBoxStatefulSet(obj interface{}, msg string) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.V(4).Infof("msg: %v, key: %v", msg, key)
	c.workqueue.Add(key)
}

func (c *BoxStatefulSetController) resolveControllerRef(ctx context.Context, namespace string, controllerRef *metav1.OwnerReference) *boxstatesetv1alpha1.BoxStatefulSet {
	if controllerRef.Kind != boxstatesetv1alpha1.BoxStatefulSetKind {
		return nil
	}
	boxSts, err := c.boxStatefulSetClientSet.CncosV1alpha1().BoxStatefulSets(namespace).Get(ctx, controllerRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	if boxSts.UID != controllerRef.UID {
		return nil
	}
	return boxSts
}

// enqueueBoxStatefulSet enqueues the given boxstatefulset in the work queue after given time
func (c *BoxStatefulSetController) enqueueBoxStsAfter(obj interface{}, msg string, duration time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.V(4).Infof("msg: %v, key: %v", msg, key)
	c.workqueue.AddAfter(key, duration)
}
