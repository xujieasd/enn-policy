package policy

import (
	api "k8s.io/api/networking/v1"
	coreApi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilexec "k8s.io/utils/exec"
	utilpolicy "enn-policy/pkg/policy/util"
	utilIPSet "enn-policy/pkg/util/ipset"
	utilIPTables "enn-policy/pkg/util/iptables"
	k8sIPTables "enn-policy/pkg/util/k8siptables"
	utildbus "enn-policy/pkg/util/dbus"

	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"

	"strings"
	"bytes"
	//"strconv"
	"strconv"
)

var protocolTCP = coreApi.Protocol(coreApi.ProtocolTCP)
var protocolUDP = coreApi.Protocol(coreApi.ProtocolUDP)

var namespaceName = []string{
	"namespace0",
	"namespace1",
	"namespace2",
	"namespace3",
	"namespace4",
}

//var iptablesInterface = utilIPTables.NewEnnIPTables()

var networkPolicies = []*api.NetworkPolicy{
	//0 ingress podSelector
	makeTestNetworkPolicy(namespaceName[0], "np0", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.Ingress = []api.NetworkPolicyIngressRule{
			{
				From: []api.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test1"}},
				}},
			},
		}
	}),
	//1 ingress namespaceSelector
	makeTestNetworkPolicy(namespaceName[0], "np1", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test1"}}
		np.Spec.Ingress = []api.NetworkPolicyIngressRule{
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns1":"ns1"}},
				}},
			},
		}
	}),
	//2 ingress ipBlock
	makeTestNetworkPolicy(namespaceName[1], "np2", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.Ingress = []api.NetworkPolicyIngressRule{
			{
				From: []api.NetworkPolicyPeer{{
					IPBlock: &api.IPBlock{
						CIDR: "172.10.0.0/16",
					},
				}},
			},
		}
	}),
	//3 ingress port
	makeTestNetworkPolicy(namespaceName[1], "np3", func(np *api.NetworkPolicy){
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
	//4 egress podSelector
	makeTestNetworkPolicy(namespaceName[2], "np4", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
		np.Spec.Egress = []api.NetworkPolicyEgressRule{
			{
				To: []api.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test1","run3":"test3"}},
				}},
			},
		}
	}),
	//5 egress namespaceSelector
	makeTestNetworkPolicy(namespaceName[2], "np5", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test1","run2":"test2"}}
		np.Spec.Egress = []api.NetworkPolicyEgressRule{
			{
				To: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns2":"ns2","ns3":"ns3"}},
				}},
			},
		}
	}),
	//6 egress ipBlock
	makeTestNetworkPolicy(namespaceName[0], "np6", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test1"}}
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
	//7 egress port
	makeTestNetworkPolicy(namespaceName[0], "np7", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeEgress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test1"}}
		np.Spec.Egress = []api.NetworkPolicyEgressRule{
			{
				To: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns1":"ns1"}},
				}},
				Ports: []api.NetworkPolicyPort{{
					Protocol: &protocolUDP,
					Port: &intstr.IntOrString{Type: intstr.String, StrVal: "1234"},
				}},
			},
		}
	}),
	//8 ingress multi-networkPolicyPeer
	makeTestNetworkPolicy(namespaceName[3], "np8", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test1","run2":"test2"}}
		np.Spec.Ingress = []api.NetworkPolicyIngressRule{
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns2":"ns2","ns3":"ns3"}},
				}},
			},
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns2":"ns22"}},
				}},
			},
		}
	}),
	//9 same namespace of #8
	makeTestNetworkPolicy(namespaceName[3], "np9", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test11"}}
		np.Spec.Ingress = []api.NetworkPolicyIngressRule{
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns3":"ns33"}},
				}},
			},
		}
	}),
	//10 ingress namespaceSelector
	makeTestNetworkPolicy(namespaceName[4], "np10", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test11","run2":"test2"}}
		np.Spec.Ingress = []api.NetworkPolicyIngressRule{
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns2":"ns22","ns3":"ns33"}},
				}},
			},
		}
	}),
	//11 ingress update namespaceSelector from #10 delete namespace label
	makeTestNetworkPolicy(namespaceName[4], "np10", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test11","run2":"test2"}}
		np.Spec.Ingress = []api.NetworkPolicyIngressRule{
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns2":"ns22"}},
				}},
			},
		}
	}),
	//12 ingress update namespaceSelector from #11 add namespace label
	makeTestNetworkPolicy(namespaceName[4], "np10", func(np *api.NetworkPolicy){
		np.Spec.PolicyTypes = []api.PolicyType{api.PolicyTypeIngress}
		np.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{"run1":"test11","run2":"test2"}}
		np.Spec.Ingress = []api.NetworkPolicyIngressRule{
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns2":"ns22"}},
				}},
			},
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns3":"ns33"}},
				}},
			},
			{
				From: []api.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"ns4":"ns4"}},
				}},
			},
		}
	}),
}

func NewFakeEnnPolicy(ipRange string) *EnnPolicy{

	execInterface := utilexec.New()
	//ipsetInterface := fakeIPSet.NewFaker()
	ipsetInterface := utilIPSet.NewEnnIPSet(execInterface)
	iptablesInterface := utilIPTables.NewEnnIPTables()

	protocol := k8sIPTables.ProtocolIpv4
	var k8siptInterface k8sIPTables.Interface
	var dbus utildbus.Interface

	dbus = utildbus.New()
	k8siptInterface = k8sIPTables.New(execInterface, dbus, protocol)

	ennpolicy := EnnPolicy{
		client:                  nil,
		hostName:                "",
		clusterCIDR:             "",
		iPRange:                 ipRange,
		networkPolicySynced:     true,
		podSynced:               true,
		namespaceSynced:         true,
		initAllSynced:           false,
		execInterface:           execInterface,
		ipsetInterface:          ipsetInterface,
		iptablesInterface:       iptablesInterface,
		k8siptablesInterface:    k8siptInterface,
		networkPolicyChanges:    utilpolicy.NewNetworkPolicyChangeMap(),
		podChanges:              utilpolicy.NewPodLabelChangeMap(),
		namespaceChanges:        utilpolicy.NewNamespaceChangeMap(),
		networkPolicyMap:        make(utilpolicy.NetworkPolicyMap),
		podMatchLabelMap:        make(utilpolicy.PodMatchLabelMap),
		namespaceMatchLabelMap:  make(utilpolicy.NamespaceMatchLabelMap),
		namespacePodMap:         make(utilpolicy.NamespacePodMap),
		namespaceInfoMap:        make(utilpolicy.NamespaceInfoMap),
		activeIPSets:            make(map[string]*utilIPSet.IPSet),
		podLabelSet:             make(map[utilpolicy.NamespacedLabel]*utilIPSet.IPSet),
		namespacePodLabelSet:    make(map[utilpolicy.Label]*utilIPSet.IPSet),
		namespacePodSet:         make(map[string]*utilIPSet.IPSet),
		existingFilterChains:    make(map[k8sIPTables.Chain]string),
		activeFilterChains :     make(map[k8sIPTables.Chain]bool),
		iptablesData:            bytes.NewBuffer(nil),
		filterChains:            bytes.NewBuffer(nil),
		filterRules:             bytes.NewBuffer(nil),
	}

	ennpolicy.setInitialized(ennpolicy.networkPolicySynced && ennpolicy.podSynced && ennpolicy.namespaceSynced)

	return &ennpolicy
}


// test ingress rules for podSelector
// will only test iptables rules
func TestNetworkPolicyAddIngressPodSelector(t *testing.T){

	fnp := NewFakeEnnPolicy("0.0.0.0/0")

	fnp.OnNetworkPolicyAdd(networkPolicies[0])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[0].Namespace, "ingress")

	if !ok{
		t.Errorf("Add Ingress PodSelector: get dispatch entry fail")
		return
	}

	if len(dispatchEntry) != 1{
		t.Errorf("Add Ingress PodSelector: dispatch entry number is %d, expected 1", len(dispatchEntry))
		return
	}

	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, dispatchEntry[0])
	//-N ENN-DPATCH-RRBZQTV5BG7K3RAG
	//-A ENN-DPATCH-RRBZQTV5BG7K3RAG -m comment --comment "accept rule selected by policy namespace0/np0: src pod match run1=test1" -m set --match-set ENN-PODSET-A4VMB22TEM7HZO7H src -j ACCEPT
	if len(lists) != 2{
		t.Errorf("Add Ingress PodSelector: dispatch entry len is %d, expected 2", len(lists))
		return
	}

	ok = checkDispatchChainPodSelector(
		t,
		lists[1],
		networkPolicies[0].Namespace,
		"ingress",
		"",
		"",
		"run1",
		"test1",
	)
	if !ok{
		t.Errorf("Add Ingress PodSelector: check dispatch error")
	}
}

// test ingress rules for namespaceSelector
// will only test iptables rules
func TestNetworkPolicyAddIngressNamespaceSelector(t *testing.T){

	fnp := NewFakeEnnPolicy("0.0.0.0/0")

	fnp.OnNetworkPolicyAdd(networkPolicies[1])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[1].Namespace, "ingress")

	if !ok{
		t.Errorf("Add Ingress NamespaceSelector: get dispatch entry fail")
		return
	}

	if len(dispatchEntry) != 1{
		t.Errorf("Add Ingress NamespaceSelector: dispatch entry number is %d, expected 1", len(dispatchEntry))
		return
	}

	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, dispatchEntry[0])

	if len(lists) != 2{
		t.Errorf("Add Ingress NamespaceSelector: dispatch entry len is %d, expected 2", len(lists))
		return
	}

	ok = checkDispatchChainNamespaceSelector(
		t,
		lists[1],
		networkPolicies[1].Namespace,
		"ingress",
		"run1",
		"test1",
		"ns1",
		"ns1",
	)
	if !ok{
		t.Errorf("Add Ingress NamespaceSelector: check dispatch error")
	}
}

