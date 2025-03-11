package box_statefulset

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/integer"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	boxstatesetv1alpha1 "cncos.io/box-controller/pkg/apis/boxstatefulset/v1alpha1"
	"cncos.io/box-controller/pkg/history"
)

const MaxBatchSize = 500

type BoxStatefulSetControlInterface interface {
	// UpdateBoxStatefulSet implements the control logic for Box creation, update, and deletion, and
	// persistent volume creation, update, and deletion.
	// If an implementation returns a non-nil error, the invocation will be retried using a rate-limited strategy.
	// Implementors should sink any errors that they do not wish to trigger a retry, and they may feel free to
	// exit exceptionally at any point provided they wish the update to be re-run at a later point in time.
	UpdateBoxStatefulSet(ctx context.Context, boxSts *boxstatesetv1alpha1.BoxStatefulSet, boxes []*boxv1alpha1.Box) (*boxstatesetv1alpha1.BoxStatefulSetStatus, error)
	// ListRevisions returns a array of the ControllerRevisions that represent the revisions of BoxStatefulSet. If the returned
	// error is nil, the returns slice of ControllerRevisions is valid.
	ListRevisions(boxSts *boxstatesetv1alpha1.BoxStatefulSet) ([]*apps.ControllerRevision, error)
	// AdoptOrphanRevisions adopts any orphaned ControllerRevisions that match BoxStatefulSet's Selector. If all adoptions are
	// successful the returned error is nil.
	AdoptOrphanRevisions(boxSts *boxstatesetv1alpha1.BoxStatefulSet, revisions []*apps.ControllerRevision) error
}

// NewDefaultBoxStatefulSetControl returns a new instance of the default implementation BoxStatefulSetControlInterface that
// implements the documented semantics for BoxStatefulSets. boxControl is the BoxControlInterface used to create, update,
// and delete Boxes and to create PersistentVolumeClaims. statusUpdater is the BoxStatefulSetStatusUpdaterInterface used
// to update the status of BoxStatefulSets.
func NewDefaultBoxStatefulSetControl(
	boxControl *StatefulBoxControl,
	statusUpdater BoxStatefulSetStatusUpdaterInterface,
	controllerHistory history.Interface,
	recorder record.EventRecorder) BoxStatefulSetControlInterface {
	return &defaultBoxStatefulSetControl{boxControl, statusUpdater, controllerHistory, recorder}
}

type defaultBoxStatefulSetControl struct {
	boxControl        *StatefulBoxControl
	statusUpdater     BoxStatefulSetStatusUpdaterInterface
	controllerHistory history.Interface
	recorder          record.EventRecorder
}

func (ssc *defaultBoxStatefulSetControl) AdoptOrphanRevisions(boxSts *boxstatesetv1alpha1.BoxStatefulSet, revisions []*apps.ControllerRevision) error {
	for i := range revisions {
		adopted, err := ssc.controllerHistory.AdoptControllerRevision(boxSts, boxstatesetv1alpha1.BoxStatefulSetGroupVersionKind, revisions[i])
		if err != nil {
			return err
		}
		revisions[i] = adopted
	}
	return nil
}

func (ssc *defaultBoxStatefulSetControl) UpdateBoxStatefulSet(ctx context.Context, boxSts *boxstatesetv1alpha1.BoxStatefulSet, boxes []*boxv1alpha1.Box) (*boxstatesetv1alpha1.BoxStatefulSetStatus, error) {
	boxSts = boxSts.DeepCopy()

	// list all revisions and sort them
	revisions, err := ssc.ListRevisions(boxSts)
	if err != nil {
		klog.Error(err, "ListRevisions failed")
		return nil, err
	}
	history.SortControllerRevisions(revisions)

	currentRevision, updateRevision, status, err := ssc.performUpdate(ctx, boxSts, boxes, revisions)
	if err != nil {
		klog.Error(err, "performUpdate failed")
		errs := []error{err}
		if agg, ok := err.(utilerrors.Aggregate); ok {
			errs = agg.Errors()
		}

		if err := ssc.truncateHistory(boxSts, boxes, revisions, currentRevision, updateRevision); err != nil {
			klog.Error(err, "truncateHistory failed")
			errs = append(errs, err)
		}
		return nil, utilerrors.NewAggregate(errs)
	}
	if err := ssc.truncateHistory(boxSts, boxes, revisions, currentRevision, updateRevision); err != nil {
		klog.Error(err, "truncateHistory failed")
		return status, err
	}

	// maintain the set's revision history limit
	return status, nil
}

