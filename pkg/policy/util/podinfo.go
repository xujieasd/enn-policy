package util

import (
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"github.com/golang/glog"
	"sync"
	"reflect"
)

type PodInfoMap map[string]*PodInfo

// PodMatchLabelMap represent the pods set for corresponding label of pod,
// key: NamespacedLabel, value: PodInfoMap which key is pod ip, value is podInfo
type PodMatchLabelMap map[NamespacedLabel]PodInfoMap

// NamespacePodMap represent the pods set for corresponding namespace,
// key: namsepace, value: mPodInfoMap which key is pod ip, value is podInfo
type NamespacePodMap map[string]PodInfoMap

// PodInfo is a collection of containers that can run on a host. This resource is created by clients and scheduled onto hosts.
// PodInfo will collect useful information from Pod spec
type PodInfo struct {

	IP                string
	Name              string
	Namespace         string
	Labels            map[string]string
}

type PodChangeMap struct {
	Lock                sync.Mutex
	PodItems            map[types.NamespacedName]*PodLabelChange
	NamespacePodItems   map[types.NamespacedName]*NamespacePodChange
	NamespaceLabelItems map[types.NamespacedName]*NamespacePodChange

}

type PodLabelChange struct {
	Previous PodMatchLabelMap
	Current  PodMatchLabelMap
}

type NamespacePodChange struct {
	Previous NamespacePodMap
	Current  NamespacePodMap
}

func NewPodLabelChangeMap() PodChangeMap {
	return PodChangeMap{
		PodItems:            make(map[types.NamespacedName]*PodLabelChange),
		NamespacePodItems:   make(map[types.NamespacedName]*NamespacePodChange),
		NamespaceLabelItems: make(map[types.NamespacedName]*NamespacePodChange),
	}
}

func (pcm *PodChangeMap) CleanUpItem(){
	pcm.PodItems            = make(map[types.NamespacedName]*PodLabelChange)
	pcm.NamespacePodItems   = make(map[types.NamespacedName]*NamespacePodChange)
	pcm.NamespaceLabelItems = make(map[types.NamespacedName]*NamespacePodChange)
}

func (pcm *PodChangeMap) Update(namespacedName *types.NamespacedName, previous, current *api.Pod) bool{
	glog.V(3).Infof("UpdatePodChangeMap start")

	pcm.Lock.Lock()
	defer pcm.Lock.Unlock()

	// update PodLabelChange, which means update pod sets with corresponding podSpec label
	podChange, exists := pcm.PodItems[*namespacedName]

	if !exists{
		podChange = &PodLabelChange{}
		podChange.Previous = PodToPodMatchLabelMap(previous)
		pcm.PodItems[*namespacedName] = podChange
	}
	podChange.Current = PodToPodMatchLabelMap(current)
	if reflect.DeepEqual(podChange.Previous, podChange.Current) {
		delete(pcm.PodItems, *namespacedName)
	}

	// update namespacePodChange, which means update pod set with corresponding namespace
	// namespacePodChange is used to update NamespacePodMap
	namespacePodChange, exists := pcm.NamespacePodItems[*namespacedName]

	if !exists{
		namespacePodChange = &NamespacePodChange{}
		namespacePodChange.Previous = PodToNamespacePodMap(previous)
		pcm.NamespacePodItems[*namespacedName] = namespacePodChange
	}
	namespacePodChange.Current = PodToNamespacePodMap(current)
	if reflect.DeepEqual(namespacePodChange.Previous, namespacePodChange.Current){
		delete(pcm.NamespacePodItems, *namespacedName)
	}

	// update namespaceLabelChange, which means update pod set with corresponding namespace
	// namespaceLabelChange is used to update NamespaceMatchLabelMap
	namespaceLabelChange, exists := pcm.NamespaceLabelItems[*namespacedName]
	if !exists{
		namespaceLabelChange = &NamespacePodChange{}
		namespaceLabelChange.Previous = PodToNamespacePodMap(previous)
		pcm.NamespaceLabelItems[*namespacedName] = namespaceLabelChange
	}
	namespaceLabelChange.Current = PodToNamespacePodMap(current)
	if reflect.DeepEqual(namespaceLabelChange.Previous, namespaceLabelChange.Current){
		delete(pcm.NamespaceLabelItems, *namespacedName)
	}

	glog.V(6).Infof("PodChangeMap changed podItem number is %d, changed namespacePodItem number is %d, changed namespaceLabelItem number is %d",
		len(pcm.PodItems), len(pcm.NamespacePodItems), len(pcm.NamespaceLabelItems))
	return len(pcm.PodItems) > 0 || len(pcm.NamespacePodItems) > 0 || len(pcm.NamespaceLabelItems) > 0
}