// test ingress rules for ipBlock
// will only test iptables rules
func TestNetworkPolicyAddIngressIPBlock(t *testing.T){

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	fnp.OnNetworkPolicyAdd(networkPolicies[2])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[2].Namespace, "ingress")

	if !ok{
		t.Errorf("Add Ingress IPBlock: get dispatch entry fail")
		return
	}

	if len(dispatchEntry) != 1{
		t.Errorf("Add Ingress IPBlock: dispatch entry number is %d, expected 1", len(dispatchEntry))
		return
	}

	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, dispatchEntry[0])

	if len(lists) != 2{
		t.Errorf("Add Ingress IPBlock: dispatch entry len is %d, expected 2", len(lists))
		return
	}

	ok = checkDispatchChainIPBlock(
		t,
		lists[1],
		networkPolicies[2].Namespace,
		"ingress",
		"",
		"",
		"172.10.0.0/16",
	)

	if !ok{
		t.Errorf("Add Ingress IPBlock: check dispatch error")
	}
}

// test ingress rules when add "ports" in networkPolicy
// will only test iptables rules
func TestNetworkPolicyAddIngressPort(t *testing.T){

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	fnp.OnNetworkPolicyAdd(networkPolicies[3])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[3].Namespace, "ingress")

	if !ok{
		t.Errorf("Add Ingress Port: get dispatch entry fail")
		return
	}

	if len(dispatchEntry) != 1{
		t.Errorf("Add Ingress Port: dispatch entry number is %d, expected 1", len(dispatchEntry))
		return
	}

	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, dispatchEntry[0])

	if len(lists) != 2{
		t.Errorf("Add Ingress Port: dispatch entry len is %d, expected 2", len(lists))
		return
	}

	ok = checkDispatchChainPort(
		t,
		lists[1],
		networkPolicies[3].Namespace,
		"ingress",
		"",
		"",
		"tcp",
		"6789",
	)

	if !ok{
		t.Errorf("Add Ingress Port: check dispatch error")
	}
}

// test egress rules for podSelector
// will only test iptables rules
func TestNetworkPolicyAddEgressPodSelector(t *testing.T){

	fnp := NewFakeEnnPolicy("0.0.0.0/0")

	fnp.OnNetworkPolicyAdd(networkPolicies[4])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[4].Namespace, "egress")

	if !ok{
		t.Errorf("Add Egress PodSelector: get dispatch entry fail")
		return
	}

	if len(dispatchEntry) != 2{
		t.Errorf("Add Egress PodSelector: dispatch entry number is %d, expected 2", len(dispatchEntry))
		return
	}

	ennEntry := ennDispatchChainName(
		networkPolicies[4].Namespace,
		networkPolicies[4].Name,
		strconv.Itoa(TYPE_EGRESS),
		"podSelector",
		"",
		"run1",
		"test1")
	for _, entry := range dispatchEntry{

		if entry == ennEntry{
			lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
			if len(lists) != 2{
				t.Errorf("Add Egress PodSelector: dispatch entry len is %d, expected 2, entry:%s", len(lists), entry)
				return
			}

			ok = checkDispatchChainPodSelector(
				t,
				lists[1],
				networkPolicies[4].Namespace,
				"egress",
				"",
				"",
				"run1",
				"test1",
			)
			if !ok{
				t.Errorf("Add Egress PodSelector: check dispatch error")
			}
		} else{
			lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
			if len(lists) != 2{
				t.Errorf("Add Egress PodSelector: dispatch entry len is %d, expected 2, entry:%s", len(lists), entry)
				return
			}

			ok = checkDispatchChainPodSelector(
				t,
				lists[1],
				networkPolicies[4].Namespace,
				"egress",
				"",
				"",
				"run3",
				"test3",
			)
			if !ok{
				t.Errorf("Add Egress PodSelector: check dispatch error")
			}
		}
	}

}


// test egress rules for namespaceSelector
// will only test iptables rules
func TestNetworkPolicyAddEgressNamespaceSelector(t *testing.T){

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	fnp.OnNetworkPolicyAdd(networkPolicies[5])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[5].Namespace, "egress")

	if !ok{
		t.Errorf("Add Egress NamespaceSelector: get dispatch entry fail")
		return
	}

	if len(dispatchEntry) != 2{
		t.Errorf("Add Egress NamespaceSelector: dispatch entry number is %d, expected 2", len(dispatchEntry))
		return
	}

	ennEntry := ennDispatchChainName(
		networkPolicies[5].Namespace,
		networkPolicies[5].Name,
		strconv.Itoa(TYPE_EGRESS),
		"namespaceSelector",
		"",
		"ns2",
		"ns2")
	for _, entry := range dispatchEntry {
		if entry == ennEntry{
			// dispatch for namespace selector n2=n2
			// -A ENN-PLY-E-FLYGA4ITYZL5AVLH -m comment --comment "entry for namespaceSelector" -j ENN-DPATCH-ZYW3R7UAHYFD5CLT
			lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)

			if len(lists) != 3{
				t.Errorf("Add Egress NamespaceSelector: dispatch entry len is %d, expected 3, entry: %s", len(lists), entry)
				return
			}

			for _, list := range lists{
				if strings.Contains(list, "run1=test1"){
					// -A ENN-DPATCH-ZYW3R7UAHYFD5CLT -m comment --comment "accept rule selected by policy namespace2/np5: src match run1=test1, dst namespace match ns2=ns2"
					// -m set --match-set ENN-PODSET-3WH7O4RMU7J5Q4P4 src -m set --match-set ENN-NSSET-2DX3JVYC4LR6FVW3 dst -j ACCEPT
					ok = checkDispatchChainNamespaceSelector(
						t,
						list,
						networkPolicies[5].Namespace,
						"egress",
						"run1",
						"test1",
						"ns2",
						"ns2",
					)
					if !ok{
						t.Errorf("Add Egress NamespaceSelector: check dispatch error")
					}
				} else if strings.Contains(list, "run2=test2"){
					// -A ENN-DPATCH-ZYW3R7UAHYFD5CLT -m comment --comment "accept rule selected by policy namespace2/np5: src match run2=test2, dst namespace match ns2=ns2"
					// -m set --match-set ENN-PODSET-IXGJO7JWEV77LVL5 src -m set --match-set ENN-NSSET-2DX3JVYC4LR6FVW3 dst -j ACCEPT
					ok = checkDispatchChainNamespaceSelector(
						t,
						list,
						networkPolicies[5].Namespace,
						"egress",
						"run2",
						"test2",
						"ns2",
						"ns2",
					)
					if !ok{
						t.Errorf("Add Egress NamespaceSelector: check dispatch error")
					}
				} else {
					if !strings.Contains(list, "-N"){
						t.Errorf("invalid rule for entry %s", entry)
						t.Errorf("%s",list)
					}
				}
			}
		} else {
			// dispatch for namespace selector n3=n3
			// -A ENN-PLY-E-FLYGA4ITYZL5AVLH -m comment --comment "entry for namespaceSelector" -j ENN-DPATCH-2VR6PQKQMSWHOHWA
			lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)

			if len(lists) != 3{
				t.Errorf("Add Egress NamespaceSelector: dispatch entry len is %d, expected 3, entry: %s", len(lists), entry)
				return
			}

			for _, list := range lists{
				if strings.Contains(list, "run1=test1"){
					// -A ENN-DPATCH-2VR6PQKQMSWHOHWA -m comment --comment "accept rule selected by policy namespace2/np5: src match run1=test1,
					// dst namespace match ns3=ns3" -m set --match-set ENN-PODSET-3WH7O4RMU7J5Q4P4 src -m set --match-set ENN-NSSET-BMTEHLTEP3GJQJC7 dst -j ACCEPT
					ok = checkDispatchChainNamespaceSelector(
						t,
						list,
						networkPolicies[5].Namespace,
						"egress",
						"run1",
						"test1",
						"ns3",
						"ns3",
					)
					if !ok{
						t.Errorf("Add Egress NamespaceSelector: check dispatch error")
					}
				} else if strings.Contains(list, "run2=test2"){
					// -A ENN-DPATCH-2VR6PQKQMSWHOHWA -m comment --comment "accept rule selected by policy namespace2/np5: src match run2=test2,
					// dst namespace match ns3=ns3" -m set --match-set ENN-PODSET-IXGJO7JWEV77LVL5 src -m set --match-set ENN-NSSET-BMTEHLTEP3GJQJC7 dst -j ACCEPT
					ok = checkDispatchChainNamespaceSelector(
						t,
						list,
						networkPolicies[5].Namespace,
						"egress",
						"run2",
						"test2",
						"ns3",
						"ns3",
					)
					if !ok{
						t.Errorf("Add Egress NamespaceSelector: check dispatch error")
					}
				} else {
					if !strings.Contains(list, "-N"){
						t.Errorf("invalid rule for entry %s", entry)
						t.Errorf("%s",list)
					}
				}
			}
		}
	}
}


// test egress rules for ipBlock
// will only test iptables rules
func TestNetworkPolicyAddEgressIPBlock(t *testing.T){

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	fnp.OnNetworkPolicyAdd(networkPolicies[6])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[6].Namespace, "egress")

	if !ok{
		t.Errorf("Add Egress IPBlock: get dispatch entry fail")
		return
	}

	if len(dispatchEntry) != 1{
		t.Errorf("Add Egress IPBlock: dispatch entry number is %d, expected 1", len(dispatchEntry))
		return
	}

	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, dispatchEntry[0])

	if len(lists) != 2{
		t.Errorf("Add Egress IPBlock: dispatch entry len is %d, expected 2", len(lists))
		return
	}

	ok = checkDispatchChainIPBlock(
		t,
		lists[1],
		networkPolicies[6].Namespace,
		"egress",
		"run1",
		"test1",
		"198.168.2.0/24",
	)

	if !ok{
		t.Errorf("Add Egress IPBlock: check dispatch error")
	}
}

