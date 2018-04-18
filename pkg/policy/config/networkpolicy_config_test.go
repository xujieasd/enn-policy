package config

import (
	api "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/apimachinery/pkg/watch"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/apimachinery/pkg/util/wait"

	"testing"
	"time"
	"sync"
	"sort"
	"reflect"
)

type sortedNetworkPolicies []*api.NetworkPolicy

func (s sortedNetworkPolicies) Len() int {
	return len(s)
}
func (s sortedNetworkPolicies) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortedNetworkPolicies) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

type NetworkPolicyHandlerMock struct {
	lock      sync.Mutex

	state     map[types.NamespacedName]*api.NetworkPolicy
	synced    bool
	updated   chan []*api.NetworkPolicy
	process   func ([]*api.NetworkPolicy)
}

func NewNetworkPolicyHandlerMock() *NetworkPolicyHandlerMock {
	shm := &NetworkPolicyHandlerMock{
		state:   make(map[types.NamespacedName]*api.NetworkPolicy),
		updated: make(chan []*api.NetworkPolicy, 5),
	}
	shm.process = func(networkPolicys []*api.NetworkPolicy) {
		shm.updated <- networkPolicys
	}
	return shm
}

func (h *NetworkPolicyHandlerMock) OnNetworkPolicyAdd(networkPolicy *api.NetworkPolicy) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: networkPolicy.Namespace, Name: networkPolicy.Name}
	h.state[namespacedName] = networkPolicy
	h.sendNetworkPolicies()
}

func (h *NetworkPolicyHandlerMock) OnNetworkPolicyUpdate(oldNetworkPolicy, networkPolicy *api.NetworkPolicy) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: networkPolicy.Namespace, Name: networkPolicy.Name}
	h.state[namespacedName] = networkPolicy
	h.sendNetworkPolicies()
}

func (h *NetworkPolicyHandlerMock) OnNetworkPolicyDelete(networkPolicy *api.NetworkPolicy) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: networkPolicy.Namespace, Name: networkPolicy.Name}
	delete(h.state, namespacedName)
	h.sendNetworkPolicies()
}

func (h *NetworkPolicyHandlerMock) OnNetworkPolicySynced() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.synced = true
	h.sendNetworkPolicies()
}

func (h *NetworkPolicyHandlerMock) sendNetworkPolicies() {
	if !h.synced {
		return
	}
	networkPolicys := make([]*api.NetworkPolicy, 0, len(h.state))
	for _, policy := range h.state {
		networkPolicys = append(networkPolicys, policy)
	}
	sort.Sort(sortedNetworkPolicies(networkPolicys))
	h.process(networkPolicys)
}

func (h *NetworkPolicyHandlerMock) ValidateNetworkPolicies(t *testing.T, expectedNetworkPolicies []*api.NetworkPolicy) {

	var networkPolicies []*api.NetworkPolicy
	for {
		select {
		case networkPolicies = <-h.updated:
			if reflect.DeepEqual(networkPolicies,expectedNetworkPolicies){
				return
			}
		case <-time.After(wait.ForeverTestTimeout):
			t.Errorf("Timed out. Expected %#v, Got %#v", expectedNetworkPolicies, networkPolicies)
			return
		}
	}
}

func TestNetworkPolicyAddAndNotified(t *testing.T){

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
//	client.PrependWatchReactor("networkPolicies", ktesting.DefaultWatchReactor(fakeWatch, nil))
	client.PrependWatchReactor("*", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewNetworkPolicyConfig(sharedInformers.Networking().V1().NetworkPolicies(), time.Minute)
	handler := NewNetworkPolicyHandlerMock()
	config.RegisterEventHandler(handler)

	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"

	networkPolicy := &api.NetworkPolicy{
		ObjectMeta:  metav1.ObjectMeta{
			Namespace: "testns1",
			Name: "test1",
		},
		Spec:  api.NetworkPolicySpec{
			PolicyTypes:  []api.PolicyType{api.PolicyTypeIngress, api.PolicyTypeEgress},
			Ingress:  []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: label1}}},
				},
			},
			Egress: []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: label2}}},
				},
			},
		},
	}

	fakeWatch.Add(networkPolicy)
	handler.ValidateNetworkPolicies(t, []*api.NetworkPolicy{networkPolicy})

}

func TestNetWorkPolicyAddRemoveAndNotified(t *testing.T){

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
//	client.PrependWatchReactor("networkPolicies", ktesting.DefaultWatchReactor(fakeWatch, nil))
	client.PrependWatchReactor("*", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewNetworkPolicyConfig(sharedInformers.Networking().V1().NetworkPolicies(), time.Minute)
	handler := NewNetworkPolicyHandlerMock()
	config.RegisterEventHandler(handler)

	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"

	networkPolicy1 := &api.NetworkPolicy{
		ObjectMeta:  metav1.ObjectMeta{
			Namespace: "testns1",
			Name: "test1",
		},
		Spec:  api.NetworkPolicySpec{
			PolicyTypes:  []api.PolicyType{api.PolicyTypeIngress},
			Ingress:  []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: label1}}},
				},
			},
		},
	}

	networkPolicy2 := &api.NetworkPolicy{
		ObjectMeta:  metav1.ObjectMeta{
			Namespace: "testns２",
			Name: "test２",
		},
		Spec:  api.NetworkPolicySpec{
			PolicyTypes:  []api.PolicyType{api.PolicyTypeIngress},
			Egress: []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: label2}}},
				},
			},
		},
	}

	fakeWatch.Add(networkPolicy1)
	handler.ValidateNetworkPolicies(t,[]*api.NetworkPolicy{networkPolicy1})

	fakeWatch.Add(networkPolicy2)
	handler.ValidateNetworkPolicies(t,[]*api.NetworkPolicy{networkPolicy1,networkPolicy2})

	fakeWatch.Delete(networkPolicy1)
	handler.ValidateNetworkPolicies(t,[]*api.NetworkPolicy{networkPolicy2})
}

func TestNetworkPolicyMultipleHandlerAddAndNotified(t *testing.T){
	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
//	client.PrependWatchReactor("networkPolicies", ktesting.DefaultWatchReactor(fakeWatch, nil))
	client.PrependWatchReactor("*", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewNetworkPolicyConfig(sharedInformers.Networking().V1().NetworkPolicies(), time.Minute)

	handler1 := NewNetworkPolicyHandlerMock()
	config.RegisterEventHandler(handler1)
	handler2 := NewNetworkPolicyHandlerMock()
	config.RegisterEventHandler(handler2)

	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"

	networkPolicy1 := &api.NetworkPolicy{
		ObjectMeta:  metav1.ObjectMeta{
			Namespace: "testns1",
			Name: "test1",
		},
		Spec:  api.NetworkPolicySpec{
			PolicyTypes:  []api.PolicyType{api.PolicyTypeIngress},
			Ingress:  []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: label1}}},
				},
			},
		},
	}

	networkPolicy2 := &api.NetworkPolicy{
		ObjectMeta:  metav1.ObjectMeta{
			Namespace: "testns2",
			Name: "test2",
		},
		Spec:  api.NetworkPolicySpec{
			PolicyTypes:  []api.PolicyType{api.PolicyTypeIngress},
			Egress: []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: label2}}},
				},
			},
		},
	}

	fakeWatch.Add(networkPolicy1)
	fakeWatch.Add(networkPolicy2)

	handler1.ValidateNetworkPolicies(t,[]*api.NetworkPolicy{networkPolicy1,networkPolicy2})
	handler2.ValidateNetworkPolicies(t,[]*api.NetworkPolicy{networkPolicy1,networkPolicy2})
}