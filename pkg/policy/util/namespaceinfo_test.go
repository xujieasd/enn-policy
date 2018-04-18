package util

import (
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"reflect"
)

var nsName = []string{
	"test0",
	"test1",
	"test2",
	"test3",
}

var nsPodInfos = []*PodInfo{
	{//0
		Name: podName[0],
		Namespace: nsName[0],
		IP: podIP[0],
	},
	{//1
		Name: podName[1],
		Namespace: nsName[1],
		IP: podIP[1],
	},
	{//2
		Name: podName[2],
		Namespace: nsName[1],
		IP: podIP[2],
	},
	{//3
		Name: podName[3],
		Namespace: nsName[2],
		IP: podIP[3],
	},
	{//4
		Name: podName[4],
		Namespace: nsName[2],
		IP: podIP[4],
	},
	{//5
		Name: podName[5],
		Namespace: nsName[2],
		IP: podIP[5],
	},
	{//6
		Name: podName[6],
		Namespace: nsName[2],
		IP: podIP[6],
	},
	{//7
		Name: podName[7],
		Namespace: nsName[3],
		IP: podIP[7],
	},
}

func makeTestNamespace(name string, nsFunc func(*api.Namespace)) *api.Namespace{

	ns := &api.Namespace{
		ObjectMeta:  metav1.ObjectMeta{
			Name: name,
		},
		Spec:  api.NamespaceSpec{},
	}
	nsFunc(ns)
	return ns
}

// this test case will add some namespaces one by one
// and then check if their corresponding map is correct
func TestNamespaceInfoAdd(t *testing.T){

	namespaceMatchLabelMap := make(NamespaceMatchLabelMap)
	namespacePodMap := make(NamespacePodMap)
	namespaceInfoMap := make(NamespaceInfoMap)

	namespaceChanges := NewNamespaceChangeMap()

	nlabel1 := make(map[string]string)
	nlabel1["ns1"] = "p1"
	nlabel2 := make(map[string]string)
	nlabel2["ns2"] = "p2"
	nlabel3 := make(map[string]string)
	nlabel3["ns1"] = "p1"
	nlabel3["ns3"] = "p3"
	nlabel4 := make(map[string]string)
	nlabel4["ns3"] = "p3"
	nlabel4["ns4"] = "p4"

	labels := []Label{
		{LabelKey: "ns1", LabelValue:"p1"},
		{LabelKey: "ns2", LabelValue:"p2"},
		{LabelKey: "ns3", LabelValue:"p3"},
		{LabelKey: "ns4", LabelValue:"p4"},
	}

	podInfoMaps := []PodInfoMap{
		{//0    podInfoMap for namespace ns0
			podIP[0]: nsPodInfos[0],
		},
		{//1    podInfoMap for namespace ns1
			podIP[1]: nsPodInfos[1],
			podIP[2]: nsPodInfos[2],
		},
		{//2    podInfoMap for namespace ns2
			podIP[3]: nsPodInfos[3],
			podIP[4]: nsPodInfos[4],
			podIP[5]: nsPodInfos[5],
			podIP[6]: nsPodInfos[6],
		},
		{//3    podInfoMap for namespace ns3
			podIP[7]: nsPodInfos[7],
		},
		{//4    namespace ns1 and ns3 has label ns1=p1
			podIP[0]: nsPodInfos[0],
			podIP[3]: nsPodInfos[3],
			podIP[4]: nsPodInfos[4],
			podIP[5]: nsPodInfos[5],
			podIP[6]: nsPodInfos[6],
		},
		{//5    namespace ns3 and ns4 has label ns3=p3
			podIP[3]: nsPodInfos[3],
			podIP[4]: nsPodInfos[4],
			podIP[5]: nsPodInfos[5],
			podIP[6]: nsPodInfos[6],
			podIP[7]: nsPodInfos[7],
		},
	}

	// pod has some namespace will be in same podInfo Map
	namespacePodMap = NamespacePodMap{

		nsName[0]: podInfoMaps[0],
		nsName[1]: podInfoMaps[1],
		nsName[2]: podInfoMaps[2],
		nsName[3]: podInfoMaps[3],

	}

	namespaceInfos := []*NamespaceInfo{
		{
			Name: nsName[0],
			Labels: nlabel1,
		},
		{
			Name: nsName[1],
			Labels: nlabel2,
		},
		{
			Name: nsName[2],
			Labels: nlabel3,
		},
		{
			Name: nsName[3],
			Labels: nlabel4,
		},
	}

	expectedNIMap := []NamespaceInfoMap{
		{
			nsName[0]: namespaceInfos[0],
		},
		{
			nsName[0]: namespaceInfos[0],
			nsName[1]: namespaceInfos[1],
		},
		{
			nsName[0]: namespaceInfos[0],
			nsName[1]: namespaceInfos[1],
			nsName[2]: namespaceInfos[2],
		},
		{
			nsName[0]: namespaceInfos[0],
			nsName[1]: namespaceInfos[1],
			nsName[2]: namespaceInfos[2],
			nsName[3]: namespaceInfos[3],
		},

	}

	expectedNMLMap := []NamespaceMatchLabelMap{
		{
			labels[0]: podInfoMaps[0],
		},
		{
			labels[0]: podInfoMaps[0],
			labels[1]: podInfoMaps[1],
		},
		{
			labels[0]: podInfoMaps[4],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[2],
		},
		{
			labels[0]: podInfoMaps[4],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[5],
			labels[3]: podInfoMaps[3],
		},
	}

	namespaces := []*api.Namespace{
		makeTestNamespace(nsName[0], func(namespace *api.Namespace) {
			namespace.Labels = nlabel1
		}),
		makeTestNamespace(nsName[1], func(namespace *api.Namespace) {
			namespace.Labels = nlabel2
		}),
		makeTestNamespace(nsName[2], func(namespace *api.Namespace) {
			namespace.Labels = nlabel3
		}),
		makeTestNamespace(nsName[3], func(namespace *api.Namespace) {
			namespace.Labels = nlabel4
		}),
	}

	for i := 0; i < 4; i++ {
		namespaceName := types.NamespacedName{Namespace: namespaces[i].Namespace, Name: namespaces[i].Name}
		namespaceChanges.Update(&namespaceName, nil, namespaces[i])
		number := len(namespaceChanges.Items)
		if number != 1{
			t.Errorf("case %d namespace map change map pod len is %d, expected 1", i, number)
		}

		namespaceChanges.Lock.Lock()
		UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
		UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
		namespaceChanges.CleanUpItem()
		namespaceChanges.Lock.Unlock()

		ok := checkAllNamespaceMapValid(
			t,
			namespaceInfoMap, expectedNIMap[i],
			namespaceMatchLabelMap, expectedNMLMap[i],
		)
		if !ok{
			t.Errorf("case %d: check all namespace map failed", i)
		}
	}
}