// test egress rules when add "ports" in networkPolicy
// will only test iptables rules
func TestNetworkPolicyAddEgressPort(t *testing.T){

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	fnp.OnNetworkPolicyAdd(networkPolicies[7])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[7].Namespace, "egress")

	if !ok{
		t.Errorf("Add Egress Port: get dispatch entry fail")
		return
	}

	if len(dispatchEntry) != 1{
		t.Errorf("Add Egress Port: dispatch entry number is %d, expected 1", len(dispatchEntry))
		return
	}

	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, dispatchEntry[0])

	if len(lists) != 2{
		t.Errorf("Add Egress Port: dispatch entry len is %d, expected 2", len(lists))
		return
	}

	ok = checkDispatchChainPort(
		t,
		lists[1],
		networkPolicies[7].Namespace,
		"egress",
		"run1",
		"test1",
		"udp",
		"1234",
	)

	if !ok{
		t.Errorf("Add Egress Port: check dispatch error")
	}

	ok = checkDispatchChainNamespaceSelector(
		t,
		lists[1],
		networkPolicies[7].Namespace,
		"egress",
		"run1",
		"test1",
		"ns1",
		"ns1",
	)
	if !ok{
		t.Errorf("Add Egress NamespaceSelector: check dispatch error")
	}

}

// this test case have 2 parts
// 1. will add 11 networkPolicies one by one, including all kinds of networkPolicies, then check whether iptables is correct
// 2. delete some networkPolicies in different namespaces, then check whether iptables is correct
func TestNetworkPolicyAddDelete(t *testing.T){
	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	for i := 0; i < 11; i++{
		fnp.OnNetworkPolicyAdd(networkPolicies[i])
	}

	// namespace0 ingress
	dispatchEntry, ok := getDispatchEntry(t, fnp, namespaceName[0], "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 2{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[0], len(dispatchEntry), 2)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 2 {
			t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 2)
		}
	}

	// namespace0 egress
	dispatchEntry, ok = getDispatchEntry(t, fnp, namespaceName[0], "egress")
	if !ok{
		t.Errorf("Egress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 2{
		t.Errorf("Egress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[0], len(dispatchEntry), 2)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 2 {
			t.Errorf("Egress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 2)
		}
	}

	// namespace1 ingress
	dispatchEntry, ok = getDispatchEntry(t, fnp, namespaceName[1], "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 2{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[1], len(dispatchEntry), 2)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 2 {
			t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 2)
		}
	}

	// namespace2 egress
	dispatchEntry, ok = getDispatchEntry(t, fnp, namespaceName[2], "egress")
	if !ok{
		t.Errorf("Egress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 4{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[2], len(dispatchEntry), 4)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if entry == ennDispatchChainName(
			networkPolicies[5].Namespace,
			networkPolicies[5].Name,
			strconv.Itoa(TYPE_EGRESS),
			"namespaceSelector",
			"",
			"ns2",
			"ns2"){
			if len(lists) != 3 {
				t.Errorf("Egress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
			}

		} else if entry == ennDispatchChainName(
			networkPolicies[5].Namespace,
			networkPolicies[5].Name,
			strconv.Itoa(TYPE_EGRESS),
			"namespaceSelector",
			"",
			"ns3",
			"ns3"){
			if len(lists) != 3 {
				t.Errorf("Egress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
			}

		} else {
			if len(lists) != 2 {
				t.Errorf("Egress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 2)
			}
		}
	}

	// namespace3 egress
	dispatchEntry, ok = getDispatchEntry(t, fnp, namespaceName[3], "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 4{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[3], len(dispatchEntry), 4)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if entry == ennDispatchChainName(
			networkPolicies[9].Namespace,
			networkPolicies[9].Name,
			strconv.Itoa(TYPE_INGRESS),
			"namespaceSelector",
			"",
			"ns3",
			"ns33"){
			if len(lists) != 2 {
				t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 2)
			}
		} else {
			if len(lists) != 3 {
				t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
			}
		}
	}

	// namespace4 egress
	dispatchEntry, ok = getDispatchEntry(t, fnp, namespaceName[4], "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 2{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[4], len(dispatchEntry), 2)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 3 {
			t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
		}
	}

	fnp.OnNetworkPolicyDelete(networkPolicies[0])
	fnp.OnNetworkPolicyDelete(networkPolicies[4])
	fnp.OnNetworkPolicyDelete(networkPolicies[8])

	dispatchEntry, ok = getDispatchEntry(t, fnp, namespaceName[0], "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 1{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[0], len(dispatchEntry), 1)
	}

	dispatchEntry, ok = getDispatchEntry(t, fnp, namespaceName[2], "egress")
	if !ok{
		t.Errorf("Egress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 2{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[2], len(dispatchEntry), 2)
	}

	dispatchEntry, ok = getDispatchEntry(t, fnp, namespaceName[3], "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", namespaceName[0])
		return
	}
	if len(dispatchEntry) != 1{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", namespaceName[3], len(dispatchEntry), 1)
	}
}

// this test case have 4 parts
// 0. add one networkPolicy (ingress rule with namespaceSelector)
// 1. update networkPolicy without change, so iptables should keep as the same
// 2. update networkPolicy (delete some labels from namespaceSelector), then check iptables
// 3. update networkPolicy (add some labels from namespaceSelector), then check ipatebles
// 4. update networkPolicy with the initial one, so iptables should be as the same as step0
func TestNetworkPolicyUpdate(t *testing.T){

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	fnp.OnNetworkPolicyAdd(networkPolicies[10])

	dispatchEntry, ok := getDispatchEntry(t, fnp, networkPolicies[10].Namespace, "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", networkPolicies[10].Namespace)
		return
	}
	if len(dispatchEntry) != 2{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", networkPolicies[10].Namespace, len(dispatchEntry), 2)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 3 {
			t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
		}
	}

	// update without change
	fnp.OnNetworkPolicyUpdate(networkPolicies[10],networkPolicies[10])

	dispatchEntry, ok = getDispatchEntry(t, fnp, networkPolicies[10].Namespace, "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", networkPolicies[10].Namespace)
		return
	}
	if len(dispatchEntry) != 2{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", networkPolicies[10].Namespace, len(dispatchEntry), 2)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 3 {
			t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
		}
	}

	// update delete namespace label
	fnp.OnNetworkPolicyUpdate(networkPolicies[10],networkPolicies[11])

	dispatchEntry, ok = getDispatchEntry(t, fnp, networkPolicies[11].Namespace, "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", networkPolicies[11].Namespace)
		return
	}
	if len(dispatchEntry) != 1{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", networkPolicies[11].Namespace, len(dispatchEntry), 1)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 3 {
			t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
		}
	}

	// update add namespace label
	fnp.OnNetworkPolicyUpdate(networkPolicies[11],networkPolicies[12])

	dispatchEntry, ok = getDispatchEntry(t, fnp, networkPolicies[12].Namespace, "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", networkPolicies[12].Namespace)
		return
	}
	if len(dispatchEntry) != 3{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", networkPolicies[12].Namespace, len(dispatchEntry), 3)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 3 {
			t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
		}
	}

	// update back to networkPolicy #10
	fnp.OnNetworkPolicyUpdate(networkPolicies[12],networkPolicies[10])

	dispatchEntry, ok = getDispatchEntry(t, fnp, networkPolicies[10].Namespace, "ingress")
	if !ok{
		t.Errorf("Ingress: get dispatch entry for namespace %s fail", networkPolicies[10].Namespace)
		return
	}
	if len(dispatchEntry) != 2{
		t.Errorf("Ingress: dispatch entry for namespace %s len incorrect %d, expect %d", networkPolicies[10].Namespace, len(dispatchEntry), 2)
	}
	for _, entry := range dispatchEntry{
		lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, entry)
		if len(lists) != 3 {
			t.Errorf("Ingress: rule number for dispatch entry:%s is not correct: %d, expect %d", entry, len(lists), 3)
		}
		if entry == ennDispatchChainName(
			networkPolicies[10].Namespace,
			networkPolicies[10].Name,
			strconv.Itoa(TYPE_INGRESS),
			"namespaceSelector",
			"",
			"ns2",
			"ns22"){
			for _, list := range lists{
				if strings.Contains(list, "run1=test11"){
					ok = checkDispatchChainNamespaceSelector(
						t,
						list,
						networkPolicies[10].Namespace,
						"ingress",
						"run1",
						"test11",
						"ns2",
						"ns22",
					)
					if !ok{
						t.Errorf("update networkPolicy: check dispatch error")
					}
				} else if strings.Contains(list, "run2=test2"){
					ok = checkDispatchChainNamespaceSelector(
						t,
						list,
						networkPolicies[10].Namespace,
						"ingress",
						"run2",
						"test2",
						"ns2",
						"ns22",
					)
					if !ok{
						t.Errorf("update networkPolicy: check dispatch error")
					}
				} else {
					if !strings.Contains(list, "-N"){
						t.Errorf("invalid rule for entry %s", entry)
						t.Errorf("%s",list)
					}
				}
			}
		} else {
			for _, list := range lists{
				if strings.Contains(list, "run1=test11"){
					ok = checkDispatchChainNamespaceSelector(
						t,
						list,
						networkPolicies[10].Namespace,
						"ingress",
						"run1",
						"test11",
						"ns3",
						"ns33",
					)
					if !ok{
						t.Errorf("update networkPolicy: check dispatch error")
					}
				} else if strings.Contains(list, "run2=test2"){
					ok = checkDispatchChainNamespaceSelector(
						t,
						list,
						networkPolicies[10].Namespace,
						"ingress",
						"run2",
						"test2",
						"ns3",
						"ns33",
					)
					if !ok{
						t.Errorf("update networkPolicy: check dispatch error")
					}
				} else {
					if !strings.Contains(list, "-N"){
						t.Errorf("invalid rule for entry %s", entry)
						t.Errorf("%s",list)
					}
				}
			}
		}

	}
}

