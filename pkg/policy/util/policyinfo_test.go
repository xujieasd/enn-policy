package util

import (
	api "k8s.io/api/networking/v1"
	coreApi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"reflect"
	"strings"
)

func makeTestNetworkPolicy(namespace, name string, npFunc func(*api.NetworkPolicy)) *api.NetworkPolicy{

	np := &api.NetworkPolicy{
		ObjectMeta:  metav1.ObjectMeta{
			Namespace: namespace,
			Name: name,
		},
		Spec:  api.NetworkPolicySpec{},
	}
	npFunc(np)
	return np
}

// this test case will add some networkPolicy one by one
// will add all kinds of networkPolicy rules,
// PolicyTypes include ingress, egress,
// NetworkPolicyPeer include PodSelector, NamespaceSelector, IPBlock
// Ports include TCP and UDP
// and then check if their corresponding map is correct
func TestNetworkPolicyMapAdd(t *testing.T){

	networkPolicyMap := make(NetworkPolicyMap)
	networkPolicyChanges := NewNetworkPolicyChangeMap()

	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"
	label3 := make(map[string]string)
	label3["run3"] = "labeltest3"
	label4 := make(map[string]string)
	label4["run4"] = "labeltest4"


	namespace := []string{
		"testns1",
		"testns2",
		"testns3",
		"testns4",
		"testns5",
		"testns6",
		"testns7",
		"testns8",
	}
	name := []string{
		"testnp1",
		"testnp2",
		"testnp3",
		"testnp4",
		"testns5",
		"testns6",
		"testns7",
		"testns8",
	}

	protocolTCP := coreApi.Protocol(coreApi.ProtocolTCP)
	protocolUDP := coreApi.Protocol(coreApi.ProtocolUDP)

	networkPolicies := []*api.NetworkPolicy{
		makeTestNetworkPolicy(namespace[0], name[0], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[1], name[1], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[2], name[2], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						IPBlock: &api.IPBlock{
							CIDR: "172.10.2.0/16",
						},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[3], name[3], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolTCP,
						Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 6789},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[4], name[4], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label3},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[5], name[5], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label4},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[6], name[6], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{
						IPBlock: &api.IPBlock{
							CIDR: "198.168.2.0/24",
						},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[7], name[7], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolUDP,
						Port: &intstr.IntOrString{Type: intstr.String, StrVal: "1234"},
					}},
				},
			}
		}),
	}

	for i := 0; i < 8; i++{
		namespaceName := types.NamespacedName{Namespace: networkPolicies[i].Namespace, Name: networkPolicies[i].Name}
		networkPolicyChanges.Update(&namespaceName, nil, networkPolicies[i])
		number := len(networkPolicyChanges.Items)
		if number != 1{
			t.Errorf("case %d policy map change map len is %d, expected 1", i, number)
		}
		UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)

		ok := checkNetworkPolicyNumber(t, networkPolicyMap, i+1)
		if !ok{
			t.Errorf("case %d invalid networkPolicy number", i)
		}
		ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[i])
		if !ok{
			t.Errorf("case %d invalid networkPolicy %s:%s", i, namespace[i], name[i])
		}
	}
}