func (pcm *PodChangeMap) Changed() bool{
	return len(pcm.PodItems) > 0 || len(pcm.NamespacePodItems) > 0 || len(pcm.NamespaceLabelItems) > 0
}

func PodToPodMatchLabelMap(pod *api.Pod) PodMatchLabelMap {

	if pod == nil{
		return nil
	}

	glog.V(7).Infof("watch podInfo: namespacedName: %s:%s, ip: %s, phase: %s", pod.Namespace, pod.Name, pod.Status.PodIP, pod.Status.Phase)

	for i := range pod.Status.Conditions{
		glog.V(7).Infof("pod condition type: %s, states: %s", string(pod.Status.Conditions[i].Type), string(pod.Status.Conditions[i].Status))
	}

	if len(pod.Labels) == 0{
		glog.V(7).Infof("skip build map because pod: %s/%s does not have label", pod.Namespace, pod.Name)
	}

	// when a new pod is created, we will first watch OnPodAdd event, pod name is assigned but pod ip is not
	// then OnPodUpdate event comes, which will update pod ip from empty to real ip
	// also when pod is deleted, we will first watch OnPodUpdate event, which update pod ip from real ip to empty
	// then OnPodDelete event comes, which will delete the pod (previous pod with empty ip and current pod is nil)
	// enn-policy should not handle pod with empty ip, enn-policy stores ip in it's map only when pod ip is real,
	// so if pod ip is empty, skip build map

	if pod.Status.PodIP == "" {
		glog.V(6).Infof("skip build map because pod: %s/%s ip is empty", pod.Namespace, pod.Name)
		return nil
	}

	if pod.Spec.HostNetwork{
		glog.V(6).Infof("skip build pod map because pod: %s/%s host network is true", pod.Namespace, pod.Name)
		return nil
	}

	if !IsPodValid(pod){
		glog.V(6).Infof("pod: %s/%s is not valid pod phase: %s", pod.Namespace, pod.Name, string(pod.Status.Phase))
		return nil
	}

	podMatchLabelMap := make(PodMatchLabelMap)

	for k,v := range pod.Labels{
		namespacedLabel := NamespacedLabel{
			Namespace:   pod.Namespace,
			LabelKey:    k,
			LabelValue:  v,
		}
		infoMap := make(PodInfoMap)
		podInfo := &PodInfo{
			IP:           pod.Status.PodIP,
			Name:         pod.Name,
			Namespace:    pod.Namespace,
			Labels:       pod.Labels,
		}

		infoMap[pod.Status.PodIP] = podInfo
		podMatchLabelMap[namespacedLabel] = infoMap
	}

	return podMatchLabelMap
}