// TestPodAdd will add pod into EnnPolicy and then whether corresponding ipset is correct
// ipset include namespace ipset, pod label ipset, namespace label ipset
// will check both ipset and entry
func TestPodAdd(t *testing.T){

	namespaces := []*coreApi.Namespace{
		//0
		makeTestNamespace(namespaceName[2], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns1":"ns1"}
		}),
		//1
		makeTestNamespace(namespaceName[3], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns3":"ns33"}
		}),
		//2
		makeTestNamespace(namespaceName[4], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns3":"ns33","ns1":"ns1"}
		}),
	}

	pods := []*coreApi.Pod{
		// 0 this pod in podLabelSet namespace[3]:run1=test11 and namespaceLabelSet ns3=ns33
		makeTestPod(namespaceName[3], "pod0", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test11","run2":"test2"}
			pod.Status.PodIP = "10.0.0.0"
		}),
		// 1 this pod in namespaceLabelSet ns3=ns33
		makeTestPod(namespaceName[3], "pod1", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run2":"test2"}
			pod.Status.PodIP = "10.0.0.1"
		}),
		// 2 this pod in podLabelSet namespace[3]:run1=test11 and namespaceLabelSet ns3=ns33
		makeTestPod(namespaceName[3], "pod2", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test11"}
			pod.Status.PodIP = "10.0.0.2"
		}),
		// 3 this pod in namespaceLabelSet ns3=ns33
		makeTestPod(namespaceName[3], "pod3", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1","run3":"test3"}
			pod.Status.PodIP = "10.0.0.3"
		}),
		// 4 this pod in podLabelSet namespace[3]:run1=test11 and namespaceLabelSet ns3=ns33
		makeTestPod(namespaceName[3], "pod4", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test11","run3":"test3"}
			pod.Status.PodIP = "10.0.0.4"
		}),
		// 5 this pod in namespaceLabelSet ns3=ns33
		makeTestPod(namespaceName[4], "pod5", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.5"
		}),
		// 6 this pod in namespaceLabelSet ns3=ns33
		makeTestPod(namespaceName[4], "pod6", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.6"
		}),
		// 7 this pod none of the set
		makeTestPod(namespaceName[2], "pod7", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.7"
		}),
	}

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	for i := 0; i < 3; i++{
		fnp.OnNamespaceAdd(namespaces[i])
	}
	// first add two pods
	for i := 0; i < 2; i++{
		fnp.OnPodAdd(pods[i])
	}
	// np.Spec.PodSelector : map[string]string{"run1":"test11"}
	// NamespaceSelector: map[string]string{"ns3":"ns33"}
	fnp.OnNetworkPolicyAdd(networkPolicies[9])

	ok := checkIPRangeIPSet(t, fnp, "10.244.0.0/16", "10.244.0.0/16")
	if !ok{
		t.Errorf("check ipRange %s IPSet fail", "10.244.0.0/16")
	}
	ok = checkNamespacePodIPSet(t, fnp, namespaceName[3], "10.0.0.0", "10.0.0.1")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test11", "10.0.0.0")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test11")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns33", "10.0.0.0", "10.0.0.1")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns33")
	}

	// then add rest 6 pods
	for i := 2; i < 8; i++{
		fnp.OnPodAdd(pods[i])
	}

	ok = checkNamespacePodIPSet(t, fnp, namespaceName[3],
		"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test11",
		"10.0.0.0", "10.0.0.2", "10.0.0.4")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test11")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns33",
		"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns33")
	}
}

// TestPodDelete will delete some pods from given iptables and ipsets
// and will check whether ipset and entry is correct
func TestPodDelete(t *testing.T){

	namespaces := []*coreApi.Namespace{
		//0
		makeTestNamespace(namespaceName[1], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns1":"ns1"}
		}),
		//1
		makeTestNamespace(namespaceName[2], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2"}
		}),
		//2
		makeTestNamespace(namespaceName[3], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2","ns3":"ns3"}
		}),
		//3
		makeTestNamespace(namespaceName[4], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns22","ns3":"ns3"}
		}),
	}

	pods := []*coreApi.Pod{
		// 0 this pod in podLabelSet namespace[3]:run2=test2 and namespaceLabelSet ns2=ns2,ns3=ns3
		makeTestPod(namespaceName[3], "pod0", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test11","run2":"test2"}
			pod.Status.PodIP = "10.0.0.0"
		}),
		// 1 this pod in podLabelSet namespace[3]:run2=test2 and namespaceLabelSet ns2=ns2,ns3=ns3
		makeTestPod(namespaceName[3], "pod1", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run2":"test2"}
			pod.Status.PodIP = "10.0.0.1"
		}),
		// 2 this pod in podLabelSet namespace[3]:run1=test1 and namespaceLabelSet ns2=ns2,ns3=ns3
		makeTestPod(namespaceName[3], "pod2", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1"}
			pod.Status.PodIP = "10.0.0.2"
		}),
		// 3 this pod in podLabelSet namespace[3]:run1=test1 and namespaceLabelSet ns2=ns2,ns3=ns3
		makeTestPod(namespaceName[3], "pod3", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1","run3":"test3"}
			pod.Status.PodIP = "10.0.0.3"
		}),
		// 4 this pod in namespaceLabelSet ns2=ns2,ns3=ns3
		makeTestPod(namespaceName[3], "pod4", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test11","run3":"test3"}
			pod.Status.PodIP = "10.0.0.4"
		}),
		// 5 this pod in namespaceLabelSet ns2=ns22,ns3=ns3
		makeTestPod(namespaceName[4], "pod5", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.5"
		}),
		// 6 this pod in namespaceLabelSet ns2=ns22,ns3=ns3
		makeTestPod(namespaceName[4], "pod6", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.6"
		}),
		// 7 this pod in namespaceLabelSet ns2=ns2
		makeTestPod(namespaceName[2], "pod7", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.7"
		}),
		// 8 this pod in none of set
		makeTestPod(namespaceName[1], "pod8", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.8"
		}),
	}

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	//fnp.CleanupLeftovers()

	for i := 0; i < 4; i++{
		fnp.OnNamespaceAdd(namespaces[i])
	}
	for i := 0; i < 8; i++{
		fnp.OnPodAdd(pods[i])
	}
	//np.Spec.PodSelector: map[string]string{"run1":"test1","run2":"test2"}}
	//NamespaceSelector: map[string]string{"ns2":"ns2","ns3":"ns3"}},
	//NamespaceSelector: map[string]string{"ns2":"ns22"}},

	fnp.OnNetworkPolicyAdd(networkPolicies[8])

	ok := checkNamespacePodIPSet(t, fnp, namespaceName[3],
		"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test1",
		"10.0.0.2", "10.0.0.3")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test1")
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run2", "test2",
		"10.0.0.0", "10.0.0.1")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run2","test2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2",
		"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.7")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3",
		"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22",
		"10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}

	// begin to delete some pods
	fnp.OnPodDelete(pods[0])
	fnp.OnPodDelete(pods[2])
	fnp.OnPodDelete(pods[4])
	fnp.OnPodDelete(pods[6])

	ok = checkNamespacePodIPSet(t, fnp, namespaceName[3],
		"10.0.0.1", "10.0.0.3")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test1",
		"10.0.0.3")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test1")
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run2", "test2",
		"10.0.0.1")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run2","test2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2",
		"10.0.0.1", "10.0.0.3", "10.0.0.7")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3",
		"10.0.0.1",  "10.0.0.3", "10.0.0.5")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22",
		"10.0.0.5")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}
}

// this test case have 4 parts:
// 1. update pods without change, so corresponding ipset should keep as the same
// 2. update pods: delete pod label, and check whether corresponding ipset is correct
// 3. update pods: add pod label, and check whether corresponding ipset is correct
// 4. update pods to #1, so corresponding ipset should be as same as #1
func TestPodUpdate(t *testing.T){

	namespaces := []*coreApi.Namespace{
		//0
		makeTestNamespace(namespaceName[3], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2","ns3":"ns3"}
		}),
	}

	pods := []*coreApi.Pod{
		// 0 this pod in podLabelSet namespace[3]:run2=test2 and namespaceLabelSet ns2=ns2,ns3=ns3
		makeTestPod(namespaceName[3], "pod1", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run2":"test2"}
			pod.Status.PodIP = "10.0.0.1"
		}),
		// 1 this pod in podLabelSet namespace[3]:run1=test1 and namespaceLabelSet ns2=ns2,ns3=ns3
		makeTestPod(namespaceName[3], "pod2", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1"}
			pod.Status.PodIP = "10.0.0.2"
		}),
		// 2 this pod in podLabelSet namespace[3]:run1=test1 and namespaceLabelSet ns2=ns2,ns3=ns3
		makeTestPod(namespaceName[3], "pod3", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1","run3":"test3"}
			pod.Status.PodIP = "10.0.0.3"
		}),
		// 3 this pod in namespaceLabelSet ns2=ns2,ns3=ns3
		// #2 delete pod label run1=test1
		makeTestPod(namespaceName[3], "pod3", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run3":"test3"}
			pod.Status.PodIP = "10.0.0.3"
		}),
		// 4 this pod in podLabelSet namespace[3]:run1=test1, run2=test2 and namespaceLabelSet ns2=ns2,ns3=ns3
		// #3 add pod label run1=test1 run2=test2
		makeTestPod(namespaceName[3], "pod3", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1","run2":"test2","run4":"test4"}
			pod.Status.PodIP = "10.0.0.3"
		}),
	}

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	//np.Spec.PodSelector: map[string]string{"run1":"test1","run2":"test2"}}
	//NamespaceSelector: map[string]string{"ns2":"ns2","ns3":"ns3"}},
	//NamespaceSelector: map[string]string{"ns2":"ns22"}},

	fnp.OnNamespaceAdd(namespaces[0])
	fnp.OnPodAdd(pods[0])
	fnp.OnPodAdd(pods[1])
	fnp.OnPodAdd(pods[2])
	fnp.OnNetworkPolicyAdd(networkPolicies[8])

	ok := checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test1",
		"10.0.0.2", "10.0.0.3")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test1")
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run2", "test2",
		"10.0.0.1")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run2","test2")
	}

	//update pod without change
	fnp.OnPodUpdate(pods[2],pods[2])
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test1",
		"10.0.0.2", "10.0.0.3")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test1")
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run2", "test2",
		"10.0.0.1")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run2","test2")
	}

	//update pod : delete label
	fnp.OnPodUpdate(pods[2],pods[3])
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test1",
		"10.0.0.2")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test1")
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run2", "test2",
		"10.0.0.1")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run2","test2")
	}

	//update pod : add label
	fnp.OnPodUpdate(pods[3],pods[4])
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test1",
		"10.0.0.2", "10.0.0.3")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test1")
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run2", "test2",
		"10.0.0.1", "10.0.0.3")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run2","test2")
	}

	//update pod : back to pods[2]
	fnp.OnPodUpdate(pods[4],pods[2])
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run1", "test1",
		"10.0.0.2", "10.0.0.3")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run1","test1")
	}
	ok = checkPodLabelIPSet(t, fnp, namespaceName[3], "run2", "test2",
		"10.0.0.1")
	if !ok{
		t.Errorf("check pod label set %s:%s=%s IPSet fail", namespaceName[3],"run2","test2")
	}
}

