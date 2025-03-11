package box_statefulset

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	boxstatesetv1alpha1 "cncos.io/box-controller/pkg/apis/boxstatefulset/v1alpha1"
	boxstatefulsetclientset "cncos.io/box-controller/pkg/boxstatefulset-generated/clientset/versioned"
)

// BoxStatefulSetStatusUpdaterInterface is an interface used to update the StatefulSetStatus associated with a StatefulSet.
// For any use other than testing, clients should create an instance using NewRealStatefulSetStatusUpdater.
type BoxStatefulSetStatusUpdaterInterface interface {
	// UpdateBoxStatefulSetStatus sets the set's Status to status. Implementations are required to retry on conflicts,
	// but fail on other errors. If the returned error is nil set's Status has been successfully set to status.
	UpdateBoxStatefulSetStatus(ctx context.Context, boxSts *boxstatesetv1alpha1.BoxStatefulSet, status *boxstatesetv1alpha1.BoxStatefulSetStatus) error
}

// NewRealBoxStatefulSetStatusUpdater returns a StatefulSetStatusUpdaterInterface that updates the Status of a StatefulSet,
// using the supplied client and setLister.
func NewRealBoxStatefulSetStatusUpdater(
	boxStatefulSetClientSet boxstatefulsetclientset.Interface) BoxStatefulSetStatusUpdaterInterface {
	return &realBoxStatefulSetStatusUpdater{boxStatefulSetClientSet}
}

type realBoxStatefulSetStatusUpdater struct {
	boxStatefulSetClientSet boxstatefulsetclientset.Interface
}

func (ssu *realBoxStatefulSetStatusUpdater) UpdateBoxStatefulSetStatus(ctx context.Context, boxSts *boxstatesetv1alpha1.BoxStatefulSet, status *boxstatesetv1alpha1.BoxStatefulSetStatus) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		boxSts.Status = *status
		// TODO: This context.TODO should use a real context once we have RetryOnConflictWithContext
		_, updateErr := ssu.boxStatefulSetClientSet.CncosV1alpha1().BoxStatefulSets(boxSts.Namespace).UpdateStatus(ctx, boxSts, metav1.UpdateOptions{})
		if updateErr == nil {
			return nil
		} else {
			klog.Errorf("update BoxStatefulSets(%s/%s) status failed, err: %s, status:%s", boxSts.Namespace, boxSts.Name, updateErr, toString(status))
		}
		updated, err := ssu.boxStatefulSetClientSet.CncosV1alpha1().BoxStatefulSets(boxSts.Namespace).Get(ctx, boxSts.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("get BoxStatefulSets failed, err: %s", err)
		} else {
			// make a copy so we don't mutate the shared cache
			boxSts = updated.DeepCopy()
		}
		return updateErr
	})
}
