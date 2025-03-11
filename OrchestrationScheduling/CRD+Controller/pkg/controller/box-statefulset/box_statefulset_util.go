package box_statefulset

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	boxstatefulsetv1alpha1 "cncos.io/box-controller/pkg/apis/boxstatefulset/v1alpha1"
	boxutil "cncos.io/box-controller/pkg/util/box"
)

const (
	ControllerRevisionHashLabelKey = "controller-revision-hash"
	BoxStatefulSetBoxName          = "box-stateful-set-box-name"
)

// boxInOrdinalRange returns true if the box ordinal is within the allowed
// range of ordinals that this StatefulSet is set to control.
func boxInOrdinalRange(box *boxv1alpha1.Box, boxSts *boxstatefulsetv1alpha1.BoxStatefulSet) bool {
	ordinal := getOrdinal(box)
	return ordinal >= getStartOrdinal(boxSts) && ordinal <= getEndOrdinal(boxSts)
}

// getStartOrdinal gets the first possible ordinal (inclusive).
// Returns spec.ordinals.start if spec.ordinals is set, otherwise returns 0.
func getStartOrdinal(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet) int {
	//if utilfeature.DefaultFeatureGate.Enabled(features.StatefulSetStartOrdinal) {
	//	if boxSts.Spec.Ordinals != nil {
	//		return int(boxSts.Spec.Ordinals.Start)
	//	}
	//}
	return 0
}

// getEndOrdinal gets the last possible ordinal (inclusive).
func getEndOrdinal(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet) int {
	return getStartOrdinal(boxSts) + int(*boxSts.Spec.Replicas) - 1
}

// getOrdinal gets pod's ordinal. If pod has no ordinal, -1 is returned.
func getOrdinal(box *boxv1alpha1.Box) int {
	_, ordinal := getParentNameAndOrdinal(box)
	return ordinal
}

// boxStatefulSetRegex is a regular expression that extracts the parent BoxStatefulSet and ordinal from the Name of a Box
var boxStatefulSetRegex = regexp.MustCompile("(.*)-([0-9]+)$")

// getParentNameAndOrdinal gets the name of box's parent BoxStatefulSet and box's ordinal as extracted from its Name. If
// the Box was not created by a BoxStatefulSet, its parent is considered to be empty string, and its ordinal is considered
// to be -1.
func getParentNameAndOrdinal(box *boxv1alpha1.Box) (string, int) {
	parent := ""
	ordinal := -1
	subMatches := boxStatefulSetRegex.FindStringSubmatch(box.Name)
	if len(subMatches) < 3 {
		return parent, ordinal
	}
	parent = subMatches[1]
	if i, err := strconv.ParseInt(subMatches[2], 10, 32); err == nil {
		ordinal = int(i)
	}
	return parent, ordinal
}

// nextRevision finds the next valid revision number based on revisions. If the length of revisions
// is 0 this is 1. Otherwise, it is 1 greater than the largest revision's Revision. This method
// assumes that revisions has been sorted by Revision.
func nextRevision(revisions []*apps.ControllerRevision) int64 {
	count := len(revisions)
	if count <= 0 {
		return 1
	}
	return revisions[count-1].Revision + 1
}

// ApplyRevision returns a new BoxStatefulSet constructed by restoring the state in revision to set. If the returned error
// is nil, the returned BoxStatefulSet is valid.
func ApplyRevision(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, revision *apps.ControllerRevision) (*boxstatefulsetv1alpha1.BoxStatefulSet, error) {
	clone := boxSts.DeepCopy()
	patched, err := strategicpatch.StrategicMergePatch([]byte(runtime.EncodeOrDie(patchCodec, clone)), revision.Data.Raw, clone)
	if err != nil {
		return nil, err
	}
	restoredSet := &boxstatefulsetv1alpha1.BoxStatefulSet{}
	err = json.Unmarshal(patched, restoredSet)
	if err != nil {
		return nil, err
	}
	return restoredSet, nil
}

