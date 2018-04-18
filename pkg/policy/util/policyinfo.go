package util

import (
	policyApi "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"github.com/golang/glog"
	"sync"
	"reflect"
)

const(
	TypeIngress       = "ingress"
	TypeEgress        = "egress"
)

type Label struct {
	LabelKey          string
	LabelValue        string
}

type NamespacedLabel struct {

	Namespace         string
	LabelKey          string
	LabelValue        string
}

type NetworkPolicyMap map[types.NamespacedName]*NetworkPolicyInfo

// NetworkPolicyInfo will collect useful information from networkPolicy spec
type NetworkPolicyInfo struct {

	Name              string
	Namespace         string

	// Selects the pods to which this NetworkPolicy object applies.
	PodSelector       map[string]string
	// the set of pods which corresponding to PodSelector, key: portIP value:PodInfo
	TargetPods        map[string]PodInfo
        // List of ingress rules to be applied to the selected pods.
	Ingress           []IngressRule
	// List of egress rules to be applied to the selected pods.
	Egress            []EgressRule
	// List of rule types that the NetworkPolicy relates to.
	// Valid options are Ingress, Egress, or Ingress,Egress.
	PolicyType        []string
}

// IngressRule describes a particular set of traffic that is allowed to the pods
// IngressRule will collect useful information from networkPolicy spec
type IngressRule struct {

	Ports             []PolicyPort
	PodSelector       []LabelSelector
	NamespaceSelector []LabelSelector
	CIDR              []string
	// todo handler exceptCIDR
	ExceptCIDR        []string
}

// EgressRule describes a particular set of traffic that is allowed out of pods
// EgressRule will collect useful information from networkPolicy spec
type EgressRule struct {

	Ports             []PolicyPort
	PodSelector       []LabelSelector
	NamespaceSelector []LabelSelector
	CIDR              []string
	// todo handler exceptCIDR
	ExceptCIDR        []string
}

type TypedRule struct {

	Ports             []PolicyPort
	PodSelector       []LabelSelector
	NamespaceSelector []LabelSelector
	CIDR              []string
	// todo handler exceptCIDR
	ExceptCIDR        []string
}

type PolicyPort struct {

	Protocol          string
	Port              string
}

type LabelSelector struct {
	Label             map[string]string
	// the set of pods which corresponding to PodSelector and NamespaceSelector, key: portIP value:PodInfo
	MatchPods         map[string]PodInfo
}


type NetworkPolicyChangeMap struct {
	Lock  sync.Mutex
	Items map[types.NamespacedName]*NetworkPolicyChange
}

type NetworkPolicyChange struct {
	Previous *NetworkPolicyInfo
	Current  *NetworkPolicyInfo
}

func NewNetworkPolicyChangeMap() NetworkPolicyChangeMap {
	return NetworkPolicyChangeMap{
		Items: make(map[types.NamespacedName]*NetworkPolicyChange),
	}
}

func (ncm *NetworkPolicyChangeMap) Update(namespacedName *types.NamespacedName, previous, current *policyApi.NetworkPolicy) bool{
	glog.V(3).Infof("UpdateNetworkPolicyChangeMap start")

	ncm.Lock.Lock()
	defer ncm.Lock.Unlock()

	change, exists := ncm.Items[*namespacedName]

	if !exists{
		change = &NetworkPolicyChange{}
		change.Previous = buildNetworkPolicyInfo(previous)
		ncm.Items[*namespacedName] = change
	}
	change.Current = buildNetworkPolicyInfo(current)
	if reflect.DeepEqual(change.Previous, change.Current) {
		delete(ncm.Items, *namespacedName)
	}
	glog.V(6).Infof("NetworkPolicyChangeMap changed item number is %d", len(ncm.Items))
	return len(ncm.Items) > 0
}