func (ssc *defaultBoxStatefulSetControl) ListRevisions(boxSts *boxstatesetv1alpha1.BoxStatefulSet) ([]*apps.ControllerRevision, error) {
	selector, err := metav1.LabelSelectorAsSelector(boxSts.Spec.Selector)
	if err != nil {
		return nil, err
	}
	return ssc.controllerHistory.ListControllerRevisions(boxSts, selector)
}

func (ssc *defaultBoxStatefulSetControl) performUpdate(
	ctx context.Context, boxSts *boxstatesetv1alpha1.BoxStatefulSet, boxes []*boxv1alpha1.Box, revisions []*apps.ControllerRevision) (*apps.ControllerRevision, *apps.ControllerRevision, *boxstatesetv1alpha1.BoxStatefulSetStatus, error) {
	var currentStatus *boxstatesetv1alpha1.BoxStatefulSetStatus
	logger := klog.FromContext(ctx)
	// get the current, and update revisions
	currentRevision, updateRevision, collisionCount, err := ssc.getStatefulSetRevisions(boxSts, revisions)
	if err != nil {
		klog.Error(err, "getStatefulSetRevisions failed")
		return currentRevision, updateRevision, currentStatus, err
	}

	// perform the main update function and get the status
	currentStatus, err = ssc.updateStatefulSet(ctx, boxSts, currentRevision, updateRevision, collisionCount, boxes)
	if err != nil {
		klog.Error(err, "updateStatefulSet failed")
	}
	if err != nil && currentStatus == nil {
		return currentRevision, updateRevision, nil, err
	}

	// make sure to update the latest status even if there is an error with non-nil currentStatus
	statusErr := ssc.updateBoxStatefulSetStatus(ctx, boxSts, currentStatus)
	if statusErr == nil {
		logger.V(4).Info("Updated status", "statefulSet", klog.KObj(boxSts),
			"replicas", currentStatus.Replicas,
			"readyReplicas", currentStatus.ReadyReplicas,
			"currentReplicas", currentStatus.CurrentReplicas,
			"updatedReplicas", currentStatus.UpdatedReplicas)
	} else {
		klog.Error(statusErr, "updateBoxStatefulSetStatus failed")
	}

	switch {
	case err != nil && statusErr != nil:
		logger.Error(statusErr, "Could not update status", "BoxStatefulSet", klog.KObj(boxSts))
		return currentRevision, updateRevision, currentStatus, err
	case err != nil:
		return currentRevision, updateRevision, currentStatus, err
	case statusErr != nil:
		return currentRevision, updateRevision, currentStatus, statusErr
	}

	logger.V(4).Info("BoxStatefulSet revisions", "BoxStatefulSet", klog.KObj(boxSts),
		"currentRevision", currentStatus.CurrentRevision,
		"updateRevision", currentStatus.UpdateRevision)

	return currentRevision, updateRevision, currentStatus, nil
}