// this test case has 4 step
// 1. add new pod with empty ip, so enn-policy should do nothing
// 2. update pod ip from empty ip to real ip, so enn-policy will add this pod into corresponding map
// 3. update pod ip from real ip to empty ip, so enn-policy will delete this pod from corresponding map
// 4. delete pod with empty ip, so enn-policy will do nothing since pod already deleted in step3

// BTW, in real situation, when kubernetes create a new pod, enn-policy will watch
// 1) OnPodAdd() <- assign pod name but with empty ip, enn-policy do nothing
// 2) OnPodUpdate() <- change pod status, enn-policy do nothing
// 3) OnPodUpdate() <- second time change pod status, enn-policy do nothing
// 4) OnPodUpdate() <- assign ip to this pod, enn-policy will to add pod operation
// and when kubernetes delete a pod, enn-policy will watch
// 1) OnPodUpdate() <- change pod status, enn-policy do nothing
// 2) OnPodUpdate() <- delete pod ip, enn-policy will do delete pod operation
// 3) OnPodUpdate() <- second time change pod status, enn-policy do nothing
// 4) OnPodDelete() <- finally remove pod from kubernetes, but enn-policy still do nothing since pod already deleted in 2)
func TestEmptyIPPod(t *testing.T){

	namespaces := []*coreApi.Namespace{
		//0
		makeTestNamespace(namespaceName[3], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2"}
		}),
	}

	pods := []*coreApi.Pod{
		// 0 this pod should not in podLabelSet namespace[3]:run2=test2 and namespaceLabelSet ns2=ns2
		makeTestPod(namespaceName[3], "pod1", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run2":"test2"}
		}),
		// 1 this pod in podLabelSet namespace[3]:run2=test2 and namespaceLabelSet ns2=ns2
		makeTestPod(namespaceName[3], "pod1", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run2":"test2"}
			pod.Status.PodIP = "10.0.0.1"
		}),

	}

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	fnp.OnNamespaceAdd(namespaces[0])
	fnp.OnNetworkPolicyAdd(networkPolicies[8])

	// 1. add empty ip pod
	fnp.OnPodAdd(pods[0])

	namespacedLabel := utilpolicy.NamespacedLabel{
		Namespace:   namespaceName[3],
		LabelKey:    "run2",
		LabelValue:  "test2",
	}

	label := utilpolicy.Label{
		LabelKey:     "ns2",
		LabelValue:   "ns2",
	}

	_, ok := fnp.podMatchLabelMap[namespacedLabel]
	if ok{
		t.Errorf("find unexpected podInfoMap in podMatchLabelMap: %s:%s=%s", namespaceName[3], "run2", "test2")
	}

	_, ok = fnp.namespaceMatchLabelMap[label]
	if ok{
		t.Errorf("find unexpected podInfoMap in namespaceMatchLabelMap: %s=%s", "ns2", "ns2")
	}

	_, ok = fnp.namespacePodMap[namespaceName[3]]
	if ok{
		t.Errorf("find unexpected podInfoMap in namespacePodMap: %s", namespaceName[3])
	}

	// 2. update empty ip to real ip
	fnp.OnPodUpdate(pods[0], pods[1])

	infoMap, ok := fnp.podMatchLabelMap[namespacedLabel]
	if !ok{
		t.Errorf("cannot find podInfoMap in podMatchLabelMap: %s:%s=%s", namespaceName[3], "run2", "test2")
	} else {
		if len(infoMap) != 1{
			t.Errorf("podInfoMap in podMatchLabelMap: %s:%s=%s length is not correct:%d expect 1", namespaceName[3], "run2", "test2", len(infoMap))
		}
	}

	infoMap, ok = fnp.namespaceMatchLabelMap[label]
	if !ok{
		t.Errorf("cannot find podInfoMap in namespaceMatchLabelMap: %s=%s", "ns2", "ns2")
	} else {
		if len(infoMap) != 1{
			t.Errorf("podInfoMap in namespaceMatchLabelMap: %s=%s length is not correct:%d expect 1", "run2", "test2", len(infoMap))
		}
	}

	infoMap, ok = fnp.namespacePodMap[namespaceName[3]]
	if !ok{
		t.Errorf("cannot find podInfoMap in namespacePodMap: %s", namespaceName[3])
	} else {
		if len(infoMap) != 1{
			t.Errorf("podInfoMap in namespacePodMap: %s length is not correct:%d expect 1", namespaceName[3], len(infoMap))
		}
	}

	// 3. update real ip to empty
	fnp.OnPodUpdate(pods[1], pods[0])

	infoMap, ok = fnp.podMatchLabelMap[namespacedLabel]
	if !ok{
		t.Errorf("cannot find podInfoMap in podMatchLabelMap: %s:%s=%s", namespaceName[3], "run2", "test2")
	} else {
		if len(infoMap) != 0{
			t.Errorf("podInfoMap in podMatchLabelMap: %s:%s=%s length is not correct:%d expect 0", namespaceName[3], "run2", "test2", len(infoMap))
		}
	}

	infoMap, ok = fnp.namespaceMatchLabelMap[label]
	if !ok{
		t.Errorf("cannot find podInfoMap in namespaceMatchLabelMap: %s=%s", "ns2", "ns2")
	} else {
		if len(infoMap) != 0{
			t.Errorf("podInfoMap in namespaceMatchLabelMap: %s=%s length is not correct:%d expect 0", "run2", "test2", len(infoMap))
		}
	}

	infoMap, ok = fnp.namespacePodMap[namespaceName[3]]
	if !ok{
		t.Errorf("cannot find podInfoMap in namespacePodMap: %s", namespaceName[3])
	} else {
		if len(infoMap) != 0{
			t.Errorf("podInfoMap in namespacePodMap: %s length is not correct:%d expect 0", namespaceName[3], len(infoMap))
		}
	}

	// 4. delete pod
	fnp.OnPodDelete(pods[0])

	infoMap, ok = fnp.podMatchLabelMap[namespacedLabel]
	if !ok{
		t.Errorf("cannot find podInfoMap in podMatchLabelMap: %s:%s=%s", namespaceName[3], "run2", "test2")
	} else {
		if len(infoMap) != 0{
			t.Errorf("podInfoMap in podMatchLabelMap: %s:%s=%s length is not correct:%d expect 0", namespaceName[3], "run2", "test2", len(infoMap))
		}
	}

	infoMap, ok = fnp.namespaceMatchLabelMap[label]
	if !ok{
		t.Errorf("cannot find podInfoMap in namespaceMatchLabelMap: %s=%s", "ns2", "ns2")
	} else {
		if len(infoMap) != 0{
			t.Errorf("podInfoMap in namespaceMatchLabelMap: %s=%s length is not correct:%d expect 0", "run2", "test2", len(infoMap))
		}
	}

	infoMap, ok = fnp.namespacePodMap[namespaceName[3]]
	if !ok{
		t.Errorf("cannot find podInfoMap in namespacePodMap: %s", namespaceName[3])
	} else {
		if len(infoMap) != 0{
			t.Errorf("podInfoMap in namespacePodMap: %s length is not correct:%d expect 0", namespaceName[3], len(infoMap))
		}
	}
}

// this test case will add some namespace and delete some namespace, then check whether corresponding ipset is correct
// ipset include namespace ipset and namespace label ipset
// when we delete a namespace, we must first delete all pods in this namespace
func TestNamespaceAddDelete(t *testing.T){

	namespaces := []*coreApi.Namespace{
		//0
		makeTestNamespace(namespaceName[1], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns1":"ns1"}
		}),
		//1
		makeTestNamespace(namespaceName[2], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2"}
		}),
		//2
		makeTestNamespace(namespaceName[3], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2","ns3":"ns3"}
		}),
		//3
		makeTestNamespace(namespaceName[4], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns22","ns3":"ns3"}
		}),
	}

	pods := []*coreApi.Pod{
		// 0 namespace1
		makeTestPod(namespaceName[1], "pod0", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.1"
		}),
		// 1 namespace1
		makeTestPod(namespaceName[1], "pod1", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.2"
		}),
		// 2 namespace2
		makeTestPod(namespaceName[2], "pod2", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.3"
		}),
		// 3 namespace2
		makeTestPod(namespaceName[2], "pod3", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.4"
		}),
		// 4 namespace3
		makeTestPod(namespaceName[3], "pod4", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1","run2":"test2"}
			pod.Status.PodIP = "10.0.0.5"
		}),
		// 5 namespace3
		makeTestPod(namespaceName[3], "pod5", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1","run2":"test2"}
			pod.Status.PodIP = "10.0.0.6"
		}),
		// 6 namespace4
		makeTestPod(namespaceName[4], "pod6", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.7"
		}),
		// 7 namespace4
		makeTestPod(namespaceName[4], "pod7", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.8"
		}),
	}

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	for i := 0; i < 4; i++{
		fnp.OnNamespaceAdd(namespaces[i])
	}
	for i := 0; i < 8; i++{
		fnp.OnPodAdd(pods[i])
	}

	//np.Spec.PodSelector: map[string]string{"run1":"test1","run2":"test2"}}
	//NamespaceSelector: map[string]string{"ns2":"ns2","ns3":"ns3"}},
	//NamespaceSelector: map[string]string{"ns2":"ns22"}},
	fnp.OnNetworkPolicyAdd(networkPolicies[8])

	ok := checkNamespacePodIPSet(t, fnp, namespaceName[3], "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3", "10.0.0.5", "10.0.0.6", "10.0.0.7", "10.0.0.8")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22", "10.0.0.7", "10.0.0.8")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}

	// delete namespace4
	fnp.OnPodDelete(pods[6])
	fnp.OnPodDelete(pods[7])
	fnp.OnNamespaceDelete(namespaces[3])

	ok = checkNamespacePodIPSet(t, fnp, namespaceName[3], "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}

	// delete namespace3
	fnp.OnPodDelete(pods[4])
	fnp.OnPodDelete(pods[5])
	fnp.OnNetworkPolicyDelete(networkPolicies[8])
	fnp.OnNamespaceDelete(namespaces[2])

	ok = checkNamespacePodIPSetDelete(t, fnp, namespaceName[3])
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2", "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}
}