func PodToNamespacePodMap(pod *api.Pod) NamespacePodMap {

	if pod == nil{
		return nil
	}

	// when a new pod is created, we will first watch OnPodAdd event, pod name is assigned but pod ip is not
	// then OnPodUpdate event comes, which will update pod ip from empty to real ip
	// also when pod is deleted, we will first watch OnPodUpdate event, which update pod ip from real ip to empty
	// then OnPodDelete event comes, which will delete the pod (previous pod with empty ip and current pod is nil)
	// enn-policy should not handle pod with empty ip, enn-policy stores ip in it's map only when pod ip is real,
	// so if pod ip is empty, skip build map

	if pod.Status.PodIP == "" {
		glog.V(6).Infof("skip build map because pod: %s/%s ip is empty", pod.Namespace, pod.Name)
		return nil
	}

	if pod.Spec.HostNetwork{
		glog.V(6).Infof("skip build pod map because pod: %s/%s host network is true", pod.Namespace, pod.Name)
		return nil
	}

	if !IsPodValid(pod){
		glog.V(6).Infof("pod: %s/%s is not valid pod phase: %s", pod.Namespace, pod.Name, string(pod.Status.Phase))
		return nil
	}

	namespacePodMap := make(NamespacePodMap)

	infoMap := make(PodInfoMap)
	podInfo := &PodInfo{
		IP:           pod.Status.PodIP,
		Name:         pod.Name,
		Namespace:    pod.Namespace,
		Labels:       pod.Labels,
	}

	infoMap[pod.Status.PodIP] = podInfo
	namespacePodMap[pod.Namespace] = infoMap

	return namespacePodMap
}

func UpdatePodMatchLabelMap(podMatchLabelMap PodMatchLabelMap, changes *PodChangeMap){

	for _, change := range changes.PodItems {
		podMatchLabelMap.unmerge(change.Previous)
		podMatchLabelMap.merge(change.Current)
	}
}

func (pmlm *PodMatchLabelMap) merge(other PodMatchLabelMap){

	for namespaceLabel, infoMap := range other{
		oldInfoMap, has := (*pmlm)[namespaceLabel]
		if !has {
			glog.V(1).Infof("add new namespaceLabel %s:%s=%s to PodMatchLabelMap",
				namespaceLabel.Namespace,
				namespaceLabel.LabelKey,
				namespaceLabel.LabelValue,
			)
			tempInfoMap := make(PodInfoMap)
			for ip, portInfo := range infoMap{
				glog.V(1).Infof("add new pod %s:%s to new pod set",
					portInfo.Name,
					portInfo.IP,
				)
				tempInfoMap[ip] = portInfo
			}
			(*pmlm)[namespaceLabel] = tempInfoMap
		} else {
			glog.V(1).Infof("update namespaceLabel %s:%s=%s to PodMatchLabelMap",
				namespaceLabel.Namespace,
				namespaceLabel.LabelKey,
				namespaceLabel.LabelValue,
			)
			for ip, portInfo := range infoMap{
				_, hasPod := oldInfoMap[ip]
				if !hasPod{
					glog.V(1).Infof("add new pod %s:%s",
						portInfo.Name,
						portInfo.IP,
					)
				} else {
					glog.V(1).Infof("exist pod ip, so update pod %s:%s",
						portInfo.Name,
						portInfo.IP,
					)
				}
				oldInfoMap[ip] = portInfo
			}
			(*pmlm)[namespaceLabel] = oldInfoMap
		}
	}
}

func (pmlm *PodMatchLabelMap) unmerge(other PodMatchLabelMap){

	for namespaceLabel, infoMap := range other{
		oldInfoMap, has := (*pmlm)[namespaceLabel]
		if !has{
			glog.Errorf("try to remove pod from PodMatchLabelMap %s:%s=%s, but namespaceLabel do not exists",
				namespaceLabel.Namespace,
				namespaceLabel.LabelKey,
				namespaceLabel.LabelValue,
			)
			continue
		} else {
			for ip, portInfo := range infoMap{
				_, hasPod := oldInfoMap[ip]
				if !hasPod{
					glog.Errorf("try to remove pod from PodMatchLabelMap %s:%s=%s, but pod %s:%s do not exists",
						namespaceLabel.Namespace,
						namespaceLabel.LabelKey,
						namespaceLabel.LabelValue,
						portInfo.Name,
						portInfo.IP,
					)
				} else {
					glog.V(1).Infof("remove pod %s:%s from PodMatchLabelMap %s:%s=%s",
						portInfo.Name,
						portInfo.IP,
						namespaceLabel.Namespace,
						namespaceLabel.LabelKey,
						namespaceLabel.LabelValue,
					)
					delete(oldInfoMap, ip)
				}
			}
		}
		(*pmlm)[namespaceLabel] = oldInfoMap
	}

}

