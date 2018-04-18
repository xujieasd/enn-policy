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

type sortedNamespaces []*api.Namespace

func (s sortedNamespaces) Len() int {
	return len(s)
}
func (s sortedNamespaces) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortedNamespaces) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

type NamespaceHandlerMock struct {
	lock sync.Mutex

	state   map[types.NamespacedName]*api.Namespace
	synced  bool
	updated chan []*api.Namespace
	process func([]*api.Namespace)
}

func NewNamespaceHandlerMock() *NamespaceHandlerMock {
	shm := &NamespaceHandlerMock{
		state:   make(map[types.NamespacedName]*api.Namespace),
		updated: make(chan []*api.Namespace, 5),
	}
	shm.process = func(namespaces []*api.Namespace) {
		shm.updated <- namespaces
	}
	return shm
}

func (h *NamespaceHandlerMock) OnNamespaceAdd(namespace *api.Namespace) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: namespace.Namespace, Name: namespace.Name}
	h.state[namespacedName] = namespace
	h.sendNamespaces()
}

func (h *NamespaceHandlerMock) OnNamespaceUpdate(oldNamespace, namespace *api.Namespace) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: namespace.Namespace, Name: namespace.Name}
	h.state[namespacedName] = namespace
	h.sendNamespaces()
}

func (h *NamespaceHandlerMock) OnNamespaceDelete(namespace *api.Namespace) {
	h.lock.Lock()
	defer h.lock.Unlock()
	namespacedName := types.NamespacedName{Namespace: namespace.Namespace, Name: namespace.Name}
	delete(h.state, namespacedName)
	h.sendNamespaces()
}

func (h *NamespaceHandlerMock) OnNamespaceSynced() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.synced = true
	h.sendNamespaces()
}

func (h *NamespaceHandlerMock) sendNamespaces() {
	if !h.synced {
		return
	}
	namespaces := make([]*api.Namespace, 0, len(h.state))
	for _, namespace := range h.state {
		namespaces = append(namespaces, namespace)
	}
	sort.Sort(sortedNamespaces(namespaces))
	h.process(namespaces)
}

func (h *NamespaceHandlerMock) ValidateNamespaces(t *testing.T, expectedNamespaces []*api.Namespace) {
	// We might get 1 or more updates for N Namespace updates, because we
	// over write older snapshots of Namespaces from the producer go-routine
	// if the consumer falls behind.
	var namespaces []*api.Namespace
	for {
		select {
		case namespaces = <-h.updated:
			if reflect.DeepEqual(namespaces, expectedNamespaces) {
				return
			}
		// Unittests will hard timeout in 5m with a stack trace, prevent that
		// and surface a clearer reason for failure.
		case <-time.After(wait.ForeverTestTimeout):
			t.Errorf("Timed out. Expected %#v, Got %#v", expectedNamespaces, namespaces)
			return
		}
	}
}

func TestNamespaceAddedAndNotified(t *testing.T) {

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
	client.PrependWatchReactor("namespaces", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewNamespaceConfig(sharedInformers.Core().V1().Namespaces(), time.Minute)
	handler := NewNamespaceHandlerMock()
	config.RegisterEventHandler(handler)
	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label1 := make(map[string]string)
	label1["ns"] = "label1"

	namespace := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo1",
			Labels: label1,
		},
	}

	fakeWatch.Add(namespace)
	handler.ValidateNamespaces(t, []*api.Namespace{namespace})
}

func TestNamespaceAddRemoveAndNotified(t *testing.T){

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
	client.PrependWatchReactor("namespaces", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewNamespaceConfig(sharedInformers.Core().V1().Namespaces(), time.Minute)
	handler := NewNamespaceHandlerMock()
	config.RegisterEventHandler(handler)
	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label1 := make(map[string]string)
	label1["ns"] = "label1"

	namespace1 := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo1",
			Labels: label1,
		},
	}

	label2 := make(map[string]string)
	label2["ns"] = "label2"

	namespace2 := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo2",
			Labels: label2,
		},
	}

	fakeWatch.Add(namespace1)
	handler.ValidateNamespaces(t, []*api.Namespace{namespace1})

	fakeWatch.Add(namespace2)
	handler.ValidateNamespaces(t, []*api.Namespace{namespace1,namespace2})

	fakeWatch.Delete(namespace1)
	handler.ValidateNamespaces(t, []*api.Namespace{namespace2})
}

func TestNamespaceMultipleHandlerAddAndNotified(t *testing.T){

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
	client.PrependWatchReactor("namespaces", ktesting.DefaultWatchReactor(fakeWatch, nil))

	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformers := informers.NewSharedInformerFactory(client, time.Minute)

	config := NewNamespaceConfig(sharedInformers.Core().V1().Namespaces(), time.Minute)

	handler1 := NewNamespaceHandlerMock()
	config.RegisterEventHandler(handler1)
	handler2 := NewNamespaceHandlerMock()
	config.RegisterEventHandler(handler2)

	go sharedInformers.Start(stopCh)
	go config.Run(stopCh)

	label1 := make(map[string]string)
	label1["ns"] = "label1"

	namespace1 := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo1",
			Labels: label1,
		},
	}

	label2 := make(map[string]string)
	label2["ns"] = "label2"

	namespace2 := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo2",
			Labels: label2,
		},
	}

	fakeWatch.Add(namespace1)
	fakeWatch.Add(namespace2)

	handler1.ValidateNamespaces(t, []*api.Namespace{namespace1,namespace2})
	handler2.ValidateNamespaces(t, []*api.Namespace{namespace1,namespace2})
}