// updateStatefulSet performs the update function for a StatefulSet. This method creates, updates, and deletes Pods in
// the set in order to conform the system to the target state for the set. The target state always contains
// set.Spec.Replicas Pods with a Ready Condition. If the UpdateStrategy.Type for the set is
// RollingUpdateStatefulSetStrategyType then all Pods in the set must be at set.Status.CurrentRevision.
// If the UpdateStrategy.Type for the set is OnDeleteStatefulSetStrategyType, the target state implies nothing about
// the revisions of Pods in the set. If the UpdateStrategy.Type for the set is PartitionStatefulSetStrategyType, then
// all Pods with ordinal less than UpdateStrategy.Partition.Ordinal must be at Status.CurrentRevision and all other
// Pods must be at Status.UpdateRevision. If the returned error is nil, the returned StatefulSetStatus is valid and the
// update must be recorded. If the error is not nil, the method should be retried until successful.
func (ssc *defaultBoxStatefulSetControl) updateStatefulSet(
	ctx context.Context,
	boxSts *boxstatesetv1alpha1.BoxStatefulSet,
	currentRevision *apps.ControllerRevision,
	updateRevision *apps.ControllerRevision,
	collisionCount int32,
	boxes []*boxv1alpha1.Box) (*boxstatesetv1alpha1.BoxStatefulSetStatus, error) {
	logger := klog.FromContext(ctx)
	// get the current and update revisions of the set.
	currentSet, err := ApplyRevision(boxSts, currentRevision)
	if err != nil {
		klog.Error(err, "apply current revision failed")
		return nil, err
	}
	updateSet, err := ApplyRevision(boxSts, updateRevision)
	if err != nil {
		klog.Error(err, "apply update revision failed")
		return nil, err
	}

	// set the generation, and revisions in the returned status
	status := boxstatesetv1alpha1.BoxStatefulSetStatus{}
	status.ObservedGeneration = boxSts.Generation
	status.CurrentRevision = currentRevision.Name
	status.UpdateRevision = updateRevision.Name
	status.CollisionCount = new(int32)
	*status.CollisionCount = collisionCount

	updateStatus(&status, boxSts.Spec.MinReadySeconds, currentRevision, updateRevision, boxes)

	replicaCount := int(*boxSts.Spec.Replicas)
	// slice that will contain all Pods such that getStartOrdinal(set) <= getOrdinal(pod) <= getEndOrdinal(set)
	replicas := make([]*boxv1alpha1.Box, replicaCount)
	// slice that will contain all Pods such that getOrdinal(pod) < getStartOrdinal(set) OR getOrdinal(pod) > getEndOrdinal(set)
	condemned := make([]*boxv1alpha1.Box, 0, len(boxes))
	unhealthy := 0
	var firstUnhealthyBox *boxv1alpha1.Box

	// First we partition pods into two lists valid replicas and condemned Pods
	for _, box := range boxes {
		if boxInOrdinalRange(box, boxSts) {
			// if the ordinal of the pod is within the range of the current number of replicas,
			// insert it at the indirection of its ordinal
			replicas[getOrdinal(box)-getStartOrdinal(boxSts)] = box
		} else if getOrdinal(box) >= 0 {
			// if the ordinal is valid, but not within the range add it to the condemned list
			condemned = append(condemned, box)
		}
		// If the ordinal could not be parsed (ord < 0), ignore the Box.
	}

	// for any empty indices in the sequence [0,set.Spec.Replicas) create a new Pod at the correct revision
	for ord := getStartOrdinal(boxSts); ord <= getEndOrdinal(boxSts); ord++ {
		replicaIdx := ord - getStartOrdinal(boxSts)
		if replicas[replicaIdx] == nil {
			replicas[replicaIdx] = newVersionedBoxStatefulSet(
				currentSet,
				updateSet,
				currentRevision.Name,
				updateRevision.Name, ord)
		}
	}

	// sort the condemned Pods by their ordinals
	sort.Sort(descendingOrdinal(condemned))

	// find the first unhealthy Pod
	for i := range replicas {
		if !isHealthy(replicas[i]) {
			unhealthy++
			if firstUnhealthyBox == nil {
				firstUnhealthyBox = replicas[i]
			}
		}
	}

	// or the first unhealthy condemned Pod (condemned are sorted in descending order for ease of use)
	for i := len(condemned) - 1; i >= 0; i-- {
		if !isHealthy(condemned[i]) {
			unhealthy++
			if firstUnhealthyBox == nil {
				firstUnhealthyBox = condemned[i]
			}
		}
	}

	if unhealthy > 0 {
		logger.V(4).Info("StatefulSet has unhealthy Pods", "BoxStatefulSet", klog.KObj(boxSts), "unhealthyReplicas", unhealthy, "pod", klog.KObj(firstUnhealthyBox))
	}

	// If the StatefulSet is being deleted, don't do anything other than updating
	// status.
	if boxSts.DeletionTimestamp != nil {
		return &status, nil
	}

	monotonic := !(boxSts.Spec.PodManagementPolicy == apps.ParallelPodManagement)

	// First, process each living replica. Exit if we run into an error or something blocking in monotonic mode.
	processReplicaFn := func(i int) (bool, error) {
		return ssc.processReplica(ctx, boxSts, currentRevision, updateRevision, currentSet, updateSet, monotonic, replicas, i)
	}
	if shouldExit, err := runForAll(replicas, processReplicaFn, monotonic); shouldExit || err != nil {
		updateStatus(&status, boxSts.Spec.MinReadySeconds, currentRevision, updateRevision, replicas, condemned)
		return &status, err
	}

	// Fix pod claims for condemned pods, if necessary.
	//if utilfeature.DefaultFeatureGate.Enabled(features.StatefulSetAutoDeletePVC) {
	//	fixPodClaim := func(i int) (bool, error) {
	//		if matchPolicy, err := ssc.podControl.ClaimsMatchRetentionPolicy(ctx, updateSet, condemned[i]); err != nil {
	//			return true, err
	//		} else if !matchPolicy {
	//			if err := ssc.podControl.UpdatePodClaimForRetentionPolicy(ctx, updateSet, condemned[i]); err != nil {
	//				return true, err
	//			}
	//		}
	//		return false, nil
	//	}
	//	if shouldExit, err := runForAll(condemned, fixPodClaim, monotonic); shouldExit || err != nil {
	//		updateStatus(&status, set.Spec.MinReadySeconds, currentRevision, updateRevision, replicas, condemned)
	//		return &status, err
	//	}
	//}

	// At this point, in monotonic mode all of the current Replicas are Running, Ready and Available,
	// and we can consider termination.
	// We will wait for all predecessors to be Running and Ready prior to attempting a deletion.
	// We will terminate Pods in a monotonically decreasing order.
	// Note that we do not resurrect Pods in this interval. Also note that scaling will take precedence over
	// updates.
	processCondemnedFn := func(i int) (bool, error) {
		return ssc.processCondemned(ctx, boxSts, firstUnhealthyBox, monotonic, condemned, i)
	}
	if shouldExit, err := runForAll(condemned, processCondemnedFn, monotonic); shouldExit || err != nil {
		updateStatus(&status, boxSts.Spec.MinReadySeconds, currentRevision, updateRevision, replicas, condemned)
		return &status, err
	}

	updateStatus(&status, boxSts.Spec.MinReadySeconds, currentRevision, updateRevision, replicas, condemned)

	// for the OnDelete strategy we short circuit. Pods will be updated when they are manually deleted.
	if boxSts.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType {
		return &status, nil
	}

	//if utilfeature.DefaultFeatureGate.Enabled(features.MaxUnavailableStatefulSet) {
	//	return updateStatefulSetAfterInvariantEstablished(ctx,
	//		ssc,
	//		set,
	//		replicas,
	//		updateRevision,
	//		status,
	//	)
	//}

	// we compute the minimum ordinal of the target sequence for a destructive update based on the strategy.
	updateMin := 0
	if boxSts.Spec.UpdateStrategy.RollingUpdate != nil {
		updateMin = int(*boxSts.Spec.UpdateStrategy.RollingUpdate.Partition)
	}
	// we terminate the Pod with the largest ordinal that does not match the update revision.
	for target := len(replicas) - 1; target >= updateMin; target-- {

		// delete the Pod if it is not already terminating and does not match the update revision.
		if getBoxRevision(replicas[target]) != updateRevision.Name && !isTerminating(replicas[target]) {
			logger.V(2).Info("Box of BoxStatefulSet is terminating for update",
				"BoxStatefulSet", klog.KObj(boxSts), "Box", klog.KObj(replicas[target]))
			if err := ssc.boxControl.DeleteBoxByBoxStateful(ctx, boxSts, replicas[target]); err != nil {
				if !errors.IsNotFound(err) {
					klog.Error(err, "DeleteBoxByBoxStateful failed")
					return &status, err
				}
			}
			status.CurrentReplicas--
			return &status, err
		}

		// wait for unhealthy Pods on update
		if !isHealthy(replicas[target]) {
			logger.V(4).Info("BoxStatefulSet is waiting for Box to update",
				"BoxStatefulSet", klog.KObj(boxSts), "Box", klog.KObj(replicas[target]))
			return &status, nil
		}

	}
	return &status, nil
}