// this test case will update namespace label which has 3 parts:
// 1. update namespace without change, so corresponding ipset should keep as the same
// 2. update namespace label (delete label, update to new label, add label), then check corresponding ipset
// 3. update namespace back to step1, so ipset should as same as step1
func TestNamespaceUpdate(t *testing.T){

	namespaces := []*coreApi.Namespace{
		//0
		makeTestNamespace(namespaceName[2], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2"}
		}),
		//1
		makeTestNamespace(namespaceName[3], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2","ns3":"ns3"}
		}),
		//2
		makeTestNamespace(namespaceName[4], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns22","ns3":"ns3"}
		}),
		//3 #0 add label
		makeTestNamespace(namespaceName[2], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns2","ns3":"ns3"}
		}),
		//4 #1 update label
		makeTestNamespace(namespaceName[3], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns22","ns3":"ns33"}
		}),
		//5 #2 delete label
		makeTestNamespace(namespaceName[4], func(namespace *coreApi.Namespace) {
			namespace.Labels = map[string]string{"ns2":"ns22"}
		}),
	}

	pods := []*coreApi.Pod{
		// 0 namespace2
		makeTestPod(namespaceName[2], "pod2", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.1"
		}),
		// 1 namespace2
		makeTestPod(namespaceName[2], "pod3", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.2"
		}),
		// 2 namespace3
		makeTestPod(namespaceName[3], "pod4", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1","run2":"test2"}
			pod.Status.PodIP = "10.0.0.3"
		}),
		// 3 namespace3
		makeTestPod(namespaceName[3], "pod5", func(pod *coreApi.Pod) {
			pod.Labels = map[string]string{"run1":"test1","run2":"test2"}
			pod.Status.PodIP = "10.0.0.4"
		}),
		// 4 namespace4
		makeTestPod(namespaceName[4], "pod6", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.5"
		}),
		// 5 namespace4
		makeTestPod(namespaceName[4], "pod7", func(pod *coreApi.Pod) {
			pod.Status.PodIP = "10.0.0.6"
		}),
	}

	fnp := NewFakeEnnPolicy("10.244.0.0/16")

	//fnp.CleanupLeftovers()

	for i := 0; i < 3; i++{
		fnp.OnNamespaceAdd(namespaces[i])
	}
	for i := 0; i < 6; i++{
		fnp.OnPodAdd(pods[i])
	}

	//np.Spec.PodSelector: map[string]string{"run1":"test1","run2":"test2"}}
	//NamespaceSelector: map[string]string{"ns2":"ns2","ns3":"ns3"}},
	//NamespaceSelector: map[string]string{"ns2":"ns22"}},
	fnp.OnNetworkPolicyAdd(networkPolicies[8])

	ok := checkNamespacePodIPSet(t, fnp, namespaceName[3], "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2", "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}

	// update namespace without change
	fnp.OnNamespaceUpdate(namespaces[0],namespaces[0])
	fnp.OnNamespaceUpdate(namespaces[1],namespaces[1])
	fnp.OnNamespaceUpdate(namespaces[2],namespaces[2])

	ok = checkNamespacePodIPSet(t, fnp, namespaceName[3], "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2", "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}

	// update namespace: namespace2 add label, namespace3 update all labels, namespace4 delete label
	fnp.OnNamespaceUpdate(namespaces[0],namespaces[3])
	fnp.OnNamespaceUpdate(namespaces[1],namespaces[4])
	fnp.OnNamespaceUpdate(namespaces[2],namespaces[5])

	ok = checkNamespacePodIPSet(t, fnp, namespaceName[3], "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2", "10.0.0.1", "10.0.0.2")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3", "10.0.0.1", "10.0.0.2")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}

	fnp.OnNamespaceUpdate(namespaces[3],namespaces[0])
	fnp.OnNamespaceUpdate(namespaces[4],namespaces[1])
	fnp.OnNamespaceUpdate(namespaces[5],namespaces[2])

	ok = checkNamespacePodIPSet(t, fnp, namespaceName[3], "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace %s IPSet fail", namespaceName[3])
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns2", "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns2")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns3", "ns3", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns3","ns3")
	}
	ok = checkNamespaceLabelIPSet(t, fnp, "ns2", "ns22", "10.0.0.5", "10.0.0.6")
	if !ok{
		t.Errorf("check namespace label set %s=%s IPSet fail", "ns2","ns22")
	}
}

// check cleanUp, so all iptables and ipsets contains with "enn" should be deleted
func TestCleaUp(t *testing.T){

	fnp := NewFakeEnnPolicy("0.0.0.0/0")

	fnp.CleanupLeftovers()

	filterLists, _ := fnp.iptablesInterface.ListChains(FILTER_TABLE)
	for _, chainName := range filterLists{
		if strings.HasPrefix(chainName, "ENN"){
			t.Errorf("unexpected enn rule find")
			t.Errorf("%s", chainName)
		}
	}

	ipsetsName, _ := fnp.ipsetInterface.ListIPSetsName()
	for _, ipsetName := range ipsetsName {
		if strings.HasPrefix(ipsetName, "ENN") {
			t.Errorf("unexpected enn ipset find: %s", ipsetName)
		}
	}
}

func getDispatchEntry(t *testing.T, fnp *EnnPolicy, namespace, InOrE string) ([]string, bool){

	dispatchEntry := make([]string, 0)
	ok := checkIPTablesEntry(t, fnp)
	if !ok{
		t.Errorf("check iptables entry fail")
		return dispatchEntry, false
	}

	namespaceEntry, ok := checkEnnEntry(t, fnp, namespace, InOrE)
	if !ok{
		t.Errorf("check enn entry fail")
		return dispatchEntry, false
	}

	policyEntry, ok := checkNamespaceChain(t, fnp, namespaceEntry, fnp.iPRange, InOrE)
	if !ok{
		t.Errorf("check namespace entry fail")
		return dispatchEntry, false
	}

	dispatchEntry, ok = checkPolicyChain(t, fnp, policyEntry)
	if !ok{
		t.Errorf("check policy entry fail")
		return dispatchEntry, false
	}

	return dispatchEntry, true
}

func checkIPTablesEntry(t *testing.T, fnp *EnnPolicy) bool{

	var find bool
	find = false
	filterLists, _ := fnp.iptablesInterface.ListChains(FILTER_TABLE)
	for _, chainName := range filterLists{
		if strings.HasPrefix(chainName, ENN_INPUT_CHAIN){
			find = true
			break
		}
	}
	if !find{
		t.Errorf("cannot find ENN-INPUT chain in filter table")
		return false
	}

	find = false
	for _, chainName := range filterLists{
		if strings.HasPrefix(chainName, ENN_OUTPUT_CHAIN){
			find = true
			break
		}
	}
	if !find{
		t.Errorf("cannot find ENN_OUTPUT chain in filter table")
		return false
	}

	find = false
	for _, chainName := range filterLists{
		if strings.HasPrefix(chainName, ENN_FORWARD_CHAIN){
			find = true
			break
		}
	}
	if !find{
		t.Errorf("cannot find ENN_v chain in filter table")
		return false
	}

	find = false
	inputLists, _ := fnp.iptablesInterface.List(FILTER_TABLE, INPUT_CHAIN)
	for _, input := range inputLists {
		if strings.Contains(input, ENN_INPUT_CHAIN) {
			find = true
			break
		}
	}
	if !find{
		t.Errorf("cannot find ENN-INPUT chain in INPUT chain")
		return false
	}

	find = false
	outputLists, _ := fnp.iptablesInterface.List(FILTER_TABLE, OUTPUT_CHAIN)
	for _, input := range outputLists {
		if strings.Contains(input, ENN_OUTPUT_CHAIN) {
			find = true
			break
		}
	}
	if !find{
		t.Errorf("cannot find ENN-OUTPUT chain in OUTPUT chain")
		return false
	}

	find = false
	forwardLists, _ := fnp.iptablesInterface.List(FILTER_TABLE, FORWARD_CHAIN)
	for _, input := range forwardLists {
		if strings.Contains(input, ENN_FORWARD_CHAIN) {
			find = true
			break
		}
	}
	if !find{
		t.Errorf("cannot find ENN-FORWARD chain in FORWARD chain")
		return false
	}

	return true
}

func checkEnnEntry(t *testing.T, fnp *EnnPolicy, namespace string, InOrE string) (string, bool){

	namespaceEntryO, ok := checkEnnOutput(t, fnp, namespace, InOrE)
	if !ok{
		t.Errorf("Add Ingress PodSelector: check enn output fail")
		return "", false
	}

	namespaceEntryF, ok := checkEnnForward(t, fnp, namespace, InOrE)
	if !ok{
		t.Errorf("Add Ingress PodSelector: check enn forward fail")
		return "", false
	}
	if strings.Compare(namespaceEntryO, namespaceEntryF) != 0{
		t.Errorf("namespace entry for output %s, namespace entry for forward %s", namespaceEntryO, namespaceEntryF)
		return "", false
	}
	return namespaceEntryO, true
}