func UpdateNamespacePodMap(namespacePodMap NamespacePodMap, changes *PodChangeMap){


	for _, change := range changes.NamespacePodItems {
		namespacePodMap.unmerge(change.Previous)
		namespacePodMap.merge(change.Current)
	}
}

func (npm *NamespacePodMap) merge(other NamespacePodMap){

	for namespace, infoMap := range other {
		oldInfoMap, has := (*npm)[namespace]
		if !has{
			glog.V(1).Infof("add new namespace %s to NamespacePodMap", namespace)
			tempInfoMap := make(PodInfoMap)
			for ip, portInfo := range infoMap {
				glog.V(1).Infof("add new pod %s:%s to new pod set",
					portInfo.Name,
					portInfo.IP,
				)
				tempInfoMap[ip] = portInfo
			}
			(*npm)[namespace] = tempInfoMap
		} else {
			glog.V(1).Infof("update namespace %s to NamespacePodMap", namespace)
			for ip, portInfo := range infoMap {
				_, hasPod := oldInfoMap[ip]
				if !hasPod{
					glog.V(1).Infof("add new pod %s:%s",
						portInfo.Name,
						portInfo.IP,
					)
				} else {
					glog.V(1).Infof("exist pod ip, so update pod %s:%s",
						portInfo.Name,
						portInfo.IP,
					)
				}
				oldInfoMap[ip] = portInfo
			}
			(*npm)[namespace] = oldInfoMap
		}
	}

}

func (npm *NamespacePodMap) unmerge(other NamespacePodMap){

	for namespace, infoMap := range other{
		oldInfoMap, has := (*npm)[namespace]
		if !has{
			glog.Errorf("try to remove pod from NamespacePodMap but namespace %s do not find", namespace)
			continue
		}else{
			for ip, portInfo := range infoMap {
				_, hasPod := oldInfoMap[ip]
				if !hasPod{
					glog.Errorf("try to remove pod from PodMatchLabelMap namespace %s, but pod %s:%s do not exists",
						namespace,
						portInfo.Name,
						portInfo.IP,
					)
				} else {
					glog.V(1).Infof("remove pod %s:%s from PodMatchLabelMap namespace %s",
						portInfo.Name,
						portInfo.IP,
						namespace,
					)
					delete(oldInfoMap, ip)
				}
			}
		}
		(*npm)[namespace] = oldInfoMap
	}
}

func UpdateNamespaceMatchLabelMapByPod(
    namespaceMatchLabelMap NamespaceMatchLabelMap,
    namespaceInfoMap NamespaceInfoMap,
    changes *PodChangeMap){

	for _, change := range changes.NamespaceLabelItems {

		for namespace, podInfoMap := range change.Previous{

			namespaceInfo, ok := namespaceInfoMap[namespace]
			if !ok{
				glog.Errorf("cannot find namespace %s in namespace Info Map when Update NamespaceMatchLabelMap By Pod", namespace)
				continue
			}

			namespaceMatchLabelMap.unmergePodInfoMap(podInfoMap, namespaceInfo)

		}
		for namespace, podInfoMap :=  range change.Current{

			namespaceInfo, ok := namespaceInfoMap[namespace]
			if !ok{
				glog.Errorf("cannot find namespace %s in namespace Info Map when Update NamespaceMatchLabelMap By Pod", namespace)
				continue
			}

			namespaceMatchLabelMap.mergePodInfoMap(podInfoMap, namespaceInfo)
		}
	}

}