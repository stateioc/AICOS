package box_statefulset

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	boxstatefulsetv1alpha1 "cncos.io/box-controller/pkg/apis/boxstatefulset/v1alpha1"
	boxclientset "cncos.io/box-controller/pkg/box-generated/clientset/versioned"
	boxlisters "cncos.io/box-controller/pkg/box-generated/listers/box/v1alpha1"
)

type StatefulBoxControlObjectManager interface {
	CreateBox(ctx context.Context, box *boxv1alpha1.Box) error
	GetBox(ctx context.Context, namespace, boxName string) (*boxv1alpha1.Box, error)
	UpdateBox(ctx context.Context, box *boxv1alpha1.Box) error
	DeleteBox(ctx context.Context, box *boxv1alpha1.Box) error
	CreateClaim(ctx context.Context, claim *corev1.PersistentVolumeClaim) error
	GetClaim(ctx context.Context, namespace, claimName string) (*corev1.PersistentVolumeClaim, error)
	UpdateClaim(ctx context.Context, claim *corev1.PersistentVolumeClaim) error
}

type StatefulBoxControl struct {
	objectMgr StatefulBoxControlObjectManager
	recorder  record.EventRecorder
}

func NewStatefulBoxControl(
	client clientset.Interface,
	boxClientSet boxclientset.Interface,
	boxLister boxlisters.BoxLister,
	claimLister corelisters.PersistentVolumeClaimLister,
	recorder record.EventRecorder,
) *StatefulBoxControl {
	return &StatefulBoxControl{&realStatefulBoxControlObjectManager{client, boxClientSet, boxLister, claimLister}, recorder}
}

// realStatefulBoxControlObjectManager uses a clientset.Interface and listers.
type realStatefulBoxControlObjectManager struct {
	client       clientset.Interface
	boxClientSet boxclientset.Interface
	boxLister    boxlisters.BoxLister
	claimLister  corelisters.PersistentVolumeClaimLister
}

func (om *realStatefulBoxControlObjectManager) CreateBox(ctx context.Context, box *boxv1alpha1.Box) error {
	_, err := om.boxClientSet.CncosV1alpha1().Boxes(box.Namespace).Create(ctx, box, metav1.CreateOptions{})
	return err
}
func (om *realStatefulBoxControlObjectManager) GetBox(ctx context.Context, namespace, boxName string) (*boxv1alpha1.Box, error) {
	return om.boxClientSet.CncosV1alpha1().Boxes(namespace).Get(ctx, boxName, metav1.GetOptions{})
}
func (om *realStatefulBoxControlObjectManager) UpdateBox(ctx context.Context, box *boxv1alpha1.Box) error {
	_, err := om.boxClientSet.CncosV1alpha1().Boxes(box.Namespace).Update(ctx, box, metav1.UpdateOptions{})
	return err
}
func (om *realStatefulBoxControlObjectManager) DeleteBox(ctx context.Context, box *boxv1alpha1.Box) error {
	return om.boxClientSet.CncosV1alpha1().Boxes(box.Namespace).Delete(ctx, box.Name, metav1.DeleteOptions{})
}
func (om *realStatefulBoxControlObjectManager) CreateClaim(ctx context.Context, claim *corev1.PersistentVolumeClaim) error {
	_, err := om.client.CoreV1().PersistentVolumeClaims(claim.Namespace).Create(ctx, claim, metav1.CreateOptions{})
	return err
}
func (om *realStatefulBoxControlObjectManager) GetClaim(ctx context.Context, namespace, claimName string) (*corev1.PersistentVolumeClaim, error) {
	return om.client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, claimName, metav1.GetOptions{})
}
func (om *realStatefulBoxControlObjectManager) UpdateClaim(ctx context.Context, claim *corev1.PersistentVolumeClaim) error {
	_, err := om.client.CoreV1().PersistentVolumeClaims(claim.Namespace).Update(ctx, claim, metav1.UpdateOptions{})
	return err
}

func (sbc *StatefulBoxControl) DeleteBoxByBoxStateful(ctx context.Context, boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) error {
	err := sbc.objectMgr.DeleteBox(ctx, box)
	sbc.recordBoxEvent("delete", boxSts, box, err)
	return err
}

// recordBoxEvent records an event for verb applied to a box in a BoxStatefulSet. If err is nil the generated event will
// have a reason of corev1.EventTypeNormal. If err is not nil the generated event will have a reason of corev1.EventTypeWarning.
func (sbc *StatefulBoxControl) recordBoxEvent(verb string, boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box, err error) {
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		message := fmt.Sprintf("%s Box %s in BoxStatefulSet %s successful",
			strings.ToLower(verb), box.Name, boxSts.Name)
		sbc.recorder.Event(boxSts, corev1.EventTypeNormal, reason, message)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		message := fmt.Sprintf("%s Box %s in BoxStatefulSet %s failed error: %s",
			strings.ToLower(verb), box.Name, boxSts.Name, err)
		sbc.recorder.Event(boxSts, corev1.EventTypeWarning, reason, message)
	}
}

// recordClaimEvent records an event for verb applied to the PersistentVolumeClaim of a Box in a BoxStatefulSet. If err is
// nil the generated event will have a reason of corev1.EventTypeNormal. If err is not nil the generated event will have a
// reason of corev1.EventTypeWarning.
func (sbc *StatefulBoxControl) recordClaimEvent(verb string, boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box, claim *corev1.PersistentVolumeClaim, err error) {
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		message := fmt.Sprintf("%s Claim %s Box %s in BoxStatefulSet %s success",
			strings.ToLower(verb), claim.Name, box.Name, boxSts.Name)
		sbc.recorder.Event(boxSts, corev1.EventTypeNormal, reason, message)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		message := fmt.Sprintf("%s Claim %s for Box %s in BoxStatefulSet %s failed error: %s",
			strings.ToLower(verb), claim.Name, box.Name, boxSts.Name, err)
		sbc.recorder.Event(boxSts, corev1.EventTypeWarning, reason, message)
	}
}