// this test case have 4 parts
// 1. add some namespaces at once (means namespaceChangeMap will batch these changes)
// 2. delete these namespaces one by one
// 3. add these namespaces at once again
// 4. delete these namespaces at once
func TestNamespaceInfoAddDelete(t *testing.T){

	namespaceMatchLabelMap := make(NamespaceMatchLabelMap)
	namespacePodMap := make(NamespacePodMap)
	namespaceInfoMap := make(NamespaceInfoMap)

	namespaceChanges := NewNamespaceChangeMap()

	nlabel1 := make(map[string]string)
	nlabel1["ns1"] = "p1"
	nlabel2 := make(map[string]string)
	nlabel2["ns2"] = "p2"
	nlabel3 := make(map[string]string)
	nlabel3["ns1"] = "p1"
	nlabel3["ns3"] = "p3"
	nlabel4 := make(map[string]string)
	nlabel4["ns3"] = "p3"
	nlabel4["ns4"] = "p4"

	labels := []Label{
		{LabelKey: "ns1", LabelValue:"p1"},
		{LabelKey: "ns2", LabelValue:"p2"},
		{LabelKey: "ns3", LabelValue:"p3"},
		{LabelKey: "ns4", LabelValue:"p4"},
	}

	podInfoMaps := []PodInfoMap{
		{//0    podInfoMap for namespace ns0
			podIP[0]: nsPodInfos[0],
		},
		{//1    podInfoMap for namespace ns1
			podIP[1]: nsPodInfos[1],
			podIP[2]: nsPodInfos[2],
		},
		{//2    podInfoMap for namespace ns2
			podIP[3]: nsPodInfos[3],
			podIP[4]: nsPodInfos[4],
			podIP[5]: nsPodInfos[5],
			podIP[6]: nsPodInfos[6],
		},
		{//3    podInfoMap for namespace ns3
			podIP[7]: nsPodInfos[7],
		},
		{//4    namespace ns1 and ns3 has label ns1=p1
			podIP[0]: nsPodInfos[0],
			podIP[3]: nsPodInfos[3],
			podIP[4]: nsPodInfos[4],
			podIP[5]: nsPodInfos[5],
			podIP[6]: nsPodInfos[6],
		},
		{//5    namespace ns3 and ns4 has label ns3=p3
			podIP[3]: nsPodInfos[3],
			podIP[4]: nsPodInfos[4],
			podIP[5]: nsPodInfos[5],
			podIP[6]: nsPodInfos[6],
			podIP[7]: nsPodInfos[7],
		},
	}
	emptyPodInfoMap := make(PodInfoMap)

	// pod has some namespace will be in same podInfo Map
	namespacePodMap = NamespacePodMap{

		nsName[0]: podInfoMaps[0],
		nsName[1]: podInfoMaps[1],
		nsName[2]: podInfoMaps[2],
		nsName[3]: podInfoMaps[3],
	}

	namespaceInfos := []*NamespaceInfo{
		{
			Name: nsName[0],
			Labels: nlabel1,
		},
		{
			Name: nsName[1],
			Labels: nlabel2,
		},
		{
			Name: nsName[2],
			Labels: nlabel3,
		},
		{
			Name: nsName[3],
			Labels: nlabel4,
		},
	}

	//emptyNamespaceInfo := &NamespaceInfo{}

	expectedNIMap := []NamespaceInfoMap{
		{//0
			nsName[0]: namespaceInfos[0],
			nsName[1]: namespaceInfos[1],
			nsName[2]: namespaceInfos[2],
			nsName[3]: namespaceInfos[3],
		},
		{//1
			nsName[1]: namespaceInfos[1],
			nsName[2]: namespaceInfos[2],
			nsName[3]: namespaceInfos[3],
		},
		{//2
			nsName[2]: namespaceInfos[2],
			nsName[3]: namespaceInfos[3],
		},
		{//3
			nsName[3]: namespaceInfos[3],
		},
		{//4 empty

		},

	}

	expectedNMLMap := []NamespaceMatchLabelMap{
		{//0
			labels[0]: podInfoMaps[4],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[5],
			labels[3]: podInfoMaps[3],
		},
		{//1
			labels[0]: podInfoMaps[2],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[5],
			labels[3]: podInfoMaps[3],
		},
		{//2
			labels[0]: podInfoMaps[2],
			labels[1]: emptyPodInfoMap,
			labels[2]: podInfoMaps[5],
			labels[3]: podInfoMaps[3],
		},
		{//3
			labels[0]: emptyPodInfoMap,
			labels[1]: emptyPodInfoMap,
			labels[2]: podInfoMaps[3],
			labels[3]: podInfoMaps[3],
		},
		{//4
			labels[0]: emptyPodInfoMap,
			labels[1]: emptyPodInfoMap,
			labels[2]: emptyPodInfoMap,
			labels[3]: emptyPodInfoMap,
		},
	}

	namespaces := []*api.Namespace{
		makeTestNamespace(nsName[0], func(namespace *api.Namespace) {
			namespace.Labels = nlabel1
		}),
		makeTestNamespace(nsName[1], func(namespace *api.Namespace) {
			namespace.Labels = nlabel2
		}),
		makeTestNamespace(nsName[2], func(namespace *api.Namespace) {
			namespace.Labels = nlabel3
		}),
		makeTestNamespace(nsName[3], func(namespace *api.Namespace) {
			namespace.Labels = nlabel4
		}),
	}

	// 1. first add 4 namespace at once
	for i := 0; i < 4; i++ {
		namespaceName := types.NamespacedName{Namespace: namespaces[i].Namespace, Name: namespaces[i].Name}
		namespaceChanges.Update(&namespaceName, nil, namespaces[i])
	}

	number := len(namespaceChanges.Items)
	if number != 4{
		t.Errorf("init add: namespace map change map pod len is %d, expected 4", number)
	}

	namespaceChanges.Lock.Lock()
	UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
	UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
	namespaceChanges.CleanUpItem()
	namespaceChanges.Lock.Unlock()

	ok := checkAllNamespaceMapValid(
		t,
		namespaceInfoMap, expectedNIMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("init add: check all namespace map failed")
	}

	// 2. delete namespace one by one
	for i := 0; i < 4; i++ {
		namespaceName := types.NamespacedName{Namespace: namespaces[i].Namespace, Name: namespaces[i].Name}
		namespaceChanges.Update(&namespaceName, namespaces[i], nil)

		number := len(namespaceChanges.Items)
		if number != 1{
			t.Errorf("case %d delete: namespace map change map pod len is %d, expected 1", i, number)
		}

		namespaceChanges.Lock.Lock()
		UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
		UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
		namespaceChanges.CleanUpItem()
		namespaceChanges.Lock.Unlock()

		ok := checkAllNamespaceMapValid(
			t,
			namespaceInfoMap, expectedNIMap[i+1],
			namespaceMatchLabelMap, expectedNMLMap[i+1],
		)
		if !ok{
			t.Errorf("case %d delete: check all namespace map failed",i)
		}
	}

	// 3. second time add namespace at once
	for i := 0; i < 4; i++ {
		namespaceName := types.NamespacedName{Namespace: namespaces[i].Namespace, Name: namespaces[i].Name}
		namespaceChanges.Update(&namespaceName, nil, namespaces[i])
	}

	number = len(namespaceChanges.Items)
	if number != 4{
		t.Errorf("second time init add: namespace map change map pod len is %d, expected 4", number)
	}

	namespaceChanges.Lock.Lock()
	UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
	UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
	namespaceChanges.CleanUpItem()
	namespaceChanges.Lock.Unlock()

	ok = checkAllNamespaceMapValid(
		t,
		namespaceInfoMap, expectedNIMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("second time init add: check all namespace map failed")
	}

	// 4. delete all namespace at once
	for i := 0; i < 4; i++ {
		namespaceName := types.NamespacedName{Namespace: namespaces[i].Namespace, Name: namespaces[i].Name}
		namespaceChanges.Update(&namespaceName, namespaces[i], nil)
	}

	number = len(namespaceChanges.Items)
	if number != 4{
		t.Errorf("delete all: namespace map change map pod len is %d, expected 4", number)
	}

	namespaceChanges.Lock.Lock()
	UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
	UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
	namespaceChanges.CleanUpItem()
	namespaceChanges.Lock.Unlock()

	ok = checkAllNamespaceMapValid(
		t,
		namespaceInfoMap, expectedNIMap[4],
		namespaceMatchLabelMap, expectedNMLMap[4],
	)
	if !ok{
		t.Errorf("delete all: check all namespace map failed")
	}
}