func runForAll(boxes []*boxv1alpha1.Box, fn func(i int) (bool, error), monotonic bool) (bool, error) {
	if monotonic {
		for i := range boxes {
			if shouldExit, err := fn(i); shouldExit || err != nil {
				klog.Error(err, "run func failed")
				return true, err
			}
		}
	} else {
		if _, err := slowStartBatch(1, len(boxes), fn); err != nil {
			klog.Error(err, "slowStartBatch failed")
			return true, err
		}
	}
	return false, nil
}

func slowStartBatch(initialBatchSize int, remaining int, fn func(int) (bool, error)) (int, error) {
	successes := 0
	j := 0
	for batchSize := integer.IntMin(remaining, initialBatchSize); batchSize > 0; batchSize = integer.IntMin(integer.IntMin(2*batchSize, remaining), MaxBatchSize) {
		errCh := make(chan error, batchSize)
		var wg sync.WaitGroup
		wg.Add(batchSize)
		for i := 0; i < batchSize; i++ {
			go func(k int) {
				defer wg.Done()
				// Ignore the first parameter - relevant for monotonic only.
				if _, err := fn(k); err != nil {
					errCh <- err
				}
			}(j)
			j++
		}
		wg.Wait()
		successes += batchSize - len(errCh)
		close(errCh)
		if len(errCh) > 0 {
			errs := make([]error, 0)
			for err := range errCh {
				errs = append(errs, err)
			}
			return successes, utilerrors.NewAggregate(errs)
		}
		remaining -= batchSize
	}
	return successes, nil
}

