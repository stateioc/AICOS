package box

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
)

func IsBoxAvailable(box *boxv1alpha1.Box, minReadySeconds int32, now metav1.Time) bool {
	if !IsBoxReady(box) {
		return false
	}

	c := GetBoxReadyCondition(box.Status)
	minReadySecondsDuration := time.Duration(minReadySeconds) * time.Second
	if minReadySeconds == 0 || (!c.LastTransitionTime.IsZero() && c.LastTransitionTime.Add(minReadySecondsDuration).Before(now.Time)) {
		return true
	}
	return false
}

// IsBoxReady returns true if a pod is ready; false otherwise.
func IsBoxReady(box *boxv1alpha1.Box) bool {
	return IsBoxReadyConditionTrue(box.Status)
}

// IsBoxReadyConditionTrue returns true if a pod is ready; false otherwise.
func IsBoxReadyConditionTrue(status boxv1alpha1.BoxStatusV2) bool {
	condition := GetBoxReadyCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// GetBoxReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetBoxReadyCondition(status boxv1alpha1.BoxStatusV2) *v1.PodCondition {
	_, condition := GetPodConditionFromList(status.Conditions, v1.PodReady)
	return condition
}

// GetPodConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func GetPodConditionFromList(conditions []v1.PodCondition, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}