// this test case have 4 parts
// 1. update namespaces without change, so corresponding map should keep as the same
// 2. update labels of namespaces which labels are exist in corresponding map
// 3. update labels of namespaces which will add new namespaces labels
// 4. update namespaces to the initial one, so corresponding map should as same as step 1
func TestNamespaceInfoUpdate(t *testing.T) {

	namespaceMatchLabelMap := make(NamespaceMatchLabelMap)
	namespacePodMap := make(NamespacePodMap)
	namespaceInfoMap := make(NamespaceInfoMap)

	namespaceChanges := NewNamespaceChangeMap()

	nlabel1 := make(map[string]string)
	nlabel1["ns1"] = "p1"
	nlabel2 := make(map[string]string)
	nlabel2["ns1"] = "p1"
	nlabel2["ns2"] = "p2"
	nlabel3 := make(map[string]string)
	nlabel3["ns2"] = "p2"
	nlabel3["ns3"] = "p3"
	nlabel3["ns4"] = "p4"

	labels := []Label{
		{LabelKey: "ns1", LabelValue:"p1"},
		{LabelKey: "ns2", LabelValue:"p2"},
		{LabelKey: "ns3", LabelValue:"p3"},
		{LabelKey: "ns4", LabelValue:"p4"},
	}

	podInfoMaps := []PodInfoMap{
		{//0    podInfoMap for namespace ns0
			podIP[0]: nsPodInfos[0],
		},
		{//1    podInfoMap for namespace ns1
			podIP[1]: nsPodInfos[1],
			podIP[2]: nsPodInfos[2],
		},
		{//2    podInfoMap for namespace ns2
			podIP[3]: nsPodInfos[3],
			podIP[4]: nsPodInfos[4],
			podIP[5]: nsPodInfos[5],
			podIP[6]: nsPodInfos[6],
		},
		{//3    podInfoMap for namespace ns3
			podIP[7]: nsPodInfos[7],
		},
		{//4    podInfoMap for namespace ns1 and ns2
			podIP[1]: nsPodInfos[1],
			podIP[2]: nsPodInfos[2],
			podIP[3]: nsPodInfos[3],
			podIP[4]: nsPodInfos[4],
			podIP[5]: nsPodInfos[5],
			podIP[6]: nsPodInfos[6],
		},
	}
	emptyPodInfoMap := make(PodInfoMap)

	// pod has some namespace will be in same podInfo Map
	namespacePodMap = NamespacePodMap{

		nsName[0]: podInfoMaps[0],
		nsName[1]: podInfoMaps[1],
		nsName[2]: podInfoMaps[2],
		nsName[3]: podInfoMaps[3],
	}

	namespaceInfos := []*NamespaceInfo{
		{//0
			Name: nsName[1],
			Labels: nlabel1,
		},
		{//1
			Name: nsName[2],
			Labels: nlabel2,
		},
		{//2
			Name: nsName[1],
			Labels: nlabel2,
		},
		{//3
			Name: nsName[1],
			Labels: nlabel3,
		},
	}

	expectedNIMap := []NamespaceInfoMap{
		{//0
			nsName[1]: namespaceInfos[0],
			nsName[2]: namespaceInfos[1],
		},
		{//1
			nsName[1]: namespaceInfos[2],
			nsName[2]: namespaceInfos[1],
		},
		{//2
			nsName[1]: namespaceInfos[3],
			nsName[2]: namespaceInfos[1],
		},
	}

	expectedNMLMap := []NamespaceMatchLabelMap{
		{//0
			labels[0]: podInfoMaps[4],
			labels[1]: podInfoMaps[2],
		},
		{//1
			labels[0]: podInfoMaps[4],
			labels[1]: podInfoMaps[4],
		},
		{//2
			labels[0]: podInfoMaps[2],
			labels[1]: podInfoMaps[4],
			labels[2]: podInfoMaps[1],
			labels[3]: podInfoMaps[1],
		},
		{//4
			labels[0]: podInfoMaps[4],
			labels[1]: podInfoMaps[2],
			labels[2]: emptyPodInfoMap,
			labels[3]: emptyPodInfoMap,
		},
	}

	namespaces := []*api.Namespace{
		//0
		makeTestNamespace(nsName[1], func(namespace *api.Namespace) {
			namespace.Labels = nlabel1
		}),
		//1
		makeTestNamespace(nsName[2], func(namespace *api.Namespace) {
			namespace.Labels = nlabel2
		}),
		//2
		makeTestNamespace(nsName[1], func(namespace *api.Namespace) {
			namespace.Labels = nlabel2
		}),
		//3
		makeTestNamespace(nsName[1], func(namespace *api.Namespace) {
			namespace.Labels = nlabel3
		}),
	}

	// 0. add 2 namespaces
	for i := 0; i < 2; i++ {
		namespaceName := types.NamespacedName{Namespace: namespaces[i].Namespace, Name: namespaces[i].Name}
		namespaceChanges.Update(&namespaceName, nil, namespaces[i])
	}

	number := len(namespaceChanges.Items)
	if number != 2{
		t.Errorf("init add: namespace map change map pod len is %d, expected 2", number)
	}

	namespaceChanges.Lock.Lock()
	UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
	UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
	namespaceChanges.CleanUpItem()
	namespaceChanges.Lock.Unlock()

	ok := checkAllNamespaceMapValid(
		t,
		namespaceInfoMap, expectedNIMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("init add: check all namespace map failed")
	}

	// 1. update namespaces without change
	namespaceName := types.NamespacedName{Namespace: namespaces[0].Namespace, Name: namespaces[0].Name}
	namespaceChanges.Update(&namespaceName, namespaces[0], namespaces[0])


	number = len(namespaceChanges.Items)
	if number != 0{
		t.Errorf("update namespaces without change: namespace map change map pod len is %d, expected 0", number)
	}

	namespaceChanges.Lock.Lock()
	UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
	UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
	namespaceChanges.CleanUpItem()
	namespaceChanges.Lock.Unlock()

	ok = checkAllNamespaceMapValid(
		t,
		namespaceInfoMap, expectedNIMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("update namespaces without change: check all namespace map failed")
	}

	// 2. update namespaces with exist label
	namespaceName = types.NamespacedName{Namespace: namespaces[2].Namespace, Name: namespaces[2].Name}
	namespaceChanges.Update(&namespaceName, namespaces[0], namespaces[2])


	number = len(namespaceChanges.Items)
	if number != 1{
		t.Errorf("update namespaces with exist label: namespace map change map pod len is %d, expected 1", number)
	}

	namespaceChanges.Lock.Lock()
	UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
	UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
	namespaceChanges.CleanUpItem()
	namespaceChanges.Lock.Unlock()

	ok = checkAllNamespaceMapValid(
		t,
		namespaceInfoMap, expectedNIMap[1],
		namespaceMatchLabelMap, expectedNMLMap[1],
	)
	if !ok{
		t.Errorf("update namespaces with exist label: check all namespace map failed")
	}

	// 3. update namespaces with new label
	namespaceName = types.NamespacedName{Namespace: namespaces[3].Namespace, Name: namespaces[3].Name}
	namespaceChanges.Update(&namespaceName, namespaces[2], namespaces[3])


	number = len(namespaceChanges.Items)
	if number != 1{
		t.Errorf("update namespaces with new label: namespace map change map pod len is %d, expected 1", number)
	}

	namespaceChanges.Lock.Lock()
	UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
	UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
	namespaceChanges.CleanUpItem()
	namespaceChanges.Lock.Unlock()

	ok = checkAllNamespaceMapValid(
		t,
		namespaceInfoMap, expectedNIMap[2],
		namespaceMatchLabelMap, expectedNMLMap[2],
	)
	if !ok{
		t.Errorf("update namespaces with new label: check all namespace map failed")
	}

	// 4. update back to initial
	namespaceName = types.NamespacedName{Namespace: namespaces[0].Namespace, Name: namespaces[0].Name}
	namespaceChanges.Update(&namespaceName, namespaces[3], namespaces[0])


	number = len(namespaceChanges.Items)
	if number != 1{
		t.Errorf("update back to initial: namespace map change map pod len is %d, expected 1", number)
	}

	namespaceChanges.Lock.Lock()
	UpdateNamespaceMatchLabelMap(namespaceMatchLabelMap, namespacePodMap, &namespaceChanges)
	UpdateNamespaceInfoMap(namespaceInfoMap, &namespaceChanges)
	namespaceChanges.CleanUpItem()
	namespaceChanges.Lock.Unlock()

	ok = checkAllNamespaceMapValid(
		t,
		namespaceInfoMap, expectedNIMap[0],
		namespaceMatchLabelMap, expectedNMLMap[3],
	)
	if !ok{
		t.Errorf("update back to initial: check all namespace map failed")
	}
}