func (sbc *StatefulBoxControl) CreateBoxByBoxStatefulSet(ctx context.Context, boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) error {
	// Create the Pod's PVCs prior to creating the Pod
	if err := sbc.createPersistentVolumeClaims(ctx, boxSts, box); err != nil {
		sbc.recordBoxEvent("create", boxSts, box, err)
		return err
	}
	// If we created the PVCs attempt to create the Box
	err := sbc.objectMgr.CreateBox(ctx, box)
	// sink already exists errors
	if apierrors.IsAlreadyExists(err) {
		return err
	}
	//if utilfeature.DefaultFeatureGate.Enabled(features.StatefulSetAutoDeletePVC) {
	//	// Set PVC policy as much as is possible at this point.
	//	if err := sbc.UpdatePodClaimForRetentionPolicy(ctx, set, pod); err != nil {
	//		sbc.recordPodEvent("update", set, pod, err)
	//		return err
	//	}
	//}
	sbc.recordBoxEvent("create", boxSts, box, err)
	return err
}

// createPersistentVolumeClaims creates all of the required PersistentVolumeClaims for box, which must be a member of
// BoxStatefulSet. If all of the claims for Box are successfully created, the returned error is nil. If creation fails, this method
// may be called again until no error is returned, indicating the PersistentVolumeClaims for Box are consistent with
// BoxStatefulSet's Spec.
func (sbc *StatefulBoxControl) createPersistentVolumeClaims(ctx context.Context, boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) error {
	var errs []error
	for _, claim := range getPersistentVolumeClaims(boxSts, getOrdinal(box)) {
		pvc, err := sbc.objectMgr.GetClaim(ctx, claim.Namespace, claim.Name)
		switch {
		case apierrors.IsNotFound(err):
			err := sbc.objectMgr.CreateClaim(ctx, &claim)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to create PVC %s: %s", claim.Name, err))
			}
			if err == nil || !apierrors.IsAlreadyExists(err) {
				sbc.recordClaimEvent("create", boxSts, box, &claim, err)
			}
		case err != nil:
			errs = append(errs, fmt.Errorf("failed to retrieve PVC %s: %s", claim.Name, err))
			sbc.recordClaimEvent("create", boxSts, box, &claim, err)
		default:
			if pvc.DeletionTimestamp != nil {
				errs = append(errs, fmt.Errorf("pvc %s is being deleted", claim.Name))
			}
		}
		// TODO: Check resource requirements and accessmodes, update if necessary
	}
	return errorutils.NewAggregate(errs)
}

// createMissingPersistentVolumeClaims creates all of the required PersistentVolumeClaims for box, and updates its retention policy
func (sbc *StatefulBoxControl) createMissingPersistentVolumeClaims(ctx context.Context, boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) error {
	if err := sbc.createPersistentVolumeClaims(ctx, boxSts, box); err != nil {
		return err
	}

	//if utilfeature.DefaultFeatureGate.Enabled(features.StatefulSetAutoDeletePVC) {
	//	// Set PVC policy as much as is possible at this point.
	//	if err := spc.UpdatePodClaimForRetentionPolicy(ctx, set, pod); err != nil {
	//		spc.recordPodEvent("update", set, pod, err)
	//		return err
	//	}
	//}
	return nil
}

func (sbc *StatefulBoxControl) UpdateBoxByBoxStatefulSet(ctx context.Context, boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) error {
	attemptedUpdate := false
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// assume the Box is consistent
		consistent := true
		// if the Box does not conform to its identity, update the identity and dirty the Box
		if !identityMatches(boxSts, box) {
			updateIdentity(boxSts, box)
			consistent = false
		}
		// if the Box does not conform to the BoxStatefulSet's storage requirements, update the Box's PVC's,
		// dirty the Box, and create any missing PVCs
		if !storageMatches(boxSts, box) {
			updateStorage(boxSts, box)
			consistent = false
			if err := sbc.createPersistentVolumeClaims(ctx, boxSts, box); err != nil {
				sbc.recordBoxEvent("update", boxSts, box, err)
				return err
			}
		}
		//if utilfeature.DefaultFeatureGate.Enabled(features.StatefulSetAutoDeletePVC) {
		//	// if the Box's PVCs are not consistent with the StatefulSet's PVC deletion policy, update the PVC
		//	// and dirty the pod.
		//	if match, err := spc.ClaimsMatchRetentionPolicy(ctx, set, pod); err != nil {
		//		spc.recordPodEvent("update", set, pod, err)
		//		return err
		//	} else if !match {
		//		if err := spc.UpdatePodClaimForRetentionPolicy(ctx, set, pod); err != nil {
		//			spc.recordPodEvent("update", set, pod, err)
		//			return err
		//		}
		//		consistent = false
		//	}
		//}

		// if the Box is not dirty, do nothing
		if consistent {
			return nil
		}

		attemptedUpdate = true
		// commit the update, retrying on conflicts

		updateErr := sbc.objectMgr.UpdateBox(ctx, box)
		if updateErr == nil {
			return nil
		}

		if updated, err := sbc.objectMgr.GetBox(ctx, boxSts.Namespace, box.Name); err == nil {
			// make a copy so we don't mutate the shared cache
			box = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated Box %s/%s: %w", boxSts.Namespace, box.Name, err))
		}

		return updateErr
	})
	if attemptedUpdate {
		sbc.recordBoxEvent("update", boxSts, box, err)
	}
	return err
}