// isCreated returns true if box has been created and is maintained by the API server
func isCreated(box *boxv1alpha1.Box) bool {
	return box.Status.Phase != ""
}

// isPending returns true if box has a Phase of PodPending
func isPending(box *boxv1alpha1.Box) bool {
	return box.Status.Phase == corev1.PodPending
}

// isFailed returns true if box has a Phase of PodFailed
func isFailed(box *boxv1alpha1.Box) bool {
	return box.Status.Phase == corev1.PodFailed
}

// isTerminating returns true if box's DeletionTimestamp has been set
func isTerminating(box *boxv1alpha1.Box) bool {
	return box.DeletionTimestamp != nil
}

// isHealthy returns true if box is running and ready and has not been terminated
func isHealthy(box *boxv1alpha1.Box) bool {
	return isRunningAndReady(box) && !isTerminating(box)
}

// isRunningAndReady returns true if box is in the PodRunning Phase, if it has a condition of PodReady.
func isRunningAndReady(box *boxv1alpha1.Box) bool {
	return box.Status.Phase == corev1.PodRunning && boxutil.IsBoxReady(box)
}

func isRunningAndAvailable(box *boxv1alpha1.Box, minReadySeconds int32) bool {
	return boxutil.IsBoxAvailable(box, minReadySeconds, metav1.Now())
}

// getBoxRevision gets the revision of Box by inspecting the ControllerRevisionHashLabelKey. If box has no revision the empty
// string is returned.
func getBoxRevision(box *boxv1alpha1.Box) string {
	if box.Labels == nil {
		return ""
	}
	return box.Labels[ControllerRevisionHashLabelKey]
}

func newVersionedBoxStatefulSet(currentBoxSts, updateBoxSts *boxstatefulsetv1alpha1.BoxStatefulSet, currentRevision, updateRevision string, ordinal int) *boxv1alpha1.Box {
	if currentBoxSts.Spec.UpdateStrategy.Type == apps.RollingUpdateStatefulSetStrategyType &&
		(currentBoxSts.Spec.UpdateStrategy.RollingUpdate == nil && ordinal < (getStartOrdinal(currentBoxSts)+int(currentBoxSts.Status.CurrentReplicas))) ||
		(currentBoxSts.Spec.UpdateStrategy.RollingUpdate != nil && ordinal < (getStartOrdinal(currentBoxSts)+int(*currentBoxSts.Spec.UpdateStrategy.RollingUpdate.Partition))) {
		box := newStatefulSetBox(currentBoxSts, ordinal)
		setBoxRevision(box, currentRevision)
		return box
	}
	box := newStatefulSetBox(updateBoxSts, ordinal)
	setBoxRevision(box, updateRevision)
	return box
}

// setBoxRevision sets the revision of Box to revision by adding the ControllerRevisionHashLabelKey
func setBoxRevision(box *boxv1alpha1.Box, revision string) {
	if box.Labels == nil {
		box.Labels = make(map[string]string)
	}
	box.Labels[ControllerRevisionHashLabelKey] = revision
}

// newStatefulSetBox returns a new Box conforming to the set's Spec with an identity generated from ordinal.
func newStatefulSetBox(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, ordinal int) *boxv1alpha1.Box {
	box, _ := newBoxByBoxStatefulSet(boxSts)

	box.Name = getBoxName(boxSts, ordinal)
	box.Namespace = boxSts.Namespace
	box.Spec.Hostname = box.Name
	box.Spec.Subdomain = boxSts.Spec.ServiceName
	// updateStorage
	currentVolumes := box.Spec.Volumes
	claims := getPersistentVolumeClaims(boxSts, ordinal)
	newVolumes := make([]corev1.Volume, 0, len(claims))
	for name, claim := range claims {
		newVolumes = append(newVolumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: claim.Name,
					// TODO: Use source definition to set this value when we have one.
					ReadOnly: false,
				},
			},
		})
	}
	for i := range currentVolumes {
		if _, ok := claims[currentVolumes[i].Name]; !ok {
			newVolumes = append(newVolumes, currentVolumes[i])
		}
	}
	box.Spec.Volumes = newVolumes
	return box
}

