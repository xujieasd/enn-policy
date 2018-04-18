package util

import (
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"github.com/golang/glog"
	"sync"
	"reflect"
)
type NamespaceInfoMap map[string]*NamespaceInfo

// NamespaceMatchLabelMap represent the pods set for corresponding label of namespace,
// key: NamespacedLabel, value: PodInfoMap which key is pod ip, value is podInfo
type NamespaceMatchLabelMap map[Label]PodInfoMap

// NamespaceInfo provides a scope for Names.
// NamespaceInfo will collect useful information from Namespace spec
type NamespaceInfo struct {

	Name     string
	Labels   map[string]string
}

type NamespaceChangeMap struct {
	Lock   sync.Mutex
	Items  map[types.NamespacedName]*NamespaceChange
}

type NamespaceChange struct {
	Previous *NamespaceInfo
	Current  *NamespaceInfo
}

func NewNamespaceChangeMap() NamespaceChangeMap {
	return NamespaceChangeMap{
		Items:   make(map[types.NamespacedName]*NamespaceChange),
	}
}

func (ncm *NamespaceChangeMap) CleanUpItem(){
	ncm.Items = make(map[types.NamespacedName]*NamespaceChange)
}

func (ncm *NamespaceChangeMap) Update(namespacedName *types.NamespacedName, previous, current *api.Namespace) bool{

	glog.V(3).Infof("UpdateNamespaceChangeMap start")

	ncm.Lock.Lock()
	defer ncm.Lock.Unlock()

	change, exists := ncm.Items[*namespacedName]
	if !exists{
		change = &NamespaceChange{}
		change.Previous = buildNamespaceInfo(previous)
		ncm.Items[*namespacedName] = change
	}
	change.Current = buildNamespaceInfo(current)
	if reflect.DeepEqual(change.Previous, change.Current) {
		delete(ncm.Items, *namespacedName)
	}

	glog.V(6).Infof("NamespaceChangeMap changed item number is %d", len(ncm.Items))
	return len(ncm.Items) > 0
}

func (ncm *NamespaceChangeMap) Changed() bool{
	return len(ncm.Items) > 0
}

func buildNamespaceInfo(namespace *api.Namespace) *NamespaceInfo{

	if namespace == nil{
		return nil
	}

	namespaceInfo := &NamespaceInfo{
		Name:    namespace.Name,
		Labels:  namespace.Labels,
	}
	return namespaceInfo
}

func UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap NamespaceMatchLabelMap, namespacePodMap NamespacePodMap, changes *NamespaceChangeMap) {

	for _, change := range changes.Items {
		namespaceMatchLabelMap.unmerge(namespacePodMap, change.Previous)
		namespaceMatchLabelMap.merge(namespacePodMap, change.Current)
	}
}

func (nmlm *NamespaceMatchLabelMap) merge(namespacePodMap NamespacePodMap, other *NamespaceInfo){

	if len(namespacePodMap) == 0{
		return
	}

	if other == nil{
		return
	}

	infoMap, ok := namespacePodMap[other.Name]
	if !ok{
		glog.Errorf("cannot find namespace %s in NamespacePodMap", other.Name)
		return
	}
	for k, v := range other.Labels {
		label := Label{
			LabelKey:     k,
			LabelValue:   v,
		}
		oldInfoMap, has := (*nmlm)[label]
		if !has {
			glog.V(1).Infof("add new namespace %s pod set to NamespaceMatchLabelMap Label %s=%s",
				other.Name,
				label.LabelKey,
				label.LabelValue,
			)
			tmpInfoMap := make(PodInfoMap)
			for ip, podInfo := range infoMap{
				glog.V(4).Infof("add pod %s to new pod set", ip)
				tmpInfoMap[ip] = podInfo
			}
			(*nmlm)[label] = tmpInfoMap
		} else {
			glog.V(1).Infof("update exists namespace %s pod set to NamespaceMatchLabelMap Label %s=%s",
				other.Name,
				label.LabelKey,
				label.LabelValue,
			)
			for ip, podInfo := range infoMap{
				glog.V(4).Infof("add pod %s to exists pod set", ip)
				oldInfoMap[ip] = podInfo
			}
			(*nmlm)[label] = oldInfoMap
		}
	}

}