// this test case will add some complex networkPolicy rules
// and then check if their corresponding map is correct
func TestNetworkPolicyMapMultiAdd(t *testing.T){

	networkPolicyMap := make(NetworkPolicyMap)
	networkPolicyChanges := NewNetworkPolicyChangeMap()

	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"
	label3 := make(map[string]string)
	label3["run3"] = "labeltest3"
	label4 := make(map[string]string)
	label4["run4"] = "labeltest4"


	namespace := []string{
		"testns1",
		"testns2",
		"testns3",
		"testns4",
		"testns5",
		"testns6",
		"testns7",
		"testns8",
	}
	name := []string{
		"testnp1",
		"testnp2",
		"testnp3",
		"testnp4",
		"testns5",
		"testns6",
		"testns7",
		"testns8",
	}

	protocolTCP := coreApi.Protocol(coreApi.ProtocolTCP)
	protocolUDP := coreApi.Protocol(coreApi.ProtocolUDP)

	networkPolicies := []*api.NetworkPolicy{
		makeTestNetworkPolicy(namespace[0], name[0], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[1], name[1], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[2], name[2], func(np *api.NetworkPolicy) {
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress, api.PolicyTypeEgress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
						IPBlock: &api.IPBlock{CIDR: "198.168.2.0/24"},
					}},
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolTCP,
						Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 6789},
					}},
				},
			}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label3},
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label4},
						IPBlock: &api.IPBlock{CIDR: "198.168.1.0/24"},
					}},
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolTCP,
						Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 1234},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[3], name[3], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						IPBlock: &api.IPBlock{CIDR: "198.168.1.0/24"},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label3},
					}},
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolUDP,
						Port: &intstr.IntOrString{Type: intstr.String, StrVal: "1234"},
					}},
				},
			}
		}),
	}

	// add two networkPolicy together
	namespaceName := types.NamespacedName{Namespace: networkPolicies[0].Namespace, Name: networkPolicies[0].Name}
	networkPolicyChanges.Update(&namespaceName, nil, networkPolicies[0])
	namespaceName = types.NamespacedName{Namespace: networkPolicies[1].Namespace, Name: networkPolicies[1].Name}
	networkPolicyChanges.Update(&namespaceName, nil, networkPolicies[1])
	number := len(networkPolicyChanges.Items)
	if number != 2{
		t.Errorf("case %d policy map change map len is %d, expected 2", 1, number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok := checkNetworkPolicyNumber(t, networkPolicyMap, 2)
	if !ok{
		t.Errorf("case %d invalid networkPolicy number", 1)
	}
	ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[0])
	if !ok{
		t.Errorf("case %d invalid networkPolicy %s:%s", 1, namespace[0], name[0])
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[1])
	if !ok{
		t.Errorf("case %d invalid networkPolicy %s:%s", 1, namespace[1], name[1])
	}

	// add networkPolicy with both ingress and egress rule
	namespaceName = types.NamespacedName{Namespace: networkPolicies[2].Namespace, Name: networkPolicies[0].Name}
	networkPolicyChanges.Update(&namespaceName, nil, networkPolicies[2])
	number = len(networkPolicyChanges.Items)
	if number != 1{
		t.Errorf("case %d policy map change map len is %d, expected 1", 2, number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok = checkNetworkPolicyNumber(t, networkPolicyMap, 3)
	if !ok{
		t.Errorf("case %d invalid networkPolicy number", 2)
	}
	ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[2])
	if !ok{
		t.Errorf("case %d invalid networkPolicy %s:%s", 2, namespace[2], name[2])
	}

	// add networkPolicy with multi ingress rule
	namespaceName = types.NamespacedName{Namespace: networkPolicies[3].Namespace, Name: networkPolicies[0].Name}
	networkPolicyChanges.Update(&namespaceName, nil, networkPolicies[3])
	number = len(networkPolicyChanges.Items)
	if number != 1{
		t.Errorf("case %d policy map change map len is %d, expected 1", 3, number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok = checkNetworkPolicyNumber(t, networkPolicyMap, 4)
	if !ok{
		t.Errorf("case %d invalid networkPolicy number", 3)
	}
	ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[2])
	if !ok{
		t.Errorf("case %d invalid networkPolicy %s:%s", 3, namespace[3], name[3])
	}

}

// this test case have 4 parts
// 1. add some networkPolicy at once (means networkPolicyChangeMap will batch these changes)
// 2. delete these networkPolicy one by one
// 3. add these networkPolicy at once again
// 4. delete these networkPolicy at once
func TestNetworkPolicyMapDelete(t *testing.T){

	networkPolicyMap := make(NetworkPolicyMap)
	networkPolicyChanges := NewNetworkPolicyChangeMap()

	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"
	label3 := make(map[string]string)
	label3["run3"] = "labeltest3"
	label4 := make(map[string]string)
	label4["run4"] = "labeltest4"

	namespace := []string{
		"testns1",
		"testns2",
		"testns3",
		"testns4",
		"testns5",
		"testns6",
		"testns7",
		"testns8",
	}
	name := []string{
		"testnp1",
		"testnp2",
		"testnp3",
		"testnp4",
		"testns5",
		"testns6",
		"testns7",
		"testns8",
	}

	protocolTCP := coreApi.Protocol(coreApi.ProtocolTCP)
	protocolUDP := coreApi.Protocol(coreApi.ProtocolUDP)

	networkPolicies := []*api.NetworkPolicy{
		makeTestNetworkPolicy(namespace[0], name[0], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[1], name[1], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[2], name[2], func(np *api.NetworkPolicy) {
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress, api.PolicyTypeEgress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
						IPBlock: &api.IPBlock{CIDR: "198.168.2.0/24"},
					}},
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolTCP,
						Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 6789},
					}},
				},
			}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label3},
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label4},
						IPBlock: &api.IPBlock{CIDR: "198.168.1.0/24"},
					}},
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolTCP,
						Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 1234},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[3], name[3], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						IPBlock: &api.IPBlock{CIDR: "198.168.1.0/24"},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label3},
					}},
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolUDP,
						Port: &intstr.IntOrString{Type: intstr.String, StrVal: "1234"},
					}},
				},
			}
		}),
	}

	// 1. add 4 networkPolicy together
	for i := 0; i < 4; i++{
		namespaceName := types.NamespacedName{Namespace: networkPolicies[i].Namespace, Name: networkPolicies[i].Name}
		networkPolicyChanges.Update(&namespaceName, nil, networkPolicies[i])
	}

	number := len(networkPolicyChanges.Items)
	if number != 4{
		t.Errorf("init add: policy map change map len is %d, expected 4", number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok := checkNetworkPolicyNumber(t, networkPolicyMap, 4)
	if !ok{
		t.Errorf("init add: invalid networkPolicy number")
	}
	for i := 0; i < 4; i++{
		ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[i])
		if !ok{
			t.Errorf("init add: invalid networkPolicy %s:%s", namespace[i], name[i])
		}
	}

	// 2. delete 4 networkPolicy one by one
	for i := 0; i < 4; i++{
		namespaceName := types.NamespacedName{Namespace: networkPolicies[i].Namespace, Name: networkPolicies[i].Name}
		networkPolicyChanges.Update(&namespaceName, networkPolicies[i], nil)
		number := len(networkPolicyChanges.Items)
		if number != 1{
			t.Errorf("delete case %d: policy map change map len is %d, expected 1", i, number)
		}
		UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
		ok := checkNetworkPolicyNumber(t, networkPolicyMap, 3 - i)
		if !ok{
			t.Errorf("delete case %d: invalid networkPolicy number", i)
		}
		for j := i + 1; j < 4; j++{
			ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[j])
			if !ok{
				t.Errorf("delete case %d: invalid networkPolicy %s:%s", i, namespace[j], name[j])
			}
		}
	}

	// 3. second time add 4 networkPolicy together
	for i := 0; i < 4; i++{
		namespaceName := types.NamespacedName{Namespace: networkPolicies[i].Namespace, Name: networkPolicies[i].Name}
		networkPolicyChanges.Update(&namespaceName, nil, networkPolicies[i])
	}

	number = len(networkPolicyChanges.Items)
	if number != 4{
		t.Errorf("second time add: policy map change map len is %d, expected 4", number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok = checkNetworkPolicyNumber(t, networkPolicyMap, 4)
	if !ok{
		t.Errorf("second time add: invalid networkPolicy number")
	}
	for i := 0; i < 4; i++{
		ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[i])
		if !ok{
			t.Errorf("second time add: invalid networkPolicy %s:%s", namespace[i], name[i])
		}
	}

	// 4. delete 4 networkPolicy together
	for i := 0; i < 4; i++ {
		namespaceName := types.NamespacedName{Namespace: networkPolicies[i].Namespace, Name: networkPolicies[i].Name}
		networkPolicyChanges.Update(&namespaceName, networkPolicies[i], nil)
	}
	number = len(networkPolicyChanges.Items)
	if number != 4{
		t.Errorf("delete all: policy map change map len is %d, expected 4", number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok = checkNetworkPolicyNumber(t, networkPolicyMap, 0)
	if !ok{
		t.Errorf("delete all: invalid networkPolicy number")
	}

}

// this test case have 3 parts
// 1. update networkPolicy without change, so corresponding map should keep as the same
// 2. update networkPolicy to new networkPolicy
// 3. update networkPolicy to the initial one, so corresponding map should as same as step 1
func TestNetworkPolicyMapUpdate(t *testing.T){

	networkPolicyMap := make(NetworkPolicyMap)
	networkPolicyChanges := NewNetworkPolicyChangeMap()


	label1 := make(map[string]string)
	label1["run1"] = "labeltest1"
	label2 := make(map[string]string)
	label2["run2"] = "labeltest2"
	label3 := make(map[string]string)
	label3["run3"] = "labeltest3"
	label4 := make(map[string]string)
	label4["run4"] = "labeltest4"

	namespace := []string{
		"testns1",
	}
	name := []string{
		"testnp1",
	}

	protocolTCP := coreApi.Protocol(coreApi.ProtocolTCP)
	protocolUDP := coreApi.Protocol(coreApi.ProtocolUDP)

	networkPolicies := []*api.NetworkPolicy{
		makeTestNetworkPolicy(namespace[0], name[0], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
					}},
				},
			}
		}),
		makeTestNetworkPolicy(namespace[0], name[0], func(np *api.NetworkPolicy){
			np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress, api.PolicyTypeEgress}
			np.Spec.Ingress = []api.NetworkPolicyIngressRule{
				{
					From: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
						IPBlock: &api.IPBlock{CIDR: "198.168.2.0/24"},
					}},
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolTCP,
						Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 6789},
					}},
				},
			}
			np.Spec.Egress = []api.NetworkPolicyEgressRule{
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label1},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: label2},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						IPBlock: &api.IPBlock{CIDR: "198.168.1.0/24"},
					}},
				},
				{
					To: []api.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{MatchLabels: label3},
					}},
					Ports: []api.NetworkPolicyPort{{
						Protocol: &protocolUDP,
						Port: &intstr.IntOrString{Type: intstr.String, StrVal: "1234"},
					}},
				},
			}
		}),
	}

	// 0. init add networkPolicy0
	namespaceName := types.NamespacedName{Namespace: networkPolicies[0].Namespace, Name: networkPolicies[0].Name}
	networkPolicyChanges.Update(&namespaceName, nil, networkPolicies[0])
	number := len(networkPolicyChanges.Items)
	if number != 1{
		t.Errorf("init add: policy map change map len is %d, expected 1", number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok := checkNetworkPolicyNumber(t, networkPolicyMap, 1)
	if !ok{
		t.Errorf("init add: invalid networkPolicy number")
	}
	ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[0])
	if !ok{
		t.Errorf("init add: invalid networkPolicy %s:%s", namespace[0], name[0i])
	}

	// 1. update networkPolicy without change
	namespaceName = types.NamespacedName{Namespace: networkPolicies[0].Namespace, Name: networkPolicies[0].Name}
	networkPolicyChanges.Update(&namespaceName, networkPolicies[0], networkPolicies[0])
	number = len(networkPolicyChanges.Items)
	if number != 0{
		t.Errorf("update networkPolicy without change: policy map change map len is %d, expected 0", number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok = checkNetworkPolicyNumber(t, networkPolicyMap, 1)
	if !ok{
		t.Errorf("update networkPolicy without change: invalid networkPolicy number")
	}
	ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[0])
	if !ok{
		t.Errorf("update networkPolicy without change: invalid networkPolicy %s:%s", namespace[0], name[0])
	}

	// 2. update networkPolicy to new
	namespaceName = types.NamespacedName{Namespace: networkPolicies[1].Namespace, Name: networkPolicies[1].Name}
	networkPolicyChanges.Update(&namespaceName, networkPolicies[0], networkPolicies[1])
	number = len(networkPolicyChanges.Items)
	if number != 1{
		t.Errorf("update networkPolicy to new: policy map change map len is %d, expected 1", number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok = checkNetworkPolicyNumber(t, networkPolicyMap, 1)
	if !ok{
		t.Errorf("update networkPolicy to new: invalid networkPolicy number")
	}
	ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[1])
	if !ok{
		t.Errorf("update networkPolicy to new: invalid networkPolicy %s:%s", namespace[1], name[1])
	}

	// 3. update networkPolicy to initial
	namespaceName = types.NamespacedName{Namespace: networkPolicies[0].Namespace, Name: networkPolicies[0].Name}
	networkPolicyChanges.Update(&namespaceName, networkPolicies[1], networkPolicies[0])
	number = len(networkPolicyChanges.Items)
	if number != 1{
		t.Errorf("update networkPolicy to initial: policy map change map len is %d, expected 1", number)
	}
	UpdateNetworkPolicyMap(networkPolicyMap, &networkPolicyChanges)
	ok = checkNetworkPolicyNumber(t, networkPolicyMap, 1)
	if !ok{
		t.Errorf("update networkPolicy to initial: invalid networkPolicy number")
	}
	ok = checkNetworkPolicyValid(t, networkPolicyMap, networkPolicies[0])
	if !ok{
		t.Errorf("update networkPolicy to initial: invalid networkPolicy %s:%s", namespace[0], name[0])
	}

}

func checkNetworkPolicyNumber(t *testing.T, npMap NetworkPolicyMap, expectNumber int) bool{

	number := len(npMap)
	if number!= expectNumber{
		t.Errorf("networkPolicy Map len is not correct %d, expected %d", number, expectNumber)
		return false
	}
	return true
}

func checkNetworkPolicyValid(t *testing.T, npMap NetworkPolicyMap, networkPolicy *api.NetworkPolicy) bool{

	validNamespaceName := false
	for _, np := range npMap{
		if np.Namespace == networkPolicy.Namespace && np.Name == networkPolicy.Name{
			validNamespaceName = true

			for _, policyType := range np.PolicyType {
				if policyType == TypeIngress{
					if len(networkPolicy.Spec.Ingress) != len(np.Ingress){
						t.Errorf("invalid indexIngress expect %d, max len %d", len(np.Ingress), len(networkPolicy.Spec.Ingress))
						return false
					}
					for indexIngress, ingress := range np.Ingress{
						specIngressRule := networkPolicy.Spec.Ingress[indexIngress]
						indexPod           := len(ingress.PodSelector)
						indexNamespace     := len(ingress.NamespaceSelector)
						indexCIDR          := len(ingress.CIDR)
						tempIndexPod       := 0
						tempIndexNamespace := 0
						tempIndexCIDR      := 0

						if len(specIngressRule.Ports)!=len(ingress.Ports){
							t.Errorf("invalid port length %d, expect %d", len(ingress.Ports), (specIngressRule.Ports))
							return false
						}
						for indexPort, specPorts := range specIngressRule.Ports{
							specPort := PolicyPort{
								Port:      specPorts.Port.String(),
								Protocol:  string(*specPorts.Protocol),
							}
							port := ingress.Ports[indexPort]
							if !reflect.DeepEqual(specPort, port){
								t.Errorf("specPorts is not equal")
								return false
							}

						}
						for _, specPeer := range specIngressRule.From{
							if specPeer.PodSelector != nil{
								if tempIndexPod > indexPod-1{
									t.Errorf("invalid PodSelector index expect %d len %d", tempIndexPod, indexPod)
									return false
								}
								label := ingress.PodSelector[tempIndexPod].Label
								if !reflect.DeepEqual(label, specPeer.PodSelector.MatchLabels){
									t.Errorf("PodSelector.MatchLabels is not equal")
									return false
								}
								tempIndexPod ++
							}
							if specPeer.NamespaceSelector != nil{
								if tempIndexNamespace > indexNamespace-1{
									t.Errorf("invalid NamespaceSelector index expect %d len %d", tempIndexNamespace, indexNamespace)
									return false
								}
								label := ingress.NamespaceSelector[tempIndexNamespace].Label
								if !reflect.DeepEqual(label, specPeer.NamespaceSelector.MatchLabels){
									t.Errorf("NamespaceSelector.MatchLabels is not equal")
									return false
								}
								tempIndexNamespace ++
							}
							if specPeer.IPBlock != nil{
								if tempIndexCIDR > indexCIDR-1{
									t.Errorf("invalid IPBlock index expect %d len %d", tempIndexCIDR, indexCIDR)
									return false
								}
								cidr := ingress.CIDR[tempIndexCIDR]
								if strings.Compare(specPeer.IPBlock.CIDR, cidr) != 0{
									t.Errorf("IPBlock.CIDR is not equal %s, expect %s", specPeer.IPBlock.CIDR, cidr)
									return false
								}
								tempIndexCIDR ++
							}
						}
						if tempIndexPod != indexPod{
							t.Errorf("invalid PodSelector max index expect %d max %d", tempIndexPod, indexPod)
							return false
						}
						if tempIndexNamespace != indexNamespace{
							t.Errorf("invalid NamespaceSelector max index expect %d max %d", tempIndexNamespace, indexNamespace)
							return false
						}
						if tempIndexCIDR != indexCIDR{
							t.Errorf("invalid IPBlock max index expect %d max %d", tempIndexCIDR, indexCIDR)
							return false
						}
					}


				}else if policyType == TypeEgress{
					if len(networkPolicy.Spec.Egress) != len(np.Egress){
						t.Errorf("invalid indexEgress expect %d, max len %d", len(np.Egress), len(networkPolicy.Spec.Egress))
						return false
					}
					for indexEgress, egress := range np.Egress{
						specEgressRule := networkPolicy.Spec.Egress[indexEgress]
						indexPod           := len(egress.PodSelector)
						indexNamespace     := len(egress.NamespaceSelector)
						indexCIDR          := len(egress.CIDR)
						tempIndexPod       := 0
						tempIndexNamespace := 0
						tempIndexCIDR      := 0

						if len(specEgressRule.Ports)!=len(egress.Ports){
							t.Errorf("invalid port length %d, expect %d", len(egress.Ports), (specEgressRule.Ports))
							return false
						}
						for indexPort, specPorts := range specEgressRule.Ports{
							specPort := PolicyPort{
								Port:      specPorts.Port.String(),
								Protocol:  string(*specPorts.Protocol),
							}
							port := egress.Ports[indexPort]
							if !reflect.DeepEqual(specPort, port){
								t.Errorf("specPorts is not equal")
								return false
							}

						}

						for _, specPeer := range specEgressRule.To{
							if specPeer.PodSelector != nil{
								if tempIndexPod > indexPod-1{
									t.Errorf("invalid PodSelector index expect %d max %d", tempIndexPod, indexPod)
									return false
								}
								label := egress.PodSelector[tempIndexPod].Label
								if !reflect.DeepEqual(label, specPeer.PodSelector.MatchLabels){
									t.Errorf("PodSelector.MatchLabels is not equal")
									return false
								}
								tempIndexPod ++
							}
							if specPeer.NamespaceSelector != nil{
								if tempIndexNamespace > indexNamespace-1{
									t.Errorf("invalid NamespaceSelector index expect %d max %d", tempIndexNamespace, indexNamespace)
									return false
								}
								label := egress.NamespaceSelector[tempIndexNamespace].Label
								if !reflect.DeepEqual(label, specPeer.NamespaceSelector.MatchLabels){
									t.Errorf("NamespaceSelector.MatchLabels is not equal")
									return false
								}
								tempIndexNamespace ++
							}
							if specPeer.IPBlock != nil{
								if tempIndexCIDR > indexCIDR-1{
									t.Errorf("invalid IPBlock index expect %d max %d", tempIndexCIDR, indexCIDR)
									return false
								}
								cidr := egress.CIDR[tempIndexCIDR]
								if strings.Compare(specPeer.IPBlock.CIDR, cidr) != 0{
									t.Errorf("IPBlock.CIDR is not equal %s, expect %s", specPeer.IPBlock.CIDR, cidr)
									return false
								}
								tempIndexCIDR ++
							}
						}
						if tempIndexPod != indexPod{
							t.Errorf("invalid PodSelector max index expect %d max %d", tempIndexPod, indexPod)
							return false
						}
						if tempIndexNamespace != indexNamespace{
							t.Errorf("invalid NamespaceSelector max index expect %d max %d", tempIndexNamespace, indexNamespace)
							return false
						}
						if tempIndexCIDR != indexCIDR{
							t.Errorf("invalid IPBlock max index expect %d max %d", tempIndexCIDR, indexCIDR)
							return false
						}
					}
				}
			}

			break
		}
	}

	if !validNamespaceName{
		t.Errorf("cannot find namespaceName %s:%s", networkPolicy.Namespace, networkPolicy.Name)
		return false
	}

	return true
}