func buildNetworkPolicyInfo(networkPolicy *policyApi.NetworkPolicy) *NetworkPolicyInfo{

	if networkPolicy == nil{
		return nil
	}

	policy := &NetworkPolicyInfo{
		Name:         networkPolicy.Name,
		Namespace:    networkPolicy.Namespace,
		PodSelector:  networkPolicy.Spec.PodSelector.MatchLabels,
		TargetPods:   make(map[string]PodInfo),
		Ingress:      make([]IngressRule,0),
		Egress:       make([]EgressRule,0),
		PolicyType:   make([]string, 0),
	}

	for _, policyType := range networkPolicy.Spec.PolicyTypes{
		if policyType == policyApi.PolicyTypeIngress{
			policy.PolicyType = append(policy.PolicyType, TypeIngress)
		}else if policyType == policyApi.PolicyTypeEgress{
			policy.PolicyType = append(policy.PolicyType, TypeEgress)
		}
	}

	//todo: add targetPods

	// handlle ingress rule map
	for _, specIngress := range networkPolicy.Spec.Ingress{

		InfoIngress := IngressRule{
			Ports:              make([]PolicyPort, 0),
			PodSelector:        make([]LabelSelector, 0),
			NamespaceSelector:  make([]LabelSelector, 0),
			CIDR:               make([]string, 0),
			ExceptCIDR:         make([]string, 0),
		}

		for _, specPorts := range specIngress.Ports{
			port := PolicyPort{
				Port:      specPorts.Port.String(),
				Protocol:  string(*specPorts.Protocol),
			}
			InfoIngress.Ports = append(InfoIngress.Ports, port)
		}

		for _, specPeer := range specIngress.From{

			if specPeer.PodSelector != nil{
				podSelect := LabelSelector{
					Label:      specPeer.PodSelector.MatchLabels,
					MatchPods:  make(map[string]PodInfo),
				}

				InfoIngress.PodSelector = append(InfoIngress.PodSelector, podSelect)
				//todo: handle match pods
			}

			if specPeer.NamespaceSelector != nil {
				namespaceSelect := LabelSelector{
					Label:      specPeer.NamespaceSelector.MatchLabels,
					MatchPods:  make(map[string]PodInfo),
				}

				InfoIngress.NamespaceSelector = append(InfoIngress.NamespaceSelector, namespaceSelect)
				//todo: handle match namespace
			}

			if specPeer.IPBlock != nil {

				InfoIngress.CIDR = append(InfoIngress.CIDR, specPeer.IPBlock.CIDR)

				// todo: need to hanlder exceptCIDR
			}
		}


		policy.Ingress = append(policy.Ingress, InfoIngress)
	}

	// handle egress rule map
	for _, specEgress := range networkPolicy.Spec.Egress{

		InfoEgress := EgressRule{
			Ports:              make([]PolicyPort, 0),
			PodSelector:        make([]LabelSelector, 0),
			NamespaceSelector:  make([]LabelSelector, 0),
			CIDR:               make([]string, 0),
			ExceptCIDR:         make([]string, 0),
		}

		for _, specPorts := range specEgress.Ports{
			port := PolicyPort{
				Port:      specPorts.Port.String(),
				Protocol:  string(*specPorts.Protocol),
			}
			InfoEgress.Ports = append(InfoEgress.Ports, port)
		}

		for _, specPeer := range specEgress.To{

			if specPeer.PodSelector != nil {
				podSelect := LabelSelector{
					Label:      specPeer.PodSelector.MatchLabels,
					MatchPods:  make(map[string]PodInfo),
				}

				InfoEgress.PodSelector = append(InfoEgress.PodSelector, podSelect)
				// todo: handle match pods
			}

			if specPeer.NamespaceSelector != nil {
				namespaceSelect := LabelSelector{
					Label:      specPeer.NamespaceSelector.MatchLabels,
					MatchPods:  make(map[string]PodInfo),
				}

				InfoEgress.NamespaceSelector = append(InfoEgress.NamespaceSelector, namespaceSelect)
				// todo: handle match namespace

			}

			if specPeer.IPBlock != nil {

				InfoEgress.CIDR = append(InfoEgress.CIDR, specPeer.IPBlock.CIDR)

				// todo: need to hanlder exceptCIDR
			}
		}

		policy.Egress = append(policy.Egress, InfoEgress)
	}

	return policy
}

func UpdateNetworkPolicyMap(networkPolicyMap NetworkPolicyMap, changes *NetworkPolicyChangeMap) {

	changes.Lock.Lock()
	defer changes.Lock.Unlock()

	for namespacedName, change := range changes.Items{
		//delete(networkPolicyMap, namespacedName)
		//networkPolicyMap[namespacedName] = change.Current
		networkPolicyMap.unmerge(namespacedName, change.Previous)
		networkPolicyMap.merge(namespacedName, change.Current)
	}
	// clean change map here because we want to sync the whole networkPolicyMap
	changes.Items = make(map[types.NamespacedName]*NetworkPolicyChange)
}

func (npm *NetworkPolicyMap) merge(namespacedName types.NamespacedName, other *NetworkPolicyInfo){
	if other == nil{
		return
	}
	(*npm)[namespacedName] = other
}

func (npm *NetworkPolicyMap) unmerge(namespacedName types.NamespacedName, other *NetworkPolicyInfo){
	if other == nil{
		return
	}
	delete(*npm, namespacedName)
}