func checkEnnOutput(t *testing.T, fnp *EnnPolicy, namespace string, InOrE string) (string, bool){

	var namespaceEntry string
	if strings.Compare(InOrE,"ingress") == 0{
		namespaceEntry = "ENN-INGRESS"
	} else if strings.Compare(InOrE,"egress") == 0{
		namespaceEntry = "ENN-EGRESS"
	} else {
		t.Errorf("invalid InOrE")
		return "", false
	}

	var count int
	var entry string
	count = 0
	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, ENN_OUTPUT_CHAIN)
	for _, rule := range lists{
		if strings.Contains(rule, namespace) && strings.Contains(rule, namespaceEntry){

			count = count + 1
			strs := strings.Split(rule," ")
			entry = strs[len(strs)-1]
		}
	}

	if count == 0{
		t.Errorf("cannot find corresponding namespace: %s entry in ENN-OUTPUT chain", namespace)
		return "", false
	}

	if count > 1{
		t.Errorf("for a given namespace: %s, find more than 1 entry: %d in ENN-OUTPUT chain", namespace, count)
		return "", false
	}

	return entry, true
}

func checkEnnForward(t *testing.T, fnp *EnnPolicy, namespace string, InOrE string) (string, bool){

	var namespaceEntry string
	if strings.Compare(InOrE,"ingress") == 0{
		namespaceEntry = "ENN-INGRESS"
	} else if strings.Compare(InOrE,"egress") == 0{
		namespaceEntry = "ENN-EGRESS"
	} else {
		t.Errorf("invalid InOrE")
		return "", false
	}

	var count int
	var entry string
	count = 0
	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, ENN_FORWARD_CHAIN)
	for _, rule := range lists{
		if strings.Contains(rule, namespace) && strings.Contains(rule, namespaceEntry){

			count = count + 1
			strs := strings.Split(rule," ")
			entry = strs[len(strs)-1]

		}
	}

	if count == 0{
		t.Errorf("cannot find corresponding namespace: %s entry in ENN_FORWARD chain", namespace)
		return "", false
	}

	if count > 1{
		t.Errorf("for a given namespace: %s, find more than 1 entry: %d in ENN_FORWARD chain", namespace, count)
		return "", false
	}

	return entry, true

}

func checkNamespaceChain(t *testing.T, fnp *EnnPolicy, namespaceChain string, ipRange string, InOrE string) (string, bool){

	var policyEntry string
	if strings.Compare(InOrE,"ingress") == 0{
		policyEntry = "ENN-PLY-IN"
	} else if strings.Compare(InOrE,"egress") == 0{
		policyEntry = "ENN-PLY-E"
	} else {
		t.Errorf("invalid InOrE")
		return "", false
	}

	var count int
	var entry string
	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, namespaceChain)
	if strings.Compare(ipRange, "0.0.0.0/0") == 0{

		count = 0

		for _, rule := range lists {
			if strings.Contains(rule, "0.0.0.0/0") && strings.Contains(rule, policyEntry) {

				count = count + 1
				strs := strings.Split(rule," ")
				entry = strs[len(strs)-1]

			}
		}

		if count == 0{
			t.Errorf("cannot find corresponding policyChain in namespaceChain: %s", namespaceChain)
			return "", false
		}

		if count > 1{
			t.Errorf("for a given namespaceChain: %s, find more than 1 entry %d",namespaceChain, count)
		}

		return entry, true
	} else {

		count = 0

		for i := 1; i < len(lists); i++{

			if i == len(lists) - 1{
				if strings.Contains(lists[i], "ACCEPT"){
					continue
				} else {
					t.Errorf("cannot find default accept rule for namespaceChain %s", namespaceChain)
					t.Errorf("rule %s", lists[i])
					return "", false
				}
			} else {

				if strings.Contains(lists[i], "ENN-RANGEIP") && strings.Contains(lists[i], policyEntry) {

					count = count + 1
					strs := strings.Split(lists[i], " ")
					entry = strs[len(strs) - 1]

				}
			}
		}

		if count == 0{
			t.Errorf("cannot find corresponding policyChain in namespaceChain: %s", namespaceChain)
			return "", false
		}

		if count > 1{
			t.Errorf("for a given namespaceChain: %s, find more than 1 entry %d",namespaceChain, count)
		}

		return entry, true
	}
}

func checkPolicyChain(t *testing.T, fnp *EnnPolicy, policyChain string) ([]string, bool){

	result := make([]string, 0)
	lists, _ := fnp.iptablesInterface.List(FILTER_TABLE, policyChain)

	// e.g
	//iptables -t filter -S ENN-PLY-IN-I3AVDIGU5NARZT6N
	//-N ENN-PLY-IN-I3AVDIGU5NARZT6N
	//-A ENN-PLY-IN-I3AVDIGU5NARZT6N -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
	//-A ENN-PLY-IN-I3AVDIGU5NARZT6N -m comment --comment "entry for podSelector" -j ENN-DPATCH-RRBZQTV5BG7K3RAG
	//-A ENN-PLY-IN-I3AVDIGU5NARZT6N -m comment --comment "defualt reject rule" -j REJECT --reject-with icmp-port-unreachable
	for i := 1; i < len(lists); i++{
		if i == 1{
			if strings.Contains(lists[i], "conntrack") && strings.Contains(lists[i], "ACCEPT"){
				continue
			} else {
				t.Errorf("cannot find default accept rule for policy chain %s", policyChain)
				t.Errorf("rule %s", lists[i])
				return result, false
			}
		} else if i == len(lists) - 1{
			if strings.Contains(lists[i], "REJECT"){
				continue
			} else {
				t.Errorf("cannot find default reject rule for policy chain %s", policyChain)
				t.Errorf("rule %s", lists[i])
				return result, false
			}
		} else {
			if strings.Contains(lists[i], "ENN-DPATCH"){
				strs := strings.Split(lists[i]," ")
				entry := strs[len(strs)-1]
				result = append(result, entry)
			} else {
				t.Errorf("invalid rule for policy chain %s", policyChain)
				t.Errorf("rule %s", lists[i])
				return result, false
			}
		}
	}
	return result, true
}