func (nmlm *NamespaceMatchLabelMap) unmerge(namespacePodMap NamespacePodMap, other *NamespaceInfo){

	if len(namespacePodMap) == 0{
		return
	}

	if other == nil{
		return
	}

	infoMap, ok := namespacePodMap[other.Name]
	if !ok{
		glog.Errorf("cannot find namespace %s in NamespacePodMap", other.Name)
		return
	}

	for k, v := range other.Labels{
		label := Label{
			LabelKey:     k,
			LabelValue:   v,
		}
		oldInfoMap, has := (*nmlm)[label]
		if !has{
			glog.Errorf("try to remove namespace %s pod set from NamespaceMatchLabelMap Label %s=%s, but Label do not exists",
				other.Name,
				label.LabelKey,
				label.LabelValue,
			)
		} else {
			glog.V(1).Infof("remove namespace %s pod set from NamespaceMatchLabelMap Label %s=%s",
				other.Name,
				label.LabelKey,
				label.LabelValue,
			)
			for ip, _ := range infoMap {
				glog.V(4).Infof("del pod %s to exists pod set", ip)
				delete(oldInfoMap, ip)
			}
			(*nmlm)[label] = oldInfoMap
		}
	}
}

func (nmlm *NamespaceMatchLabelMap) mergePodInfoMap(podInfoMap PodInfoMap, other *NamespaceInfo){

	if len(podInfoMap) == 0{
		return
	}

	if other == nil{
		return
	}

	for k, v := range other.Labels {
		label := Label{
			LabelKey:     k,
			LabelValue:   v,
		}

		oldInfoMap, has := (*nmlm)[label]
		if !has {
			glog.V(1).Infof("add new namespace %s pod set to NamespaceMatchLabelMap Label %s=%s",
				other.Name,
				label.LabelKey,
				label.LabelValue,
			)
			tmpInfoMap := make(PodInfoMap)
			for ip, podInfo := range podInfoMap{
				glog.V(4).Infof("add pod %s to new pod set", ip)
				tmpInfoMap[ip] = podInfo
			}
			(*nmlm)[label] = tmpInfoMap
		} else {
			glog.V(1).Infof("update exists namespace %s pod set to NamespaceMatchLabelMap Label %s=%s",
				other.Name,
				label.LabelKey,
				label.LabelValue,
			)

			for ip, podInfo := range podInfoMap{
				glog.V(4).Infof("add pod %s to exists pod set", ip)
				oldInfoMap[ip] = podInfo
			}
			(*nmlm)[label] = oldInfoMap
		}
	}
}

func (nmlm *NamespaceMatchLabelMap) unmergePodInfoMap(podInfoMap PodInfoMap, other *NamespaceInfo){

	if len(podInfoMap) == 0{
		return
	}

	if other == nil{
		return
	}

	for k, v := range other.Labels {
		label := Label{
			LabelKey:     k,
			LabelValue:   v,
		}
		oldInfoMap, has := (*nmlm)[label]
		if !has{
			glog.Errorf("try to remove namespace %s pod set from NamespaceMatchLabelMap Label %s=%s, but Label do not exists",
				other.Name,
				label.LabelKey,
				label.LabelValue,
			)
		}else {
			glog.V(1).Infof("remove namespace %s pod set from NamespaceMatchLabelMap Label %s=%s",
				other.Name,
				label.LabelKey,
				label.LabelValue,
			)
			for ip, _ := range podInfoMap {
				glog.V(4).Infof("del pod %s to exists pod set", ip)
				delete(oldInfoMap, ip)
			}
			(*nmlm)[label] = oldInfoMap
		}
	}
}

func UpdateNamespaceInfoMap(namespaceInfoMap NamespaceInfoMap, changes *NamespaceChangeMap) {

	for _, change := range changes.Items {
		namespaceInfoMap.unmerge(change.Previous)
		namespaceInfoMap.merge(change.Current)
	}
}

func (nim *NamespaceInfoMap) merge(other *NamespaceInfo){
	if other == nil{
		return
	}
	(*nim)[other.Name] = other
}

func (nim *NamespaceInfoMap) unmerge(other *NamespaceInfo){
	if other == nil{
		return
	}
	delete(*nim, other.Name)
}