type replicaStatus struct {
	replicas          int32
	readyReplicas     int32
	availableReplicas int32
	currentReplicas   int32
	updatedReplicas   int32
}

func computeReplicaStatus(boxes []*boxv1alpha1.Box, minReadySeconds int32, currentRevision, updateRevision *apps.ControllerRevision) replicaStatus {
	status := replicaStatus{}
	for _, box := range boxes {
		if isCreated(box) {
			status.replicas++
		}

		// count the number of running and ready replicas
		if isRunningAndReady(box) {
			status.readyReplicas++
			// count the number of running and available replicas
			if isRunningAndAvailable(box, minReadySeconds) {
				status.availableReplicas++
			}

		}

		// count the number of current and update replicas
		if isCreated(box) && !isTerminating(box) {
			if getBoxRevision(box) == currentRevision.Name {
				status.currentReplicas++
			}
			if getBoxRevision(box) == updateRevision.Name {
				status.updatedReplicas++
			}
		}
	}
	return status
}

func updateStatus(status *boxstatesetv1alpha1.BoxStatefulSetStatus, minReadySeconds int32, currentRevision, updateRevision *apps.ControllerRevision, boxLists ...[]*boxv1alpha1.Box) {
	status.Replicas = 0
	status.ReadyReplicas = 0
	status.AvailableReplicas = 0
	status.CurrentReplicas = 0
	status.UpdatedReplicas = 0
	for _, list := range boxLists {
		replicaStatus := computeReplicaStatus(list, minReadySeconds, currentRevision, updateRevision)
		status.Replicas += replicaStatus.replicas
		status.ReadyReplicas += replicaStatus.readyReplicas
		status.AvailableReplicas += replicaStatus.availableReplicas
		status.CurrentReplicas += replicaStatus.currentReplicas
		status.UpdatedReplicas += replicaStatus.updatedReplicas
	}
}

func (ssc *defaultBoxStatefulSetControl) getStatefulSetRevisions(
	boxSts *boxstatesetv1alpha1.BoxStatefulSet,
	revisions []*apps.ControllerRevision) (*apps.ControllerRevision, *apps.ControllerRevision, int32, error) {
	var currentRevision, updateRevision *apps.ControllerRevision

	revisionCount := len(revisions)
	history.SortControllerRevisions(revisions)

	// Use a local copy of set.Status.CollisionCount to avoid modifying set.Status directly.
	// This copy is returned so the value gets carried over to set.Status in updateStatefulSet.
	var collisionCount int32
	if boxSts.Status.CollisionCount != nil {
		collisionCount = *boxSts.Status.CollisionCount
	}

	// create a new revision from the current set
	updateRevision, err := newRevision(boxSts, nextRevision(revisions), &collisionCount)
	if err != nil {

		return nil, nil, collisionCount, err
	}

	// find any equivalent revisions
	equalRevisions := history.FindEqualRevisions(revisions, updateRevision)
	equalCount := len(equalRevisions)

	if equalCount > 0 && history.EqualRevision(revisions[revisionCount-1], equalRevisions[equalCount-1]) {
		// if the equivalent revision is immediately prior the update revision has not changed
		updateRevision = revisions[revisionCount-1]
	} else if equalCount > 0 {
		// if the equivalent revision is not immediately prior we will roll back by incrementing the
		// Revision of the equivalent revision
		updateRevision, err = ssc.controllerHistory.UpdateControllerRevision(
			equalRevisions[equalCount-1],
			updateRevision.Revision)
		if err != nil {
			return nil, nil, collisionCount, err
		}
	} else {
		//if there is no equivalent revision we create a new one
		updateRevision, err = ssc.controllerHistory.CreateControllerRevision(boxSts, updateRevision, &collisionCount)
		if err != nil {
			return nil, nil, collisionCount, err
		}
	}

	// attempt to find the revision that corresponds to the current revision
	for i := range revisions {
		if revisions[i].Name == boxSts.Status.CurrentRevision {
			currentRevision = revisions[i]
			break
		}
	}

	// if the current revision is nil we initialize the history by setting it to the update revision
	if currentRevision == nil {
		currentRevision = updateRevision
	}

	return currentRevision, updateRevision, collisionCount, nil
}