func checkDispatchChainPodSelector(t *testing.T, dispatchChain, namespace, InOrE, specK, specV, podK, podV string) bool{

	if InOrE != "ingress" && InOrE != "egress"{
		t.Errorf("invalid InOrE %s", InOrE)
		return false
	}

	srcSet := ""
	dstSet := ""
	strs := strings.Split(dispatchChain," ")
	for i, _ := range strs{
		if strings.Compare(strs[i], "--match-set") == 0 && (i + 2) < len(strs){
			if strings.Compare(strs[i+2], "src") == 0{
				srcSet = strs[i+1]
			}
			if strings.Compare(strs[i+2], "dst") == 0{
				dstSet = strs[i+1]
			}
		}
	}
	if specK != "" && specV != ""{
		expectSet := ennLabelIPSetName(namespace, "pod", specK, specV)
		if InOrE == "ingress"{
			if strings.Compare(expectSet, dstSet) != 0{
				t.Errorf("invalid dst pod set %s, expect %s", dstSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		} else if InOrE == "egress"{
			if strings.Compare(expectSet, srcSet) != 0{
				t.Errorf("invalid src pod set %s, expect %s", srcSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		}
	}

	if podK != "" && podV != ""{
		expectSet := ennLabelIPSetName(namespace, "pod", podK, podV)
		if InOrE == "ingress"{
			if strings.Compare(expectSet, srcSet) != 0{
				t.Errorf("invalid src pod set %s, expect %s", srcSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		} else if InOrE == "egress"{
			if strings.Compare(expectSet, dstSet) != 0{
				t.Errorf("invalid dst pod set %s, expect %s", dstSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		}
	}
	return true
}

func checkDispatchChainNamespaceSelector(t *testing.T, dispatchChain, namespace, InOrE, specK, specV, namespaceK, namespaceV string) bool{

	if InOrE != "ingress" && InOrE != "egress"{
		t.Errorf("invalid InOrE %s", InOrE)
		return false
	}

	srcSet := ""
	dstSet := ""
	strs := strings.Split(dispatchChain," ")
	for i, _ := range strs{
		if strings.Compare(strs[i], "--match-set") == 0 && (i + 2) < len(strs){
			if strings.Compare(strs[i+2], "src") == 0{
				srcSet = strs[i+1]
			}
			if strings.Compare(strs[i+2], "dst") == 0{
				dstSet = strs[i+1]
			}
		}
	}
	if specK != "" && specV != ""{
		expectSet := ennLabelIPSetName(namespace, "pod", specK, specV)
		if InOrE == "ingress"{
			if strings.Compare(expectSet, dstSet) != 0{
				t.Errorf("invalid dst pod set %s, expect %s", dstSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		} else if InOrE == "egress"{
			if strings.Compare(expectSet, srcSet) != 0{
				t.Errorf("invalid src pod set %s, expect %s", srcSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		}
	}

	if namespaceK != "" && namespaceV != ""{
		expectSet := ennNSLabelIPSetName(namespaceK, namespaceV)
		if InOrE == "ingress"{
			if strings.Compare(expectSet, srcSet) != 0{
				t.Errorf("invalid src pod set %s, expect %s", srcSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		} else if InOrE == "egress"{
			if strings.Compare(expectSet, dstSet) != 0{
				t.Errorf("invalid dst pod set %s, expect %s", dstSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		}
	}
	return true
}

func checkDispatchChainIPBlock(t *testing.T, dispatchChain, namespace, InOrE, specK, specV, ipBlock string) bool{

	srcSet := ""
	dstSet := ""
	ipBlockDirect := ""
	strs := strings.Split(dispatchChain," ")
	for i, _ := range strs{
		if strings.Compare(strs[i], "--match-set") == 0 && (i + 2) < len(strs){
			if strings.Compare(strs[i+2], "src") == 0{
				srcSet = strs[i+1]
			}
			if strings.Compare(strs[i+2], "dst") == 0{
				dstSet = strs[i+1]
			}
		}
		if strings.Compare(strs[i], "-s") == 0 && (i + 1) < len(strs){
			if strings.Compare(strs[i+1], ipBlock) == 0{
				ipBlockDirect = "-s"
			}
		}
		if strings.Compare(strs[i], "-d") == 0 && (i + 1) < len(strs){
			if strings.Compare(strs[i+1], ipBlock) == 0{
				ipBlockDirect = "-d"
			}
		}
	}

	if InOrE == "ingress"{
		if strings.Compare(ipBlockDirect, "-s") != 0{
			t.Errorf("invalid ipBloclDirect %s, expect -s", ipBlockDirect)
			t.Errorf("%s", dispatchChain)
			return false
		}
		if specK != "" && specV != ""{
			expectSet := ennLabelIPSetName(namespace, "pod", specK, specV)
			if strings.Compare(expectSet, dstSet) != 0{
				t.Errorf("invalid dst pod set %s, expect %s", dstSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		}
	} else if InOrE == "egress"{
		if strings.Compare(ipBlockDirect, "-d") != 0{
			t.Errorf("invalid ipBloclDirect %s, expect -d", ipBlockDirect)
			t.Errorf("%s", dispatchChain)
			return false
		}
		if specK != "" && specV != ""{
			expectSet := ennLabelIPSetName(namespace, "pod", specK, specV)
			if strings.Compare(expectSet, srcSet) != 0{
				t.Errorf("invalid src pod set %s, expect %s", srcSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		}
	} else {
		t.Errorf("invalid InOrE %s", InOrE)
		return false
	}

	return true
}

func checkDispatchChainPort(t *testing.T, dispatchChain, namespace, InOrE, specK, specV, exprotocol, export string) bool{

	if InOrE != "ingress" && InOrE != "egress"{
		t.Errorf("invalid InOrE %s", InOrE)
		return false
	}

	srcSet := ""
	dstSet := ""
	protocol := ""
	port := ""
	strs := strings.Split(dispatchChain," ")
	for i, _ := range strs{
		if strings.Compare(strs[i], "--match-set") == 0 && (i + 2) < len(strs){
			if strings.Compare(strs[i+2], "src") == 0{
				srcSet = strs[i+1]
			}
			if strings.Compare(strs[i+2], "dst") == 0{
				dstSet = strs[i+1]
			}
		}
		if strings.Compare(strs[i], "-p") == 0 && (i + 1) < len(strs){
			protocol = strs[i+1]
		}
		if strings.Compare(strs[i], "--dport") == 0 && (i + 1) < len(strs){
			port = strs[i+1]
		}
	}
	if strings.Compare(protocol, exprotocol) != 0{
		t.Errorf("invalid protocol: %s, expect: %s", protocol, exprotocol)
		t.Errorf("%s",dispatchChain)
		return false
	}
	if strings.Compare(port, export) != 0{
		t.Errorf("invalid protocol: %s, expect: %s", port, export)
		t.Errorf("%s",dispatchChain)
		return false
	}
	if specK != "" && specV != ""{
		expectSet := ennLabelIPSetName(namespace, "pod", specK, specV)
		if InOrE == "ingress"{
			if strings.Compare(expectSet, dstSet) != 0{
				t.Errorf("invalid dst pod set %s, expect %s", dstSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		} else if InOrE == "egress"{
			if strings.Compare(expectSet, srcSet) != 0{
				t.Errorf("invalid src pod set %s, expect %s", srcSet, expectSet)
				t.Errorf("%s",dispatchChain)
				return false
			}
		}
	}
	return true
}

func checkIPRangeIPSet(t *testing.T, fnp *EnnPolicy, ipRange string, nets ...string) bool{

	ipsetName := ennIPRangeIPSetName(ipRange)
	ipset, err := fnp.ipsetInterface.GetIPSet(ipsetName)
	if err != nil{
		t.Errorf("get ipset %s error: %v", ipsetName, err)
		return false
	}
	if ipset.Name != ipsetName{
		t.Errorf("get ipset invalid ipset name: %s, expect: %s", ipset.Name, ipsetName)
		return false
	}
	if ipset.Type != utilIPSet.TypeHashNet{
		t.Errorf("get ipset invalid ipset type: %s, expect: %s", ipset.Type, utilIPSet.TypeHashNet)
		return false
	}
	entries, err := fnp.ipsetInterface.ListEntry(ipset)
	if err != nil{
		t.Errorf("list ipset %s entry error: %v", ipsetName, err)
		return false
	}
	if len(entries) != len(nets){
		t.Errorf("ipset %s invalid entry len %d, expect %d", ipsetName, len(entries), len(nets))
		return false
	}

	for _, net := range nets{
		find := false
		for _, entry := range entries{
			if entry.Net == net{
				find = true
				break
			}
		}
		if !find{
			t.Errorf("ipset %s cannot find entry %s in kernl", ipsetName, net)
			for _, entry := range entries{
				t.Errorf("kernel entry: %s", entry.Net)
			}
			return false
		}
	}

	return true
}

func checkNamespacePodIPSetDelete(t *testing.T, fnp *EnnPolicy, namespace string) bool{

	ipsetName := ennNamespaceIPSetName(namespace)
	ipsets, err := fnp.ipsetInterface.ListIPSetsName()
	if err != nil{
		t.Errorf("list ipset error: %v", err)
		return false
	}
	for _, ipset := range ipsets{
		if strings.Compare(ipset, ipsetName) == 0{
			t.Errorf("unexpected ipset name %s", ipsetName)
			return false
		}
	}

	return true
}

func checkNamespacePodIPSet(t *testing.T, fnp *EnnPolicy, namespace string, podIPs ...string) bool{

	ipsetName := ennNamespaceIPSetName(namespace)
	ipset, err := fnp.ipsetInterface.GetIPSet(ipsetName)
	if err != nil{
		t.Errorf("get ipset %s error: %v", ipsetName, err)
		return false
	}
	if ipset.Name != ipsetName{
		t.Errorf("get ipset invalid ipset name: %s, expect: %s", ipset.Name, ipsetName)
		return false
	}
	if ipset.Type != utilIPSet.TypeHashIP{
		t.Errorf("get ipset invalid ipset type: %s, expect: %s", ipset.Type, utilIPSet.TypeHashIP)
		return false
	}
	entries, err := fnp.ipsetInterface.ListEntry(ipset)
	if err != nil{
		t.Errorf("list ipset %s entry error: %v", ipsetName, err)
		return false
	}
	if len(entries) != len(podIPs){
		t.Errorf("ipset %s invalid entry len %d, expect %d", ipsetName, len(entries), len(podIPs))
		return false
	}

	for _, ip := range podIPs{
		find := false
		for _, entry := range entries{
			if entry.IP == ip{
				find = true
				break
			}
		}
		if !find{
			t.Errorf("ipset %s cannot find entry %s in kernl", ipsetName, ip)
			for _, entry := range entries{
				t.Errorf("kernel entry: %s", entry.IP)
			}
			return false
		}
	}

	return true
}

func checkPodLabelIPSet(t *testing.T, fnp *EnnPolicy, namespace, labelK, labelV string, podIPs ...string) bool{

	ipsetName := ennLabelIPSetName(namespace,"pod",labelK,labelV)
	ipset, err := fnp.ipsetInterface.GetIPSet(ipsetName)
	if err != nil{
		t.Errorf("get ipset %s error: %v", ipsetName, err)
		return false
	}
	if ipset.Name != ipsetName{
		t.Errorf("get ipset invalid ipset name: %s, expect: %s", ipset.Name, ipsetName)
		return false
	}
	if ipset.Type != utilIPSet.TypeHashIP{
		t.Errorf("get ipset invalid ipset type: %s, expect: %s", ipset.Type, utilIPSet.TypeHashIP)
		return false
	}
	entries, err := fnp.ipsetInterface.ListEntry(ipset)
	if err != nil{
		t.Errorf("list ipset %s entry error: %v", ipsetName, err)
		return false
	}
	if len(entries) != len(podIPs){
		t.Errorf("ipset %s invalid entry len %d, expect %d", ipsetName, len(entries), len(podIPs))
		return false
	}

	for _, ip := range podIPs{
		find := false
		for _, entry := range entries{
			if entry.IP == ip{
				find = true
				break
			}
		}
		if !find{
			t.Errorf("ipset %s cannot find entry %s in kernl", ipsetName, ip)
			for _, entry := range entries{
				t.Errorf("kernel entry: %s", entry.IP)
			}
			return false
		}
	}

	return true
}

func checkNamespaceLabelIPSet(t *testing.T, fnp *EnnPolicy, labelK, labelV string, podIPs ...string) bool{

	ipsetName := ennNSLabelIPSetName(labelK, labelV)
	ipset, err := fnp.ipsetInterface.GetIPSet(ipsetName)
	if err != nil{
		t.Errorf("get ipset %s error: %v", ipsetName, err)
		return false
	}
	if ipset.Name != ipsetName{
		t.Errorf("get ipset invalid ipset name: %s, expect: %s", ipset.Name, ipsetName)
		return false
	}
	if ipset.Type != utilIPSet.TypeHashIP{
		t.Errorf("get ipset invalid ipset type: %s, expect: %s", ipset.Type, utilIPSet.TypeHashIP)
		return false
	}
	entries, err := fnp.ipsetInterface.ListEntry(ipset)
	if err != nil{
		t.Errorf("list ipset %s entry error: %v", ipsetName, err)
		return false
	}
	if len(entries) != len(podIPs){
		t.Errorf("ipset %s invalid entry len %d, expect %d", ipsetName, len(entries), len(podIPs))
		return false
	}

	for _, ip := range podIPs{
		find := false
		for _, entry := range entries{
			if entry.IP == ip{
				find = true
				break
			}
		}
		if !find{
			t.Errorf("ipset %s cannot find entry %s in kernl", ipsetName, ip)
			for _, entry := range entries{
				t.Errorf("kernel entry: %s", entry.IP)
			}
			return false
		}
	}

	return true
}

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

func makeTestNamespace(name string, nsFunc func(*coreApi.Namespace)) *coreApi.Namespace{

	ns := &coreApi.Namespace{
		ObjectMeta:  metav1.ObjectMeta{
			Name: name,
		},
		Spec:  coreApi.NamespaceSpec{},
	}
	nsFunc(ns)
	return ns
}

func makeTestPod(namespace, name string, podFunc func(*coreApi.Pod)) *coreApi.Pod{

	pod := &coreApi.Pod{
		ObjectMeta:  metav1.ObjectMeta{
			Namespace: namespace,
			Name: name,
		},
		Spec:  coreApi.PodSpec{},
	}
	podFunc(pod)
	return pod
}