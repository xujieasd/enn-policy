package config

import (
	api "k8s.io/api/core/v1"
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

type sortedPods []*api.Pod

func (s sortedPods) Len() int {
	return len(s)
}
func (s sortedPods) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortedPods) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

type PodHandlerMock struct {
	lock sync.Mutex

	state   map[types.NamespacedName]*api.Pod
	synced  bool
	updated chan []*api.Pod
	process func([]*api.Pod)
}

func NewPodHandlerMock() *PodHandlerMock {
	shm := &PodHandlerMock{
		state:   make(map[types.NamespacedName]*api.Pod),
		updated: make(chan []*api.Pod, 5),
	}
	shm.process = func(pods []*api.Pod) {
		shm.updated <- pods
	}
	return shm
}

func (h *PodHandlerMock) OnPodAdd(pod *api.Pod) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	h.state[namespacedName] = pod
	h.sendPods()
}

func (h *PodHandlerMock) OnPodUpdate(oldPod, pod *api.Pod) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	h.state[namespacedName] = pod
	h.sendPods()
}

func (h *PodHandlerMock) OnPodDelete(pod *api.Pod) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	delete(h.state, namespacedName)
	h.sendPods()
}

func (h *PodHandlerMock) OnPodSynced() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.synced = true
	h.sendPods()
}

func (h *PodHandlerMock) sendPods() {
	if !h.synced {
		return
	}
	pods := make([]*api.Pod, 0, len(h.state))
	for _, pod := range h.state {
		pods = append(pods, pod)
	}
	sort.Sort(sortedPods(pods))
	h.process(pods)
}

func (h *PodHandlerMock) ValidatePods(t *testing.T, expectedPods []*api.Pod) {
	// We might get 1 or more updates for N Pod updates, because we
	// over write older snapshots of Pods from the producer go-routine
	// if the consumer falls behind.
	var pods []*api.Pod
	for {
		select {
		case pods = <-h.updated:
			if reflect.DeepEqual(pods, expectedPods) {
				return
			}
		// Unittests will hard timeout in 5m with a stack trace, prevent that
		// and surface a clearer reason for failure.
		case <-time.After(wait.ForeverTestTimeout):
			t.Errorf("Timed out. Expected %#v, Got %#v", expectedPods, pods)
			return
		}
	}
}

func TestPodAddedAndNotified(t *testing.T) {

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
	client.PrependWatchReactor("pods", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewPodConfig(sharedInformers.Core().V1().Pods(), time.Minute)
	handler := NewPodHandlerMock()
	config.RegisterEventHandler(handler)
	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label := make(map[string]string)
	label["run1"] = "labeltest1"

	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testnamespace",
			Name: "foo",
			Labels: label,
		},
		Spec: api.PodSpec{
			NodeName: "node1",
		},
		Status: api.PodStatus{
			PodIP: "10.10.0.1",
		},
	}

	fakeWatch.Add(pod)
	handler.ValidatePods(t, []*api.Pod{pod})
}

func TestPodAddRemoveAndNotified(t *testing.T){

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
	client.PrependWatchReactor("pods", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewPodConfig(sharedInformers.Core().V1().Pods(), time.Minute)
	handler := NewPodHandlerMock()
	config.RegisterEventHandler(handler)
	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"

	pod1 := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testnamespace",
			Name: "foo1",
			Labels: label1,
		},
		Spec: api.PodSpec{
			NodeName: "node1",
		},
		Status: api.PodStatus{
			PodIP: "10.10.0.1",
		},
	}

	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"

	pod2 := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testnamespace",
			Name: "foo2",
			Labels: label2,
		},
		Spec: api.PodSpec{
			NodeName: "node2",
		},
		Status: api.PodStatus{
			PodIP: "10.10.0.2",
		},
	}

	fakeWatch.Add(pod1)
	handler.ValidatePods(t, []*api.Pod{pod1})

	fakeWatch.Add(pod2)
	handler.ValidatePods(t, []*api.Pod{pod1,pod2})

	fakeWatch.Delete(pod1)
	handler.ValidatePods(t, []*api.Pod{pod2})

}

func TestPodMultipleHandlerAddAndNotified(t *testing.T){

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
	client.PrependWatchReactor("pods", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewPodConfig(sharedInformers.Core().V1().Pods(), time.Minute)
	handler1 := NewPodHandlerMock()
	config.RegisterEventHandler(handler1)
	handler2 := NewPodHandlerMock()
	config.RegisterEventHandler(handler2)
	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"

	pod1 := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testnamespace",
			Name: "foo1",
			Labels: label1,
		},
		Spec: api.PodSpec{
			NodeName: "node1",
		},
		Status: api.PodStatus{
			PodIP: "10.10.0.1",
		},
	}

	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"

	pod2 := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testnamespace",
			Name: "foo2",
			Labels: label2,
		},
		Spec: api.PodSpec{
			NodeName: "node2",
		},
		Status: api.PodStatus{
			PodIP: "10.10.0.2",
		},
	}

	fakeWatch.Add(pod1)
	fakeWatch.Add(pod2)

	handler1.ValidatePods(t, []*api.Pod{pod1,pod2})
	handler2.ValidatePods(t, []*api.Pod{pod1,pod2})
}