func newRevision(boxSts *boxstatesetv1alpha1.BoxStatefulSet, revision int64, collisionCount *int32) (*apps.ControllerRevision, error) {
	patch, err := getPatch(boxSts)
	if err != nil {
		return nil, err
	}
	cr, err := history.NewControllerRevision(boxSts,
		boxstatesetv1alpha1.BoxStatefulSetGroupVersionKind,
		boxSts.Spec.Template.Labels,
		runtime.RawExtension{Raw: patch},
		revision,
		collisionCount)
	if err != nil {
		return nil, err
	}
	if cr.ObjectMeta.Annotations == nil {
		cr.ObjectMeta.Annotations = make(map[string]string)
	}
	for key, value := range boxSts.Annotations {
		cr.ObjectMeta.Annotations[key] = value
	}
	return cr, nil
}

var patchCodec = scheme.Codecs.LegacyCodec(boxstatesetv1alpha1.SchemeGroupVersion)

// getPatch returns a strategic merge patch that can be applied to restore a StatefulSet to a
// previous version. If the returned error is nil the patch is valid. The current state that we save is just the
// PodSpecTemplate. We can modify this later to encompass more state (or less) and remain compatible with previously
// recorded patches.
func getPatch(boxSts *boxstatesetv1alpha1.BoxStatefulSet) ([]byte, error) {
	data, err := runtime.Encode(patchCodec, boxSts)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	if err != nil {
		return nil, err
	}
	objCopy := make(map[string]interface{})
	specCopy := make(map[string]interface{})
	spec := raw["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	specCopy["template"] = template
	template["$patch"] = "replace"
	objCopy["spec"] = specCopy
	patch, err := json.Marshal(objCopy)
	return patch, err
}

func (ssc *defaultBoxStatefulSetControl) processReplica(
	ctx context.Context,
	boxSts *boxstatesetv1alpha1.BoxStatefulSet,
	currentRevision *apps.ControllerRevision,
	updateRevision *apps.ControllerRevision,
	currentBoxSts *boxstatesetv1alpha1.BoxStatefulSet,
	updateBoxSts *boxstatesetv1alpha1.BoxStatefulSet,
	monotonic bool,
	replicas []*boxv1alpha1.Box,
	i int) (bool, error) {
	logger := klog.FromContext(ctx)
	// delete and recreate failed pods
	if isFailed(replicas[i]) {
		ssc.recorder.Eventf(boxSts, v1.EventTypeWarning, "RecreatingFailedPod",
			"BoxStatefulSet %s/%s is recreating failed Bod %s",
			boxSts.Namespace,
			boxSts.Name,
			replicas[i].Name)
		if err := ssc.boxControl.DeleteBoxByBoxStateful(ctx, boxSts, replicas[i]); err != nil {
			logger.Error(err, "DeleteBoxByBoxStateful failed")
			return true, err
		}
		replicaOrd := i + getStartOrdinal(boxSts)
		replicas[i] = newVersionedBoxStatefulSet(
			currentBoxSts,
			updateBoxSts,
			currentRevision.Name,
			updateRevision.Name,
			replicaOrd)
	}
	// If we find a Pod that has not been created we create the Pod
	if !isCreated(replicas[i]) {
		//if utilfeature.DefaultFeatureGate.Enabled(features.StatefulSetAutoDeletePVC) {
		//	if isStale, err := ssc.podControl.PodClaimIsStale(boxSts, replicas[i]); err != nil {
		//		return true, err
		//	} else if isStale {
		//		// If a pod has a stale PVC, no more work can be done this round.
		//		return true, err
		//	}
		//}
		if err := ssc.boxControl.CreateBoxByBoxStatefulSet(ctx, boxSts, replicas[i]); err != nil {
			logger.Error(err, "CreateBoxByBoxStatefulSet failed")
			return true, err
		}
		if monotonic {
			// if the set does not allow bursting, return immediately
			return true, nil
		}
	}

	// If the Pod is in pending state then trigger PVC creation to create missing PVCs
	if isPending(replicas[i]) {
		logger.V(4).Info(
			"BoxStatefulSet is triggering PVC creation for pending Box",
			"BoxStatefulSet", klog.KObj(boxSts), "Box", klog.KObj(replicas[i]))
		if err := ssc.boxControl.createMissingPersistentVolumeClaims(ctx, boxSts, replicas[i]); err != nil {
			logger.Error(err, "createMissingPersistentVolumeClaims failed")
			return true, err
		}
	}

	// If we find a Pod that is currently terminating, we must wait until graceful deletion
	// completes before we continue to make progress.
	if isTerminating(replicas[i]) && monotonic {
		logger.V(4).Info("BoxStatefulSet is waiting for Box to Terminate",
			"BoxStatefulSet", klog.KObj(boxSts), "Box", klog.KObj(replicas[i]))
		return true, nil
	}

	// If we have a Pod that has been created but is not running and ready we can not make progress.
	// We must ensure that all for each Pod, when we create it, all of its predecessors, with respect to its
	// ordinal, are Running and Ready.
	if !isRunningAndReady(replicas[i]) && monotonic {
		logger.V(4).Info("BoxStatefulSet is waiting for Box to be Running and Ready",
			"BoxStatefulSet", klog.KObj(boxSts), "Box", klog.KObj(replicas[i]))
		return true, nil
	}

	// If we have a Pod that has been created but is not available we can not make progress.
	// We must ensure that all for each Pod, when we create it, all of its predecessors, with respect to its
	// ordinal, are Available.
	if !isRunningAndAvailable(replicas[i], boxSts.Spec.MinReadySeconds) && monotonic {
		logger.V(4).Info("StatefulSet is waiting for Pod to be Available",
			"statefulSet", klog.KObj(boxSts), "pod", klog.KObj(replicas[i]))
		return true, nil
	}

	// Enforce the StatefulSet invariants
	retentionMatch := true
	//if utilfeature.DefaultFeatureGate.Enabled(features.StatefulSetAutoDeletePVC) {
	//	var err error
	//	retentionMatch, err = ssc.podControl.ClaimsMatchRetentionPolicy(ctx, updateSet, replicas[i])
	//	// An error is expected if the pod is not yet fully updated, and so return is treated as matching.
	//	if err != nil {
	//		retentionMatch = true
	//	}
	//}

	if identityMatches(boxSts, replicas[i]) && storageMatches(boxSts, replicas[i]) && retentionMatch {
		return false, nil
	}

	// Make a deep copy so we don't mutate the shared cache
	replica := replicas[i].DeepCopy()
	if err := ssc.boxControl.UpdateBoxByBoxStatefulSet(ctx, updateBoxSts, replica); err != nil {
		logger.Error(err, "UpdateBoxByBoxStatefulSet failed")
		return true, err
	}

	return false, nil
}

// updateBoxStatefulSetStatus updates set's Status to be equal to status. If status indicates a complete update, it is
// mutated to indicate completion. If status is semantically equivalent to set's Status no update is performed. If the
// returned error is nil, the update is successful.
func (ssc *defaultBoxStatefulSetControl) updateBoxStatefulSetStatus(
	ctx context.Context,
	boxSts *boxstatesetv1alpha1.BoxStatefulSet,
	status *boxstatesetv1alpha1.BoxStatefulSetStatus) error {
	klog.V(4).Infof("start update BoxStatefulSet(%s/%s) status, %v", boxSts.Namespace, boxSts.Name, klog.KObj(boxSts))
	// complete any in progress rolling update if necessary
	completeRollingUpdate(boxSts, status)

	// if the status is not inconsistent do not perform an update
	if !inconsistentStatus(boxSts, status) {
		return nil
	}

	// copy set and update its status
	boxSts = boxSts.DeepCopy()
	if err := ssc.statusUpdater.UpdateBoxStatefulSetStatus(ctx, boxSts, status); err != nil {
		klog.Error("UpdateBoxStatefulSetStatus failed, err:", err)
		return err
	}

	return nil
}

// truncateHistory truncates any non-live ControllerRevisions in revisions from set's history. The UpdateRevision and
// CurrentRevision in set's Status are considered to be live. Any revisions associated with the Pods in pods are also
// considered to be live. Non-live revisions are deleted, starting with the revision with the lowest Revision, until
// only RevisionHistoryLimit revisions remain. If the returned error is nil the operation was successful. This method
// expects that revisions is sorted when supplied.
func (ssc *defaultBoxStatefulSetControl) truncateHistory(
	boxSts *boxstatesetv1alpha1.BoxStatefulSet,
	boxes []*boxv1alpha1.Box,
	revisions []*apps.ControllerRevision,
	current *apps.ControllerRevision,
	update *apps.ControllerRevision) error {
	history := make([]*apps.ControllerRevision, 0, len(revisions))
	// mark all live revisions
	live := map[string]bool{}
	if current != nil {
		live[current.Name] = true
	}
	if update != nil {
		live[update.Name] = true
	}
	for i := range boxes {
		live[getBoxRevision(boxes[i])] = true
	}
	// collect live revisions and historic revisions
	for i := range revisions {
		if !live[revisions[i].Name] {
			history = append(history, revisions[i])
		}
	}
	historyLen := len(history)
	historyLimit := 10
	if boxSts.Spec.RevisionHistoryLimit != nil {
		historyLimit = int(*boxSts.Spec.RevisionHistoryLimit)
	}
	if historyLen <= historyLimit {
		return nil
	}
	// delete any non-live history to maintain the revision limit.
	history = history[:(historyLen - historyLimit)]
	for i := 0; i < len(history); i++ {
		if err := ssc.controllerHistory.DeleteControllerRevision(history[i]); err != nil {
			return err
		}
	}
	return nil
}

func (ssc *defaultBoxStatefulSetControl) processCondemned(ctx context.Context, boxSts *boxstatesetv1alpha1.BoxStatefulSet, firstUnhealthyBox *boxv1alpha1.Box, monotonic bool, condemned []*boxv1alpha1.Box, i int) (bool, error) {
	logger := klog.FromContext(ctx)
	if isTerminating(condemned[i]) {
		// if we are in monotonic mode, block and wait for terminating pods to expire
		if monotonic {
			logger.V(4).Info("BoxStatefulSet is waiting for Box to Terminate prior to scale down",
				"BoxstatefulSet", klog.KObj(boxSts), "Box", klog.KObj(condemned[i]))
			return true, nil
		}
		return false, nil
	}
	// if we are in monotonic mode and the condemned target is not the first unhealthy Pod block
	if !isRunningAndReady(condemned[i]) && monotonic && condemned[i] != firstUnhealthyBox {
		logger.V(4).Info("BoxStatefulSet is waiting for Box to be Running and Ready prior to scale down",
			"BoxstatefulSet", klog.KObj(boxSts), "Box", klog.KObj(firstUnhealthyBox))
		return true, nil
	}
	// if we are in monotonic mode and the condemned target is not the first unhealthy Pod, block.
	if !isRunningAndAvailable(condemned[i], boxSts.Spec.MinReadySeconds) && monotonic && condemned[i] != firstUnhealthyBox {
		logger.V(4).Info("BoxStatefulSet is waiting for Box to be Available prior to scale down",
			"BoxstatefulSet", klog.KObj(boxSts), "Box", klog.KObj(firstUnhealthyBox))
		return true, nil
	}

	logger.V(2).Info("Box of BoxStatefulSet is terminating for scale down",
		"BoxStatefulSet", klog.KObj(boxSts), "Box", klog.KObj(condemned[i]))
	if err := ssc.boxControl.DeleteBoxByBoxStateful(ctx, boxSts, condemned[i]); err != nil {
		logger.Error(err, "DeleteBoxByBoxStateful failed")
		return true, err
	}
	return true, nil
}