func checkAllNamespaceMapValid(t *testing.T,
	niMap, expectedNIMap NamespaceInfoMap,
	nmlMapm, expectedNMLMap NamespaceMatchLabelMap,
) bool{
	var result = true
	ok := checkNamespaceInfoMapValid(t, niMap, expectedNIMap)
	if !ok{
		t.Errorf("invalid podMatchLabelMap")
		result = false
	}

	ok = checkNamespaceMatchLabelMapValid(t, nmlMapm, expectedNMLMap)
	if !ok{
		t.Errorf("invalid namespaceMatchLabelMap")
		result = false
	}

	return result
}

func checkNamespaceInfoMapValid(t *testing.T, niMap, expectedNIMap NamespaceInfoMap) bool{

	if len(niMap) != len(expectedNIMap){
		t.Errorf("length of NamespaceInfoMap is not correct %d, expected %s", len(niMap), len(expectedNIMap))
		for name := range niMap{
			t.Errorf("name of namespaceInfoMap: %s", name)
		}
		for name := range expectedNIMap{
			t.Errorf("expected name of namespaceInfoMap: %s", name)
		}
		return false
	}

	for exName, exNamespaceInfoMap := range expectedNIMap{
		namespaceInfoMap, ok := niMap[exName]
		if !ok{
			t.Errorf("canot find expected name of namespaceInfoMap %s", exName)
			for name := range niMap{
				t.Errorf("name of namespaceInfoMap: %s", name)
			}
			return false
		}
		if namespaceInfoMap.Name != exNamespaceInfoMap.Name{
			t.Errorf("namespaceInfo name is not corrected %s, expected %s", namespaceInfoMap.Name, exNamespaceInfoMap.Name)
			return false
		}
		if !reflect.DeepEqual(namespaceInfoMap.Labels, exNamespaceInfoMap.Labels){
			t.Errorf("namespace %s is not correct", exName)
			for k,v := range namespaceInfoMap.Labels{
				t.Errorf("label is %s=%s", k, v)
			}
			for k,v := range exNamespaceInfoMap.Labels{
				t.Errorf("expecet label is %s=%s", k, v)
			}
			return false
		}
	}

	return true
}