func getPersistentVolumeClaims(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, ordinal int) map[string]corev1.PersistentVolumeClaim {
	templates := boxSts.Spec.VolumeClaimTemplates
	claims := make(map[string]corev1.PersistentVolumeClaim, len(templates))
	for i := range templates {
		claim := templates[i]
		claim.Name = getPersistentVolumeClaimName(boxSts, &claim, ordinal)
		claim.Namespace = boxSts.Namespace
		if claim.Labels != nil {
			for key, value := range boxSts.Spec.Selector.MatchLabels {
				claim.Labels[key] = value
			}
		} else {
			claim.Labels = boxSts.Spec.Selector.MatchLabels
		}
		claims[templates[i].Name] = claim
	}
	return claims
}

// getBoxName gets the name of set's child Pod with an ordinal index of ordinal
func getBoxName(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, ordinal int) string {
	return fmt.Sprintf("%s-%d", boxSts.Name, ordinal)
}

// getPersistentVolumeClaimName gets the name of PersistentVolumeClaim for a Pod with an ordinal index of ordinal. claim
// must be a PersistentVolumeClaim from set's VolumeClaims template.
func getPersistentVolumeClaimName(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, claim *corev1.PersistentVolumeClaim, ordinal int) string {
	// NOTE: This name format is used by the heuristics for zone spreading in ChooseZoneForVolume
	return fmt.Sprintf("%s-%s-%d", claim.Name, boxSts.Name, ordinal)
}

func newBoxByBoxStatefulSet(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet) (*boxv1alpha1.Box, error) {
	box := &boxv1alpha1.Box{
		ObjectMeta: metav1.ObjectMeta{
			Labels:       boxSts.Spec.Template.Labels,
			Annotations:  boxSts.Spec.Template.Annotations,
			GenerateName: boxSts.Name + "-",
			Finalizers:   boxSts.Spec.Template.Finalizers,
		},
	}
	controllerRef := metav1.NewControllerRef(boxSts, boxstatefulsetv1alpha1.BoxStatefulSetGroupVersionKind)
	if controllerRef != nil {
		box.OwnerReferences = append(box.OwnerReferences, *controllerRef)
	}

	specData, err := json.Marshal(boxSts.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}
	boxSpec := boxv1alpha1.BoxSpecV2{}
	if err := json.Unmarshal(specData, &boxSpec); err != nil {
		return nil, err
	}
	box.Spec = boxSpec

	return box, nil
}

// descendingOrdinal is a sort.Interface that Sorts a list of Boxes based on the ordinals extracted
// from the Box. Box's that have not been constructed by BoxStatefulSet's have an ordinal of -1, and are therefore pushed
// to the end of the list.
type descendingOrdinal []*boxv1alpha1.Box

func (do descendingOrdinal) Len() int {
	return len(do)
}

func (do descendingOrdinal) Swap(i, j int) {
	do[i], do[j] = do[j], do[i]
}

func (do descendingOrdinal) Less(i, j int) bool {
	return getOrdinal(do[i]) > getOrdinal(do[j])
}

// inconsistentStatus returns true if the ObservedGeneration of status is greater than set's
// Generation or if any of the status's fields do not match those of set's status.
func inconsistentStatus(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, status *boxstatefulsetv1alpha1.BoxStatefulSetStatus) bool {
	return status.ObservedGeneration > boxSts.Status.ObservedGeneration ||
		status.Replicas != boxSts.Status.Replicas ||
		status.CurrentReplicas != boxSts.Status.CurrentReplicas ||
		status.ReadyReplicas != boxSts.Status.ReadyReplicas ||
		status.UpdatedReplicas != boxSts.Status.UpdatedReplicas ||
		status.CurrentRevision != boxSts.Status.CurrentRevision ||
		status.AvailableReplicas != boxSts.Status.AvailableReplicas ||
		status.UpdateRevision != boxSts.Status.UpdateRevision
}

