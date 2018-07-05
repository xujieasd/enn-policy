package util

import (
	api "k8s.io/api/core/v1"
)

// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *api.Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// IsPodReady retruns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status api.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == api.ConditionTrue
}

// Extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetPodReadyCondition(status api.PodStatus) *api.PodCondition {
	_, condition := GetPodCondition(&status, api.PodReady)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetPodCondition(status *api.PodStatus, conditionType api.PodConditionType) (int, *api.PodCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

func IsPodValid(pod *api.Pod) bool {

	switch pod.Status.Phase{
	case api.PodSucceeded:
		return false
	case api.PodFailed:
		return false
	default:
		return true
	}
}

func IsPodNotReady(pod *api.Pod) bool {
	switch pod.Spec.RestartPolicy {
	case api.RestartPolicyNever:
		return pod.Status.Phase != api.PodFailed && pod.Status.Phase != api.PodSucceeded
	case api.RestartPolicyOnFailure:
		return pod.Status.Phase != api.PodSucceeded
	default:
		return true
	}
}