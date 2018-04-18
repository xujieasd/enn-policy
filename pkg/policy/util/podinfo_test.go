package util

import (
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

var podNamespace = []string{
	"testns1",
	"testns2",
	"testns3",
	"testns4",
	"testns5",
	"testns6",
	"testns7",
	"testns8",
}
var podName = []string{
	"testpod1",
	"testpod2",
	"testpod3",
	"testpod4",
	"testpod5",
	"testpod6",
	"testpod7",
	"testpod8",
}

var podIP = []string{
	"10.0.1.1",
	"10.0.1.2",
	"10.0.1.3",
	"10.0.1.4",
	"10.0.2.1",
	"10.0.2.2",
	"10.0.2.3",
	"10.0.3.1",
}

func makeTestPod(namespace, name string, podFunc func(*api.Pod)) *api.Pod{

	pod := &api.Pod{
		ObjectMeta:  metav1.ObjectMeta{
			Namespace: namespace,
			Name: name,
		},
		Spec:  api.PodSpec{},
	}
	podFunc(pod)
	return pod
}

// this test case will add some pods one by one
// and then check if their corresponding map is correct
func TestPodInfoAdd(t *testing.T){

	podMatchLabelMap := make(PodMatchLabelMap)
	namespacePodMap := make(NamespacePodMap)
	namespaceMatchLabelMap := make(NamespaceMatchLabelMap)
	namespaceInfoMap := make(NamespaceInfoMap)
	podChanges := NewPodLabelChangeMap()

	label1 := make(map[string]string)
	label1["pod1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["pod2"] = "labeltest2"
	label2["pod3"] = "labeltest3"
	label3 := make(map[string]string)
	label3["pod2"] = "labeltest2"
	label3["pod3"] = "labeltest3"
	label4 := make(map[string]string)
	label4["pod1"] = "labeltest1"
	label4["pod2"] = "labeltest2"
	label4["pod3"] = "labeltest3"
	label4["pod4"] = "labeltest4"

	nlabel1 := make(map[string]string)
	nlabel1["ns1"] = "p1"
	nlabel2 := make(map[string]string)
	nlabel2["ns2"] = "p2"
	nlabel3 := make(map[string]string)
	nlabel3["ns1"] = "p1"
	nlabel3["ns3"] = "p3"

	namespaceInfos := []*NamespaceInfo{
		{
			Name: podNamespace[0],
			Labels: nlabel1,
		},
		{
			Name: podNamespace[1],
			Labels: nlabel2,
		},
		{
			Name: podNamespace[2],
			Labels: nlabel3,
		},
	}

	namespaceInfoMap = NamespaceInfoMap{
		podNamespace[0]: namespaceInfos[0],
		podNamespace[1]: namespaceInfos[1],
		podNamespace[2]: namespaceInfos[2],
	}

	namespacedLabels := []NamespacedLabel{
		{Namespace: podNamespace[0], LabelKey:"pod1", LabelValue:"labeltest1"}, //0
		{Namespace: podNamespace[1], LabelKey:"pod2", LabelValue:"labeltest2"}, //1
		{Namespace: podNamespace[1], LabelKey:"pod3", LabelValue:"labeltest3"}, //2
		{Namespace: podNamespace[2], LabelKey:"pod2", LabelValue:"labeltest2"}, //3
		{Namespace: podNamespace[2], LabelKey:"pod3", LabelValue:"labeltest3"}, //4
		{Namespace: podNamespace[0], LabelKey:"pod2", LabelValue:"labeltest2"}, //5
		{Namespace: podNamespace[0], LabelKey:"pod3", LabelValue:"labeltest3"}, //6
		{Namespace: podNamespace[0], LabelKey:"pod4", LabelValue:"labeltest4"}, //7
		{Namespace: podNamespace[2], LabelKey:"pod1", LabelValue:"labeltest1"}, //8
	}

	labels := []Label{
		{LabelKey: "ns1", LabelValue:"p1"},
		{LabelKey: "ns2", LabelValue:"p2"},
		{LabelKey: "ns3", LabelValue:"p3"},
	}

	podInfos := []*PodInfo{
		{//0
		 	//Labels pod1= labeltest1
			Name: podName[0],
			Namespace: podNamespace[0],
			IP: podIP[0],
			Labels: label1,
		},
		{//1
			//Labels pod2= labeltest2
			//Labels pod3= labeltest3
			Name: podName[1],
			Namespace: podNamespace[1],
			IP: podIP[1],
			Labels: label2,
		},
		{//2
			//Labels pod2= labeltest2
			//Labels pod3= labeltest3
			Name: podName[2],
			Namespace: podNamespace[2],
			IP: podIP[2],
			Labels: label2,
		},
		{//3
			//Labels pod1= labeltest1
			Name: podName[3],
			Namespace: podNamespace[0],
			IP: podIP[3],
			Labels: label1,
		},
		{//4
			//Labels pod1= labeltest1
			//Labels pod2= labeltest2
			//Labels pod3= labeltest3
			//Labels pod4= labeltest4
			Name: podName[4],
			Namespace: podNamespace[0],
			IP: podIP[4],
			Labels: label4,
		},
		{//5
			//Labels pod1= labeltest1
			Name: podName[5],
			Namespace: podNamespace[2],
			IP: podIP[5],
			Labels: label1,
		},
	}

	podInfoMaps := []PodInfoMap{
		{//0
			podIP[0]: podInfos[0],
		},
		{//1
			podIP[1]: podInfos[1],
		},
		{//2
			podIP[2]: podInfos[2],
		},
		{//3
			podIP[0]: podInfos[0],
			podIP[3]: podInfos[3],
		},
		{//4    pod in same namespace testns1 with same label pod1= labeltest1
			podIP[0]: podInfos[0],
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
		},
		{//5
			podIP[4]: podInfos[4],
		},
		{//6
			podIP[5]: podInfos[5],
		},
		{//7    pod in same namespace testns3
			podIP[2]: podInfos[2],
			podIP[5]: podInfos[5],
		},
		{//8
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
		},
		{//9
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
			podIP[3]: podInfos[3],
		},
		{//10
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
		},
		{//11   pod in same namespace label ns1=p1
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
			podIP[5]: podInfos[5],
		},
	}

	expectedPMLMap := []PodMatchLabelMap{
		{//0
			namespacedLabels[0]: podInfoMaps[0],
		},
		{//1
			namespacedLabels[0]: podInfoMaps[0],
			namespacedLabels[1]: podInfoMaps[1],
			namespacedLabels[2]: podInfoMaps[1],
		},
		{//2
			namespacedLabels[0]: podInfoMaps[0],
			namespacedLabels[1]: podInfoMaps[1],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[2],
			namespacedLabels[4]: podInfoMaps[2],
		},
		{//3
			namespacedLabels[0]: podInfoMaps[3],
			namespacedLabels[1]: podInfoMaps[1],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[2],
			namespacedLabels[4]: podInfoMaps[2],
		},
		{//4
			namespacedLabels[0]: podInfoMaps[4],
			namespacedLabels[1]: podInfoMaps[1],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[2],
			namespacedLabels[4]: podInfoMaps[2],
			namespacedLabels[5]: podInfoMaps[5],
			namespacedLabels[6]: podInfoMaps[5],
			namespacedLabels[7]: podInfoMaps[5],
		},
		{//5
			namespacedLabels[0]: podInfoMaps[4],
			namespacedLabels[1]: podInfoMaps[1],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[2],
			namespacedLabels[4]: podInfoMaps[2],
			namespacedLabels[5]: podInfoMaps[5],
			namespacedLabels[6]: podInfoMaps[5],
			namespacedLabels[7]: podInfoMaps[5],
			namespacedLabels[8]: podInfoMaps[6],
		},

	}

	expectedNPMap := []NamespacePodMap{
		{//0
			podNamespace[0] : podInfoMaps[0],
		},
		{//1
			podNamespace[0] : podInfoMaps[0],
			podNamespace[1] : podInfoMaps[1],
		},
		{//2
			podNamespace[0] : podInfoMaps[0],
			podNamespace[1] : podInfoMaps[1],
			podNamespace[2] : podInfoMaps[2],
		},
		{//3
			podNamespace[0] : podInfoMaps[3],
			podNamespace[1] : podInfoMaps[1],
			podNamespace[2] : podInfoMaps[2],
		},
		{//4
			podNamespace[0] : podInfoMaps[4],
			podNamespace[1] : podInfoMaps[1],
			podNamespace[2] : podInfoMaps[2],
		},
		{//5
			podNamespace[0] : podInfoMaps[4],
			podNamespace[1] : podInfoMaps[1],
			podNamespace[2] : podInfoMaps[7],
		},
	}

	expectedNMLMap := []NamespaceMatchLabelMap{
		{//0
			labels[0]: podInfoMaps[0],
		},
		{//1
			labels[0]: podInfoMaps[0],
			labels[1]: podInfoMaps[1],
		},
		{//2
			labels[0]: podInfoMaps[8],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[2],
		},
		{//3
			labels[0]: podInfoMaps[9],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[2],
		},
		{//4
			labels[0]: podInfoMaps[10],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[2],
		},
		{//5
			labels[0]: podInfoMaps[11],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[7],
		},
	}

	pods := []*api.Pod{
		// single pod single label
		makeTestPod(podNamespace[0], podName[0], func(pod *api.Pod) {
			pod.Labels = label1
			pod.Status.PodIP = podIP[0]
		}),
		// single pod muti label
		makeTestPod(podNamespace[1], podName[1], func(pod *api.Pod) {
			pod.Labels = label2
			pod.Status.PodIP = podIP[1]
		}),
		// different namespace with same label
		makeTestPod(podNamespace[2], podName[2], func(pod *api.Pod) {
			pod.Labels = label2
			pod.Status.PodIP = podIP[2]
		}),
		// add new pod to exist namespace, same label
		makeTestPod(podNamespace[0], podName[3], func(pod *api.Pod) {
			pod.Labels = label1
			pod.Status.PodIP = podIP[3]
		}),
		// add new pod to exist namespace, different label
		makeTestPod(podNamespace[0], podName[4], func(pod *api.Pod) {
			pod.Labels = label4
			pod.Status.PodIP = podIP[4]
		}),
		makeTestPod(podNamespace[2], podName[5], func(pod *api.Pod) {
			pod.Labels = label1
			pod.Status.PodIP = podIP[5]
		}),
	}

	for i := 0; i < 6; i++ {
		namespaceName := types.NamespacedName{Namespace: pods[i].Namespace, Name: pods[i].Name}
		podChanges.Update(&namespaceName, nil, pods[i])

		ok := checkAllPodChangeLenValid(t, podChanges, 1, 1, 1)
		if !ok{
			t.Errorf("case %d check pod change len faild",i)
		}

		podChanges.Lock.Lock()
		UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
		UpdateNamespacePodMap(namespacePodMap, &podChanges)
		UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

		podChanges.CleanUpItem()
		podChanges.Lock.Unlock()

		ok = checkAllPodMapValid(
			t,
			podMatchLabelMap, expectedPMLMap[i],
			namespacePodMap, expectedNPMap[i],
			namespaceMatchLabelMap, expectedNMLMap[i],
		)
		if !ok{
			t.Errorf("case %d check all pod faild",i)
		}
	}
}

// this test case have 4 parts
// 1. add some pods at once (means podChangeMap will batch these changes)
// 2. delete these pods one by one
// 3. add these pods at once again
// 4. delete these pods at once
func TestPodInfoAddDelete(t *testing.T){

	podMatchLabelMap := make(PodMatchLabelMap)
	namespacePodMap := make(NamespacePodMap)
	namespaceMatchLabelMap := make(NamespaceMatchLabelMap)
	namespaceInfoMap := make(NamespaceInfoMap)
	podChanges := NewPodLabelChangeMap()

	label1 := make(map[string]string)
	label1["pod1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["pod2"] = "labeltest2"
	label2["pod3"] = "labeltest3"
	label3 := make(map[string]string)
	label3["pod2"] = "labeltest2"
	label3["pod3"] = "labeltest3"
	label4 := make(map[string]string)
	label4["pod1"] = "labeltest1"
	label4["pod2"] = "labeltest2"
	label4["pod3"] = "labeltest3"
	label4["pod4"] = "labeltest4"

	nlabel1 := make(map[string]string)
	nlabel1["ns1"] = "p1"
	nlabel2 := make(map[string]string)
	nlabel2["ns2"] = "p2"
	nlabel3 := make(map[string]string)
	nlabel3["ns1"] = "p1"
	nlabel3["ns3"] = "p3"

	labels := []Label{
		{LabelKey: "ns1", LabelValue:"p1"},
		{LabelKey: "ns2", LabelValue:"p2"},
		{LabelKey: "ns3", LabelValue:"p3"},
	}

	namespaceInfos := []*NamespaceInfo{
		{
			Name: podNamespace[0],
			Labels: nlabel1,
		},
		{
			Name: podNamespace[1],
			Labels: nlabel2,
		},
		{
			Name: podNamespace[2],
			Labels: nlabel3,
		},
	}

	namespaceInfoMap = NamespaceInfoMap{
		podNamespace[0]: namespaceInfos[0],
		podNamespace[1]: namespaceInfos[1],
		podNamespace[2]: namespaceInfos[2],
	}

	namespacedLabels := []NamespacedLabel{
		{Namespace: podNamespace[0], LabelKey:"pod1", LabelValue:"labeltest1"}, //0
		{Namespace: podNamespace[1], LabelKey:"pod2", LabelValue:"labeltest2"}, //1
		{Namespace: podNamespace[1], LabelKey:"pod3", LabelValue:"labeltest3"}, //2
		{Namespace: podNamespace[2], LabelKey:"pod2", LabelValue:"labeltest2"}, //3
		{Namespace: podNamespace[2], LabelKey:"pod3", LabelValue:"labeltest3"}, //4
		{Namespace: podNamespace[0], LabelKey:"pod2", LabelValue:"labeltest2"}, //5
		{Namespace: podNamespace[0], LabelKey:"pod3", LabelValue:"labeltest3"}, //6
		{Namespace: podNamespace[0], LabelKey:"pod4", LabelValue:"labeltest4"}, //7
		{Namespace: podNamespace[2], LabelKey:"pod1", LabelValue:"labeltest1"}, //8
	}

	podInfos := []*PodInfo{
		{//0
			//Labels pod1= labeltest1
			Name: podName[0],
			Namespace: podNamespace[0],
			IP: podIP[0],
			Labels: label1,
		},
		{//1
			//Labels pod2= labeltest2
			//Labels pod3= labeltest3
			Name: podName[1],
			Namespace: podNamespace[1],
			IP: podIP[1],
			Labels: label2,
		},
		{//2
			//Labels pod2= labeltest2
			//Labels pod3= labeltest3
			Name: podName[2],
			Namespace: podNamespace[2],
			IP: podIP[2],
			Labels: label2,
		},
		{//3
			//Labels pod1= labeltest1
			Name: podName[3],
			Namespace: podNamespace[0],
			IP: podIP[3],
			Labels: label1,
		},
		{//4
			//Labels pod1= labeltest1
			//Labels pod2= labeltest2
			//Labels pod3= labeltest3
			//Labels pod4= labeltest4
			Name: podName[4],
			Namespace: podNamespace[0],
			IP: podIP[4],
			Labels: label4,
		},
		{//5
			//Labels pod1= labeltest1
			Name: podName[5],
			Namespace: podNamespace[2],
			IP: podIP[5],
			Labels: label1,
		},
	}

	podInfoMaps := []PodInfoMap{
		{//0
			podIP[0]: podInfos[0],
		},
		{//1
			podIP[1]: podInfos[1],
		},
		{//2
			podIP[2]: podInfos[2],
		},
		{//3
			podIP[0]: podInfos[0],
			podIP[3]: podInfos[3],
		},
		{//4    pod in same namespace testns1 with same label pod1= labeltest1
			podIP[0]: podInfos[0],
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
		},
		{//5
			podIP[4]: podInfos[4],
		},
		{//6
			podIP[5]: podInfos[5],
		},
		{//7    pod in same namespace testns3
			podIP[2]: podInfos[2],
			podIP[5]: podInfos[5],
		},
		{//8
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
		},
		{//9
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
			podIP[3]: podInfos[3],
		},
		{//10
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
		},
		{//11   pod in same namespace label ns1=p1
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
			podIP[5]: podInfos[5],
		},
		{//12   #4 delete pod0
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
		},
		{//13   #11 delete pod0
			podIP[2]: podInfos[2],
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
			podIP[5]: podInfos[5],
		},
		{//14   #13 delete pod2
			podIP[3]: podInfos[3],
			podIP[4]: podInfos[4],
			podIP[5]: podInfos[5],
		},
		{//15   #14 delete pod3
			podIP[4]: podInfos[4],
			podIP[5]: podInfos[5],
		},
	}

	emptyPodInfoMap := make(PodInfoMap)

	expectedPMLMap := []PodMatchLabelMap{
		{//0
			namespacedLabels[0]: podInfoMaps[4],
			namespacedLabels[1]: podInfoMaps[1],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[2],
			namespacedLabels[4]: podInfoMaps[2],
			namespacedLabels[5]: podInfoMaps[5],
			namespacedLabels[6]: podInfoMaps[5],
			namespacedLabels[7]: podInfoMaps[5],
			namespacedLabels[8]: podInfoMaps[6],
		},
		{//1
			namespacedLabels[0]: podInfoMaps[12],
			namespacedLabels[1]: podInfoMaps[1],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[2],
			namespacedLabels[4]: podInfoMaps[2],
			namespacedLabels[5]: podInfoMaps[5],
			namespacedLabels[6]: podInfoMaps[5],
			namespacedLabels[7]: podInfoMaps[5],
			namespacedLabels[8]: podInfoMaps[6],
		},
		{//2
			namespacedLabels[0]: podInfoMaps[12],
			namespacedLabels[1]: emptyPodInfoMap,
			namespacedLabels[2]: emptyPodInfoMap,
			namespacedLabels[3]: podInfoMaps[2],
			namespacedLabels[4]: podInfoMaps[2],
			namespacedLabels[5]: podInfoMaps[5],
			namespacedLabels[6]: podInfoMaps[5],
			namespacedLabels[7]: podInfoMaps[5],
			namespacedLabels[8]: podInfoMaps[6],
		},
		{//3
			namespacedLabels[0]: podInfoMaps[12],
			namespacedLabels[1]: emptyPodInfoMap,
			namespacedLabels[2]: emptyPodInfoMap,
			namespacedLabels[3]: emptyPodInfoMap,
			namespacedLabels[4]: emptyPodInfoMap,
			namespacedLabels[5]: podInfoMaps[5],
			namespacedLabels[6]: podInfoMaps[5],
			namespacedLabels[7]: podInfoMaps[5],
			namespacedLabels[8]: podInfoMaps[6],
		},
		{//4
			namespacedLabels[0]: podInfoMaps[5],
			namespacedLabels[1]: emptyPodInfoMap,
			namespacedLabels[2]: emptyPodInfoMap,
			namespacedLabels[3]: emptyPodInfoMap,
			namespacedLabels[4]: emptyPodInfoMap,
			namespacedLabels[5]: podInfoMaps[5],
			namespacedLabels[6]: podInfoMaps[5],
			namespacedLabels[7]: podInfoMaps[5],
			namespacedLabels[8]: podInfoMaps[6],
		},
		{//5
			namespacedLabels[0]: emptyPodInfoMap,
			namespacedLabels[1]: emptyPodInfoMap,
			namespacedLabels[2]: emptyPodInfoMap,
			namespacedLabels[3]: emptyPodInfoMap,
			namespacedLabels[4]: emptyPodInfoMap,
			namespacedLabels[5]: emptyPodInfoMap,
			namespacedLabels[6]: emptyPodInfoMap,
			namespacedLabels[7]: emptyPodInfoMap,
			namespacedLabels[8]: podInfoMaps[6],
		},
		{//6
			namespacedLabels[0]: emptyPodInfoMap,
			namespacedLabels[1]: emptyPodInfoMap,
			namespacedLabels[2]: emptyPodInfoMap,
			namespacedLabels[3]: emptyPodInfoMap,
			namespacedLabels[4]: emptyPodInfoMap,
			namespacedLabels[5]: emptyPodInfoMap,
			namespacedLabels[6]: emptyPodInfoMap,
			namespacedLabels[7]: emptyPodInfoMap,
			namespacedLabels[8]: emptyPodInfoMap,
		},

	}

	expectedNPMap := []NamespacePodMap{
		{//0
			podNamespace[0] : podInfoMaps[4],
			podNamespace[1] : podInfoMaps[1],
			podNamespace[2] : podInfoMaps[7],
		},
		{//1
			podNamespace[0] : podInfoMaps[12],
			podNamespace[1] : podInfoMaps[1],
			podNamespace[2] : podInfoMaps[7],
		},
		{//2
			podNamespace[0] : podInfoMaps[12],
			podNamespace[1] : emptyPodInfoMap,
			podNamespace[2] : podInfoMaps[7],
		},
		{//3
			podNamespace[0] : podInfoMaps[12],
			podNamespace[1] : emptyPodInfoMap,
			podNamespace[2] : podInfoMaps[6],
		},
		{//4
			podNamespace[0] : podInfoMaps[5],
			podNamespace[1] : emptyPodInfoMap,
			podNamespace[2] : podInfoMaps[6],
		},
		{//5
			podNamespace[0] : emptyPodInfoMap,
			podNamespace[1] : emptyPodInfoMap,
			podNamespace[2] : podInfoMaps[6],
		},
		{//6
			podNamespace[0] : emptyPodInfoMap,
			podNamespace[1] : emptyPodInfoMap,
			podNamespace[2] : emptyPodInfoMap,
		},
	}

	expectedNMLMap := []NamespaceMatchLabelMap{
		{//0
			labels[0]: podInfoMaps[11],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[7],
		},
		{//1
			labels[0]: podInfoMaps[13],
			labels[1]: podInfoMaps[1],
			labels[2]: podInfoMaps[7],
		},
		{//2
			labels[0]: podInfoMaps[13],
			labels[1]: emptyPodInfoMap,
			labels[2]: podInfoMaps[7],
		},
		{//3
			labels[0]: podInfoMaps[14],
			labels[1]: emptyPodInfoMap,
			labels[2]: podInfoMaps[6],
		},
		{//4
			labels[0]: podInfoMaps[15],
			labels[1]: emptyPodInfoMap,
			labels[2]: podInfoMaps[6],
		},
		{//5
			labels[0]: podInfoMaps[6],
			labels[1]: emptyPodInfoMap,
			labels[2]: podInfoMaps[6],
		},
		{//6
			labels[0]: emptyPodInfoMap,
			labels[1]: emptyPodInfoMap,
			labels[2]: emptyPodInfoMap,
		},
	}

	pods := []*api.Pod{
		// 0 single pod single label
		makeTestPod(podNamespace[0], podName[0], func(pod *api.Pod) {
			pod.Labels = label1
			pod.Status.PodIP = podIP[0]
		}),
		// 1 single pod muti label
		makeTestPod(podNamespace[1], podName[1], func(pod *api.Pod) {
			pod.Labels = label2
			pod.Status.PodIP = podIP[1]
		}),
		// 2 different namespace with same label
		makeTestPod(podNamespace[2], podName[2], func(pod *api.Pod) {
			pod.Labels = label2
			pod.Status.PodIP = podIP[2]
		}),
		// 3 add new pod to exist namespace, same label
		makeTestPod(podNamespace[0], podName[3], func(pod *api.Pod) {
			pod.Labels = label1
			pod.Status.PodIP = podIP[3]
		}),
		// 4 add new pod to exist namespace, different label
		makeTestPod(podNamespace[0], podName[4], func(pod *api.Pod) {
			pod.Labels = label4
			pod.Status.PodIP = podIP[4]
		}),
		// 5
		makeTestPod(podNamespace[2], podName[5], func(pod *api.Pod) {
			pod.Labels = label1
			pod.Status.PodIP = podIP[5]
		}),
	}

	// 1. first add some pods
	for i := 0; i < 6; i++ {
		namespaceName := types.NamespacedName{Namespace: pods[i].Namespace, Name: pods[i].Name}
		podChanges.Update(&namespaceName, nil, pods[i])
	}

	// update map with 6 changes at once
	ok := checkAllPodChangeLenValid(t, podChanges, 6, 6, 6)
	if !ok{
		t.Errorf("init add: check pod change len faild")
	}

	podChanges.Lock.Lock()
	UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
	UpdateNamespacePodMap(namespacePodMap, &podChanges)
	UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

	podChanges.CleanUpItem()
	podChanges.Lock.Unlock()

	ok = checkAllPodMapValid(
		t,
		podMatchLabelMap, expectedPMLMap[0],
		namespacePodMap, expectedNPMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("init add: check all pod faild")
	}

	// 2. now try to delete pods one by one
	for i := 0; i < 6; i++ {
		namespaceName := types.NamespacedName{Namespace: pods[i].Namespace, Name: pods[i].Name}
		podChanges.Update(&namespaceName, pods[i], nil)

		ok = checkAllPodChangeLenValid(t, podChanges, 1, 1, 1)
		if !ok{
			t.Errorf("delete pod id %d check pod change len faild",i)
		}

		podChanges.Lock.Lock()
		UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
		UpdateNamespacePodMap(namespacePodMap, &podChanges)
		UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

		podChanges.CleanUpItem()
		podChanges.Lock.Unlock()

		ok = checkAllPodMapValid(
			t,
			podMatchLabelMap, expectedPMLMap[i+1],
			namespacePodMap, expectedNPMap[i+1],
			namespaceMatchLabelMap, expectedNMLMap[i+1],
		)
		if !ok{
			t.Errorf("case %d check all pod faild",i)
		}
	}

	// 3. add some pods at once again
	for i := 0; i < 6; i++ {
		namespaceName := types.NamespacedName{Namespace: pods[i].Namespace, Name: pods[i].Name}
		podChanges.Update(&namespaceName, nil, pods[i])
	}

	// update map with 6 changes at once
	ok = checkAllPodChangeLenValid(t, podChanges, 6, 6, 6)
	if !ok{
		t.Errorf("second time init add: check pod change len faild")
	}

	podChanges.Lock.Lock()
	UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
	UpdateNamespacePodMap(namespacePodMap, &podChanges)
	UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

	podChanges.CleanUpItem()
	podChanges.Lock.Unlock()

	ok = checkAllPodMapValid(
		t,
		podMatchLabelMap, expectedPMLMap[0],
		namespacePodMap, expectedNPMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("second time init add: check all pod faild")
	}

	// 4. delete these pods at once
	for i := 0; i < 6; i++ {
		namespaceName := types.NamespacedName{Namespace: pods[i].Namespace, Name: pods[i].Name}
		podChanges.Update(&namespaceName, pods[i], nil)
	}

	// update map with 6 changes at once
	ok = checkAllPodChangeLenValid(t, podChanges, 6, 6, 6)
	if !ok{
		t.Errorf("delete all: check pod change len faild")
	}

	podChanges.Lock.Lock()
	UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
	UpdateNamespacePodMap(namespacePodMap, &podChanges)
	UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

	podChanges.CleanUpItem()
	podChanges.Lock.Unlock()

	ok = checkAllPodMapValid(
		t,
		podMatchLabelMap, expectedPMLMap[6],
		namespacePodMap, expectedNPMap[6],
		namespaceMatchLabelMap, expectedNMLMap[6],
	)
	if !ok{
		t.Errorf("delete all: check all pod faild")
	}
}

// this test case have 4 parts
// 1. update pods without change, so corresponding map should keep as the same
// 2. update labels of pods which labels are exist in corresponding map
// 3. update labels of pods which will add new pod labels
// 4. update pods to the initial one, so corresponding map should as same as step 1
func TestPodInfoUpdate(t *testing.T){

	podMatchLabelMap := make(PodMatchLabelMap)
	namespacePodMap := make(NamespacePodMap)
	namespaceMatchLabelMap := make(NamespaceMatchLabelMap)
	namespaceInfoMap := make(NamespaceInfoMap)
	podChanges := NewPodLabelChangeMap()

	label0 := make(map[string]string)
	label0["pod1"] = "labeltest1"
	label1 := make(map[string]string)
	label1["pod1"] = "labeltest1"
	label1["pod2"] = "labeltest2"
	label2 := make(map[string]string)
	label2["pod2"] = "labeltest2"
	label2["pod3"] = "labeltest3"

	nlabel1 := make(map[string]string)
	nlabel1["ns1"] = "p1"
	nlabel1["ns2"] = "p2"
	nlabel2 := make(map[string]string)
	nlabel2["ns2"] = "p2"
	nlabel2["ns3"] = "p3"

	labels := []Label{
		{LabelKey: "ns1", LabelValue:"p1"},
		{LabelKey: "ns2", LabelValue:"p2"},
		{LabelKey: "ns3", LabelValue:"p3"},
	}

	namespacedLabels := []NamespacedLabel{
		{Namespace: podNamespace[0], LabelKey:"pod1", LabelValue:"labeltest1"}, //0
		{Namespace: podNamespace[0], LabelKey:"pod2", LabelValue:"labeltest2"}, //1
		{Namespace: podNamespace[1], LabelKey:"pod2", LabelValue:"labeltest2"}, //2
		{Namespace: podNamespace[1], LabelKey:"pod3", LabelValue:"labeltest3"}, //3
		{Namespace: podNamespace[0], LabelKey:"pod3", LabelValue:"labeltest3"}, //4
	}

	namespaceInfos := []*NamespaceInfo{
		{
			Name: podNamespace[0],
			Labels: nlabel1,
		},
		{
			Name: podNamespace[1],
			Labels: nlabel2,
		},
	}

	namespaceInfoMap = NamespaceInfoMap{
		podNamespace[0]: namespaceInfos[0],
		podNamespace[1]: namespaceInfos[1],
	}

	podInfos := []*PodInfo{
		{//0
			Name: podName[0],
			Namespace: podNamespace[0],
			IP: podIP[0],
			Labels: label0,
		},
		{//1
			Name: podName[1],
			Namespace: podNamespace[1],
			IP: podIP[1],
			Labels: label2,
		},
		{//2
			Name: podName[2],
			Namespace: podNamespace[0],
			IP: podIP[2],
			Labels: label1,
		},
	}

	podInfoMaps := []PodInfoMap{
		{//0
			podIP[0]: podInfos[0],
		},
		{//1
			podIP[1]: podInfos[1],
		},
		{//2
			podIP[0]: podInfos[0],
			podIP[1]: podInfos[1],
			podIP[2]: podInfos[2],
		},
		{//3
			podIP[2]: podInfos[2],
		},
		{//4
			podIP[0]: podInfos[0],
			podIP[2]: podInfos[2],
		},
	}

	emptyPodInfoMap := make(PodInfoMap)

	expectedPMLMap := []PodMatchLabelMap{
		{//0
			namespacedLabels[0]: podInfoMaps[4],
			namespacedLabels[1]: podInfoMaps[3],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[1],
		},
		{//1
			namespacedLabels[0]: podInfoMaps[4],
			namespacedLabels[1]: podInfoMaps[4],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[1],
		},
		{//2
			namespacedLabels[0]: podInfoMaps[3],
			namespacedLabels[1]: podInfoMaps[4],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[1],
			namespacedLabels[4]: podInfoMaps[0],
		},
		{//3
			namespacedLabels[0]: podInfoMaps[4],
			namespacedLabels[1]: podInfoMaps[3],
			namespacedLabels[2]: podInfoMaps[1],
			namespacedLabels[3]: podInfoMaps[1],
			namespacedLabels[4]: emptyPodInfoMap,
		},
	}

	expectedNPMap := []NamespacePodMap{
		{//0
			podNamespace[0] : podInfoMaps[4],
			podNamespace[1] : podInfoMaps[1],
		},
	}

	expectedNMLMap := []NamespaceMatchLabelMap{
		{//0
			labels[0]: podInfoMaps[4],
			labels[1]: podInfoMaps[2],
			labels[2]: podInfoMaps[1],
		},
	}

	pods := []*api.Pod{
		//0
		makeTestPod(podNamespace[0], podName[0], func(pod *api.Pod) {
			pod.Labels = label0
			pod.Status.PodIP = podIP[0]
		}),
		//1
		makeTestPod(podNamespace[1], podName[1], func(pod *api.Pod) {
			pod.Labels = label2
			pod.Status.PodIP = podIP[1]
		}),
		//2
		makeTestPod(podNamespace[0], podName[2], func(pod *api.Pod) {
			pod.Labels = label1
			pod.Status.PodIP = podIP[2]
		}),
		//3
		makeTestPod(podNamespace[0], podName[0], func(pod *api.Pod) {
			pod.Labels = label1
			pod.Status.PodIP = podIP[0]
		}),
		//4
		makeTestPod(podNamespace[0], podName[0], func(pod *api.Pod) {
			pod.Labels = label2
			pod.Status.PodIP = podIP[0]
		}),
	}

	// 0. first add 3 pods
	for i := 0; i < 3; i++ {
		namespaceName := types.NamespacedName{Namespace: pods[i].Namespace, Name: pods[i].Name}
		podChanges.Update(&namespaceName, nil, pods[i])
	}

	ok := checkAllPodChangeLenValid(t, podChanges, 3, 3, 3)
	if !ok{
		t.Errorf("init add: check pod change len faild")
	}

	podChanges.Lock.Lock()
	UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
	UpdateNamespacePodMap(namespacePodMap, &podChanges)
	UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

	podChanges.CleanUpItem()
	podChanges.Lock.Unlock()

	ok = checkAllPodMapValid(
		t,
		podMatchLabelMap, expectedPMLMap[0],
		namespacePodMap, expectedNPMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("init add: check all pod faild")
	}

	// 1. same pod update
	namespaceName := types.NamespacedName{Namespace: pods[0].Namespace, Name: pods[0].Name}
	podChanges.Update(&namespaceName, pods[0], pods[0])

	ok = checkAllPodChangeLenValid(t, podChanges, 0, 0, 0)
	if !ok{
		t.Errorf("same pod update: check pod change len faild")
	}

	podChanges.Lock.Lock()
	UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
	UpdateNamespacePodMap(namespacePodMap, &podChanges)
	UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

	podChanges.CleanUpItem()
	podChanges.Lock.Unlock()

	ok = checkAllPodMapValid(
		t,
		podMatchLabelMap, expectedPMLMap[0],
		namespacePodMap, expectedNPMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("same pod update: check all pod faild")
	}

	// 2. pod label update to exists pod label set
	namespaceName = types.NamespacedName{Namespace: pods[3].Namespace, Name: pods[3].Name}
	podChanges.Update(&namespaceName, pods[0], pods[3])

	ok = checkAllPodChangeLenValid(t, podChanges, 1, 1, 1)
	if !ok{
		t.Errorf("pod label update to old: check pod change len faild")
	}

	podChanges.Lock.Lock()
	UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
	UpdateNamespacePodMap(namespacePodMap, &podChanges)
	UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

	podChanges.CleanUpItem()
	podChanges.Lock.Unlock()

	ok = checkAllPodMapValid(
		t,
		podMatchLabelMap, expectedPMLMap[1],
		namespacePodMap, expectedNPMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("pod label update to old: check all pod faild")
	}

	// 3. pod label update to new pod label set
	namespaceName = types.NamespacedName{Namespace: pods[4].Namespace, Name: pods[4].Name}
	podChanges.Update(&namespaceName, pods[3], pods[4])

	ok = checkAllPodChangeLenValid(t, podChanges, 1, 1, 1)
	if !ok{
		t.Errorf("pod label update to new: check pod change len faild")
	}

	podChanges.Lock.Lock()
	UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
	UpdateNamespacePodMap(namespacePodMap, &podChanges)
	UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

	podChanges.CleanUpItem()
	podChanges.Lock.Unlock()

	ok = checkAllPodMapValid(
		t,
		podMatchLabelMap, expectedPMLMap[2],
		namespacePodMap, expectedNPMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("pod label update to new: check all pod faild")
	}

	// 4. back to initial pod label set
	namespaceName = types.NamespacedName{Namespace: pods[0].Namespace, Name: pods[0].Name}
	podChanges.Update(&namespaceName, pods[4], pods[0])

	ok = checkAllPodChangeLenValid(t, podChanges, 1, 1, 1)
	if !ok{
		t.Errorf("back to initial: check pod change len faild")
	}

	podChanges.Lock.Lock()
	UpdatePodMatchLabelMap(podMatchLabelMap, &podChanges)
	UpdateNamespacePodMap(namespacePodMap, &podChanges)
	UpdateNamespaceMatchLabelMapByPod(namespaceMatchLabelMap, namespaceInfoMap, &podChanges)

	podChanges.CleanUpItem()
	podChanges.Lock.Unlock()

	ok = checkAllPodMapValid(
		t,
		podMatchLabelMap, expectedPMLMap[3],
		namespacePodMap, expectedNPMap[0],
		namespaceMatchLabelMap, expectedNMLMap[0],
	)
	if !ok{
		t.Errorf("back to initial: check all pod faild")
	}
}

func checkAllPodMapValid(t *testing.T,
	pmlMap, expectedPMLMap PodMatchLabelMap,
	npMap, expectedNPMap NamespacePodMap,
	nmlMapm, expectedNMLMap NamespaceMatchLabelMap,
) bool{
	var result = true
	ok := checkPodMatchLabelMapValid(t, pmlMap, expectedPMLMap)
	if !ok{
		t.Errorf("invalid podMatchLabelMap")
		result = false
	}
	ok = checkNamespacePodMapValid(t, npMap, expectedNPMap)
	if !ok{
		t.Errorf("invalid namespacePodMap")
		result = false
	}
	ok = checkNamespaceMatchLabelMapValid(t, nmlMapm, expectedNMLMap)
	if !ok{
		t.Errorf("invalid namespaceMatchLabelMap")
		result = false
	}

	return result
}

func checkAllPodChangeLenValid(t *testing.T,
	podChanges PodChangeMap,
	expectPodLen, expectNamespacePodLen, expectNamespaceLabelLen int,
) bool{
	var result = true
	podNumber := len(podChanges.PodItems)
	if podNumber != expectPodLen{
		t.Errorf("pod map change map pod len is %d, expected %d", podNumber, expectPodLen)
		result = false
	}
	namespaceNumber := len(podChanges.NamespacePodItems)
	if namespaceNumber != expectNamespacePodLen{
		t.Errorf("pod map change map namespace len is %d, expected %d", namespaceNumber, expectNamespacePodLen)
		result = false
	}
	namespaceNumber = len(podChanges.NamespaceLabelItems)
	if namespaceNumber != expectNamespaceLabelLen{
		t.Errorf("pod map change map namespace len is %d, expected %d", namespaceNumber, expectNamespaceLabelLen)
		result = false
	}

	return result
}

func checkPodMatchLabelMapValid(t *testing.T, pmlMap, expectedPMLMap PodMatchLabelMap) bool{

	if len(pmlMap) != len(expectedPMLMap){
		t.Errorf("length of podMatchLabelMap is not correct %d, expect %d", len(pmlMap), len(expectedPMLMap))

		for namespaceLabel := range pmlMap{
			t.Errorf("namespaceLabel: %s:%s=%s", namespaceLabel.Namespace, namespaceLabel.LabelKey, namespaceLabel.LabelValue)
		}
		for namespaceLabel := range expectedPMLMap{
			t.Errorf("expected label: %s:%s=%s", namespaceLabel.Namespace, namespaceLabel.LabelKey, namespaceLabel.LabelValue)
		}
		return false
	}

	for exNamespaceLabel, exPodInfoMap := range expectedPMLMap{
		podInfoMap, ok := pmlMap[exNamespaceLabel]
		if !ok{
			t.Errorf("cannot find expected namespacelabel: %s:%s=%s", exNamespaceLabel.Namespace, exNamespaceLabel.LabelKey, exNamespaceLabel.LabelValue)
			for namespaceLabel := range pmlMap{
				t.Errorf("namespaceLabel: %s:%s=%s", namespaceLabel.Namespace, namespaceLabel.LabelKey, namespaceLabel.LabelValue)
			}
			return false
		}
		if len(podInfoMap) != len(exPodInfoMap){
			t.Errorf("podInfoMap len in namespacelabel: %s:%s=%s is not correct %d, expect %d",
				exNamespaceLabel.Namespace, exNamespaceLabel.LabelKey, exNamespaceLabel.LabelValue, len(podInfoMap), len(exPodInfoMap))
			for podIP := range podInfoMap{
				t.Errorf("pod ip: %s", podIP)
			}
			for podIP := range exPodInfoMap{
				t.Errorf("expect pod ip: %s", podIP)
			}
			return false
		}
		for exPodIP := range exPodInfoMap{
			_, ok = podInfoMap[exPodIP]
			if !ok{
				t.Errorf("cannot find expected pod ip %s in namespacelabel: %s:%s=%s",
					exPodIP,exNamespaceLabel.Namespace, exNamespaceLabel.LabelKey, exNamespaceLabel.LabelValue)
				for podIP := range podInfoMap{
					t.Errorf("pod ip: %s", podIP)
				}
				return false
			}
		}
	}

	return true
}

func checkNamespacePodMapValid(t *testing.T, npMap, expectedNPMap NamespacePodMap) bool{

	if len(npMap) != len(expectedNPMap){
		t.Errorf("length pf NamespacePodMap is not correct %d, expected %d", len(npMap), len(expectedNPMap))

		for namespace := range npMap{
			t.Errorf("namespace: %s", namespace)
		}
		for namespace := range expectedNPMap{
			t.Errorf("expected namespace: %s", namespace)
		}
		return false
	}

	for exNamespace, exPodInfoMap := range expectedNPMap{
		podInfoMap, ok := npMap[exNamespace]
		if !ok{
			t.Errorf("canot find expected namespace %s", exNamespace)
			for namespace := range npMap{
				t.Errorf("namespace: %s", namespace)
			}
			return false
		}
		if len(podInfoMap) != len(exPodInfoMap){
			t.Errorf("podInfoMap len in namespace %s is not correct %d, expect %d", exNamespace, len(podInfoMap), len(exPodInfoMap))
			for podIP := range podInfoMap{
				t.Errorf("pod ip: %s", podIP)
			}
			for podIP := range exPodInfoMap{
				t.Errorf("expect pod ip: %s", podIP)
			}
			return false
		}
		for exPodIP := range exPodInfoMap{
			_, ok = podInfoMap[exPodIP]
			if !ok{
				t.Errorf("cannot find expected pod ip %s in namespace %s", exPodIP, exNamespace)
				for podIP := range podInfoMap{
					t.Errorf("pod ip: %s", podIP)
				}
				return false
			}
		}
	}

	return true
}

func checkNamespaceMatchLabelMapValid(t *testing.T, nmlMapm, expectedNMLMap NamespaceMatchLabelMap) bool{

	if len(nmlMapm) != len(expectedNMLMap){
		t.Errorf("length of NamespaceMatchLabelMap is not correct %d, expected %d", len(nmlMapm), len(expectedNMLMap))

		for label := range nmlMapm{
			t.Errorf("label is %s=%s", label.LabelKey, label.LabelValue)
		}
		for label := range expectedNMLMap{
			t.Errorf("expect label is %s=%s", label.LabelKey, label.LabelValue)
		}
		return false
	}

	for exLabel, exPodInfoMap := range expectedNMLMap{
		podInfoMap, ok := nmlMapm[exLabel]
		if !ok{
			t.Errorf("canot find expected label %s=%s", exLabel.LabelKey, exLabel.LabelValue)
			for label := range nmlMapm{
				t.Errorf("label is %s=%s", label.LabelKey, label.LabelValue)
			}
			return false
		}
		if len(podInfoMap) != len(exPodInfoMap){
			t.Errorf("podInfoMap len in label %s=%s is not correct %d, expect %d",
				exLabel.LabelKey, exLabel.LabelValue, len(podInfoMap), len(exPodInfoMap))
			for podIP := range podInfoMap{
				t.Errorf("pod ip: %s", podIP)
			}
			for podIP := range exPodInfoMap{
				t.Errorf("expect pod ip: %s", podIP)
			}
			return false
		}
		for exPodIP := range exPodInfoMap{
			_, ok = podInfoMap[exPodIP]
			if !ok{
				t.Errorf("cannot find expected pod ip %s in label %s=%s", exPodIP, exLabel.LabelKey, exLabel.LabelValue)
				for podIP := range podInfoMap{
					t.Errorf("pod ip: %s", podIP)
				}
				return false
			}
		}
	}

	return true
}