// completeRollingUpdate completes a rolling update when all of set's replica Boxes have been updated
// to the updateRevision. status's currentRevision is set to updateRevision and its' updateRevision
// is set to the empty string. status's currentReplicas is set to updateReplicas and its updateReplicas
// are set to 0.
func completeRollingUpdate(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, status *boxstatefulsetv1alpha1.BoxStatefulSetStatus) {
	if boxSts.Spec.UpdateStrategy.Type == apps.RollingUpdateStatefulSetStrategyType &&
		status.UpdatedReplicas == status.Replicas &&
		status.ReadyReplicas == status.Replicas {
		status.CurrentReplicas = status.UpdatedReplicas
		status.CurrentRevision = status.UpdateRevision
	}
}

// identityMatches returns true if pod has a valid identity and network identity for a member of set.
func identityMatches(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) bool {
	parent, ordinal := getParentNameAndOrdinal(box)
	return ordinal >= 0 &&
		boxSts.Name == parent &&
		box.Name == getBoxName(boxSts, ordinal) &&
		box.Namespace == boxSts.Namespace &&
		box.Labels[BoxStatefulSetBoxName] == box.Name
}

// updateIdentity updates box's name, hostname, and subdomain, and BoxStatefulSetBoxName to conform to BoxStatefulSet's name
// and headless service.
func updateIdentity(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) {
	ordinal := getOrdinal(box)
	box.Name = getBoxName(boxSts, ordinal)
	box.Namespace = boxSts.Namespace
	if box.Labels == nil {
		box.Labels = make(map[string]string)
	}
	box.Labels[BoxStatefulSetBoxName] = box.Name
	//if utilfeature.DefaultFeatureGate.Enabled(features.PodIndexLabel) {
	//	pod.Labels[apps.PodIndexLabel] = strconv.Itoa(ordinal)
	//}
}

// storageMatches returns true if box's Volumes cover the set of PersistentVolumeClaims
func storageMatches(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) bool {
	ordinal := getOrdinal(box)
	if ordinal < 0 {
		return false
	}
	volumes := make(map[string]corev1.Volume, len(box.Spec.Volumes))
	for _, volume := range box.Spec.Volumes {
		volumes[volume.Name] = volume
	}
	for _, claim := range boxSts.Spec.VolumeClaimTemplates {
		volume, found := volumes[claim.Name]
		if !found ||
			volume.VolumeSource.PersistentVolumeClaim == nil ||
			volume.VolumeSource.PersistentVolumeClaim.ClaimName !=
				getPersistentVolumeClaimName(boxSts, &claim, ordinal) {
			return false
		}
	}
	return true
}

// updateStorage updates Box's Volumes to conform with the PersistentVolumeClaim of BoxStatefulSet's templates. If box has
// conflicting local Volumes these are replaced with Volumes that conform to the set's templates.
func updateStorage(boxSts *boxstatefulsetv1alpha1.BoxStatefulSet, box *boxv1alpha1.Box) {
	currentVolumes := box.Spec.Volumes
	claims := getPersistentVolumeClaims(boxSts, getOrdinal(box))
	newVolumes := make([]corev1.Volume, 0, len(claims))
	for name, claim := range claims {
		newVolumes = append(newVolumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: claim.Name,
					// TODO: Use source definition to set this value when we have one.
					ReadOnly: false,
				},
			},
		})
	}
	for i := range currentVolumes {
		if _, ok := claims[currentVolumes[i].Name]; !ok {
			newVolumes = append(newVolumes, currentVolumes[i])
		}
	}
	box.Spec.Volumes = newVolumes
}

func toString(obj interface{}) string {
	b, _ := json.Marshal(obj)
	return string(b)
}
