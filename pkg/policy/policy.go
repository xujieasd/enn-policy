package policy

import (
	policyApi "k8s.io/api/networking/v1"
	api "k8s.io/api/core/v1"
	utilexec "k8s.io/utils/exec"
	utilpolicy "enn-policy/pkg/policy/util"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/types"
	"github.com/golang/glog"
	utilIPSet "enn-policy/pkg/util/ipset"
	utiliptables "enn-policy/pkg/util/k8siptables"
	utiltool "enn-policy/pkg/util/tool"
	"enn-policy/pkg/util/iptables"
	"enn-policy/app/options"

	"sync"
	"time"
	"net"
	"sync/atomic"
	"fmt"
	"crypto/sha256"
	"encoding/base32"
	"strconv"
	"strings"
	"bytes"
	"sort"
)

const (
	FILTER_TABLE       = "filter"
	INPUT_CHAIN        = "INPUT"
	OUTPUT_CHAIN       = "OUTPUT"
	FORWARD_CHAIN      = "FORWARD"
	ENN_INPUT_CHAIN    = "ENN-INPUT"
	ENN_OUTPUT_CHAIN   = "ENN-OUTPUT"
	ENN_FORWARD_CHAIN  = "ENN-FORWARD"
)

const (
	TYPE_INGRESS       = 1
	TYPE_EGRESS        = 2
)

const (
	SYNCALL            = 0
	SYNCNETWORKPOLICY  = 1
	SYNCPOD            = 2
	SYNCNAMESPACE      = 3
)

type EnnPolicy struct {
	mu		        sync.Mutex

	client                  *kubernetes.Clientset
	syncPeriod	        time.Duration
	minSyncPeriod           time.Duration
	throttle                flowcontrol.RateLimiter

	hostName                string
	nodeIP                  net.IP
	clusterCIDR             string
	iPRange                 string

	acceptFlannel           bool
	flannelNet              string
	flannelLen              int

	initialized             int32
	networkPolicySynced     bool
	podSynced               bool
	namespaceSynced         bool
	initAllSynced           bool

	execInterface           utilexec.Interface
	ipsetInterface          utilIPSet.Interface
	iptablesInterface       iptables.Interface
	k8siptablesInterface    utiliptables.Interface

	networkPolicyChanges    utilpolicy.NetworkPolicyChangeMap
	podChanges              utilpolicy.PodChangeMap
	namespaceChanges        utilpolicy.NamespaceChangeMap

	networkPolicyMap        utilpolicy.NetworkPolicyMap
	podMatchLabelMap        utilpolicy.PodMatchLabelMap
	namespaceMatchLabelMap  utilpolicy.NamespaceMatchLabelMap
	namespacePodMap         utilpolicy.NamespacePodMap
	namespaceInfoMap        utilpolicy.NamespaceInfoMap

	// map activeIPSets stores the active ipsets created by syncPolicyRules which key is ipset name
	activeIPSets            map[string]*utilIPSet.IPSet
	// map podXLabelMap represent the label information of spec.podSelector of each networkPolicy
	podXLabelMap            map[types.NamespacedName]*utilpolicy.NamespacedLabelMap
	// map podXLabelSet represent ths ipset for spec.podSelector of each networkPolicy
	podXLabelSet            map[types.NamespacedName]*utilIPSet.IPSet
	// map podLabelSet represent the ipset for podSelector
	podLabelSet             map[utilpolicy.NamespacedLabel]*utilIPSet.IPSet
	// map namespacePodLabelSet represent the ipset for namespaceSelector
	namespacePodLabelSet    map[utilpolicy.Label]*utilIPSet.IPSet
	// map namespacePodSet represent the ipset for namespace
	namespacePodSet         map[string]*utilIPSet.IPSet

	existingFilterChains    map[utiliptables.Chain]string
	activeFilterChains      map[utiliptables.Chain]bool

	// The following buffers are used to reuse memory and avoid allocations
	// that are significantly impacting performance.
	iptablesData            *bytes.Buffer
	filterChains            *bytes.Buffer
	filterRules             *bytes.Buffer
}

func NewEnnPolicy(
    clientset               *kubernetes.Clientset,
    config                  *options.EnnPolicyConfig,
    hostName                string,
    nodeIP                  net.IP,
    execInterface           utilexec.Interface,
    ipsetInterface          utilIPSet.Interface,
    iptablesInterface       iptables.Interface,
    k8siptablesInterface    utiliptables.Interface,
)(*EnnPolicy, error){

	syncPeriod      := config.PolicyPeriod
	minSyncPeriod   := config.MinSyncPeriod
	iPRange         := config.IPRange
	acceptFlannel   := config.AcceptFlannelIP
	flannelNet      := config.FlannelNetwork
	flannelLen, err := strconv.Atoi(config.FlannelLenBit)
	if err != nil{
		glog.Errorf("invalid flannelLen %s, err: %v, set this value to default number 8", config.FlannelLenBit, err)
		flannelLen = 8
	}

	glog.V(4).Infof("start to build ennPolicy structure")
	// check valid user input
	if minSyncPeriod > syncPeriod {
		return nil, fmt.Errorf("min-sync (%v) must be < sync(%v)", minSyncPeriod, syncPeriod)
	}

	clusterCIDR, err := utilpolicy.GetPodCidrFromNodeSpec(clientset,config.HostnameOverride)
	if err != nil{
		glog.Errorf("NewEnnPolicy failure: GetPodCidr fall: %s", err.Error())
		clusterCIDR = ""
	}
	if len(clusterCIDR) == 0 {
		glog.Warningf("clusterCIDR not specified, unable to distinguish between internal and external traffic")
	}

	var throttle flowcontrol.RateLimiter
	if minSyncPeriod != 0{
		qps := float32(time.Second) / float32(minSyncPeriod)
		burst := 2
		glog.V(3).Infof("minSyncPeriod: %v, syncPeriod: %v, burstSyncs: %d", minSyncPeriod, syncPeriod, burst)
		throttle = flowcontrol.NewTokenBucketRateLimiter(qps,burst)
	}

	ennpolicy := EnnPolicy{
		client:                  clientset,
		hostName:                hostName,
		nodeIP:                  nodeIP,
		clusterCIDR:             clusterCIDR,
		iPRange:                 iPRange,
		acceptFlannel:           acceptFlannel,
		flannelNet:              flannelNet,
		flannelLen:              flannelLen,
		networkPolicySynced:     false,
		podSynced:               false,
		namespaceSynced:         false,
		initAllSynced:           true,
		execInterface:           execInterface,
		ipsetInterface:          ipsetInterface,
		iptablesInterface:       iptablesInterface,
		k8siptablesInterface:    k8siptablesInterface,
		throttle:                throttle,
		syncPeriod:              syncPeriod,
		minSyncPeriod:           minSyncPeriod,
		networkPolicyChanges:    utilpolicy.NewNetworkPolicyChangeMap(),
		podChanges:              utilpolicy.NewPodLabelChangeMap(),
		namespaceChanges:        utilpolicy.NewNamespaceChangeMap(),
		networkPolicyMap:        make(utilpolicy.NetworkPolicyMap),
		podMatchLabelMap:        make(utilpolicy.PodMatchLabelMap),
		namespaceMatchLabelMap:  make(utilpolicy.NamespaceMatchLabelMap),
		namespacePodMap:         make(utilpolicy.NamespacePodMap),
		namespaceInfoMap:        make(utilpolicy.NamespaceInfoMap),
		activeIPSets:            make(map[string]*utilIPSet.IPSet),
		podXLabelMap:            make(map[types.NamespacedName]*utilpolicy.NamespacedLabelMap),
		podXLabelSet:            make(map[types.NamespacedName]*utilIPSet.IPSet),
		podLabelSet:             make(map[utilpolicy.NamespacedLabel]*utilIPSet.IPSet),
		namespacePodLabelSet:    make(map[utilpolicy.Label]*utilIPSet.IPSet),
		namespacePodSet:         make(map[string]*utilIPSet.IPSet),
		existingFilterChains:    make(map[utiliptables.Chain]string),
		activeFilterChains :     make(map[utiliptables.Chain]bool),
		iptablesData:            bytes.NewBuffer(nil),
		filterChains:            bytes.NewBuffer(nil),
		filterRules:             bytes.NewBuffer(nil),
	}

	return &ennpolicy, nil
}

func FakePolicy(
    execInterface           utilexec.Interface,
    ipsetInterface          utilIPSet.Interface,
    iptablesInterface       iptables.Interface,
    k8siptablesInterface    utiliptables.Interface,
)(*EnnPolicy, error){
	ennpolicy := EnnPolicy{
		execInterface:           execInterface,
		ipsetInterface:          ipsetInterface,
		iptablesInterface:       iptablesInterface,
		k8siptablesInterface:    k8siptablesInterface,
	}
	return &ennpolicy, nil
}

func (policy *EnnPolicy) isInitialized() bool {
	return atomic.LoadInt32(&policy.initialized) > 0
}

func (policy *EnnPolicy) setInitialized(value bool) {
	var initialized int32
	if value {
		initialized = 1
	}
	atomic.StoreInt32(&policy.initialized, initialized)
}

func (policy *EnnPolicy) OnNetworkPolicyAdd(networkPolicy *policyApi.NetworkPolicy){
	glog.V(6).Infof("OnNetworkPolicyAdd policy name: %s, namespace: %s", networkPolicy.Name, networkPolicy.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: networkPolicy.Namespace, Name: networkPolicy.Name}
	if policy.networkPolicyChanges.Update(&namespaceName, nil, networkPolicy) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCNETWORKPOLICY)
	}
}

func (policy *EnnPolicy) OnNetworkPolicyUpdate(oldNetworkPolicy, networkPolicy *policyApi.NetworkPolicy){
	glog.V(6).Infof("OnNetworkPolicyUpdate old policy name: %s, namespace: %s; new policy name: %s, namespace: %s",
		oldNetworkPolicy.Name, oldNetworkPolicy.Namespace, networkPolicy.Name, networkPolicy.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: networkPolicy.Namespace, Name: networkPolicy.Name}
	if policy.networkPolicyChanges.Update(&namespaceName, oldNetworkPolicy, networkPolicy) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCNETWORKPOLICY)
	}
}

func (policy *EnnPolicy) OnNetworkPolicyDelete(networkPolicy *policyApi.NetworkPolicy){
	glog.V(6).Infof("OnNetworkPolicyDelete policy name: %s, namespace: %s", networkPolicy.Name, networkPolicy.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: networkPolicy.Namespace, Name: networkPolicy.Name}
	if policy.networkPolicyChanges.Update(&namespaceName, networkPolicy, nil) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCNETWORKPOLICY)
	}
}

func (policy *EnnPolicy) OnNetworkPolicySynced(){
	glog.V(6).Infof("OnNetworkPolicySynced")
	policy.mu.Lock()
	policy.networkPolicySynced = true
	policy.setInitialized(policy.networkPolicySynced && policy.podSynced && policy.namespaceSynced)
	policy.mu.Unlock()

	policy.syncEnnPolicy(SYNCNETWORKPOLICY)
}

func (policy *EnnPolicy) OnPodAdd(pod *api.Pod){
	glog.V(6).Infof("OnPodAdd pod name: %s, namespace: %s", pod.Name, pod.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	if policy.podChanges.Update(&namespaceName, nil, pod) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCPOD)
	}
}

func (policy *EnnPolicy) OnPodUpdate(oldPod, pod *api.Pod){
	glog.V(6).Infof("OnPodUpdate old pod name: %s, namespace: %s; new pod name: %s, namespace: %s",
		oldPod.Name, oldPod.Namespace, pod.Name, pod.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	if policy.podChanges.Update(&namespaceName, oldPod, pod) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCPOD)
	}
}

func (policy *EnnPolicy) OnPodDelete(pod *api.Pod){
	glog.V(6).Infof("OnPodDelete pod name: %s, namespace: %s", pod.Name, pod.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	if policy.podChanges.Update(&namespaceName, pod, nil) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCPOD)
	}
}

func (policy *EnnPolicy) OnPodSynced(){
	glog.V(6).Infof("OnPodSynced")
	policy.mu.Lock()
	policy.podSynced = true
	policy.setInitialized(policy.networkPolicySynced && policy.podSynced && policy.namespaceSynced)
	policy.mu.Unlock()

	policy.syncEnnPolicy(SYNCPOD)
}

func (policy *EnnPolicy) OnNamespaceAdd(namespace *api.Namespace){
	glog.V(6).Infof("OnNamespaceAdd namespace name: %s, namespace: %s", namespace.Name, namespace.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: namespace.Namespace, Name: namespace.Name}
	if policy.namespaceChanges.Update(&namespaceName, nil, namespace) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCNAMESPACE)
	}
}

func (policy *EnnPolicy) OnNamespaceUpdate(oldNamespace, namespace *api.Namespace){
	glog.V(6).Infof("OnNamespaceUpdate old namespace name: %s, namespace: %s; new namespace name: %s, namespace: %s",
		oldNamespace.Name, oldNamespace.Namespace, namespace.Name, namespace.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: namespace.Namespace, Name: namespace.Name}
	if policy.namespaceChanges.Update(&namespaceName, oldNamespace, namespace) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCNAMESPACE)
	}
}

func (policy *EnnPolicy) OnNamespaceDelete(namespace *api.Namespace){
	glog.V(6).Infof("OnNamespaceDelete namespace name: %s, namespace: %s", namespace.Name, namespace.Namespace)
	glog.V(6).Infof("policy initialized %v", policy.isInitialized())
	namespaceName := types.NamespacedName{Namespace: namespace.Namespace, Name: namespace.Name}
	if policy.namespaceChanges.Update(&namespaceName, namespace, nil) && policy.isInitialized() {
		policy.syncEnnPolicy(SYNCNAMESPACE)
	}
}

func (policy *EnnPolicy) OnNamespaceSynced(){
	glog.V(6).Infof("OnNamespaceSynced")
	policy.mu.Lock()
	policy.namespaceSynced = true
	policy.setInitialized(policy.networkPolicySynced && policy.podSynced && policy.namespaceSynced)
	policy.mu.Unlock()

	policy.syncEnnPolicy(SYNCNAMESPACE)
}

func (policy *EnnPolicy) SyncLoop(stopCh <-chan struct{}, wg *sync.WaitGroup){

	glog.V(2).Infof("enn policy start run loop")
	t := time.NewTicker(policy.syncPeriod)
	defer t.Stop()
	defer wg.Done()

	for{
		select {
		case <-t.C:
			glog.V(4).Infof("Periodic sync")
			policy.Sync()
		case <-stopCh:
			glog.V(4).Infof("stop sync")
			return
		}
	}
}

func (policy *EnnPolicy) Sync(){

	policy.syncEnnPolicy(SYNCALL)
}

// suncEnnPolicy will create and sync iptables rules and ipsets for networkPolicy
// suncEnnPolicy is called when enn-policy watch networkPolicy/namespace/pod events called
// or is called every syncPeriod(default value is 15m)
// input parameter syncType could be SYNCALL/SYNCNETWORKPOLICY/SYNCNAMESPACE/SYNCPOD
// syncEnnPolicy will handle different rules for different kind of syncType,
func (policy *EnnPolicy) syncEnnPolicy(syncType int){

	glog.V(4).Infof("enn policy start sync, syncType is %d", syncType)
	policy.mu.Lock()
	defer policy.mu.Unlock()

	if policy.throttle != nil{
		policy.throttle.Accept()
	}

	start := time.Now()
	defer func() {
		glog.V(4).Infof("syncEnnPolicyRules took %v", time.Since(start))
	}()

	// init activeIPSets
	// policy.activeIPSets = make(map[string]*utilIPSet.IPSet)

	// ensure whether enn-policy entries are established
	err := policy.ensureEnnEntry()
	if err!= nil{
		glog.Errorf("cannot ensure enn filter entry %v", err)
		return
	}

	// ensure ipset for flannel net
	flannelNetSet, err := policy.ensureFlannelNetSet()
	if err!= nil{
		glog.Errorf("ensure ipset for flannel net %s error %v", policy.iPRange, err)
		return
	}

	// ensure ipset for iPRange
	iPRangeSet, err := policy.ensureIPRangeSet()
	if err!= nil{
		glog.Errorf("ensure ipset for ip range %s error %v", policy.iPRange, err)
		return
	}

	// don't sync rules till we've received networkPolicy & pods & namespace
	if !policy.networkPolicySynced || !policy.podSynced || !policy.namespaceSynced {
		glog.V(2).Info("Not syncing ipvs rules until networkPolicy & pods & namespace have been received from master")
		return
	}

	// if nothing changes we need to sync the whole iptables/ipset rules
	// if only pod or namespace changes,
	// since policy is not changed, iptables rule do not need to be changed, only need to sync ipset

	glog.V(4).Infof("start to sync ennPolicy")

	if policy.initAllSynced {
		glog.V(2).Infof("first time to do sync ennPolicy, so neet to update all maps and sync all rules")
		syncType = SYNCALL
		policy.initAllSynced = false
		policy.podChanges.Lock.Lock()
		policy.namespaceChanges.Lock.Lock()
		utilpolicy.UpdateNetworkPolicyMap(policy.networkPolicyMap, &policy.networkPolicyChanges)
		utilpolicy.UpdatePodMatchLabelMap(policy.podMatchLabelMap, &policy.podChanges)
		utilpolicy.UpdateNamespacePodMap(policy.namespacePodMap, &policy.podChanges)
		utilpolicy.UpdateNamespaceInfoMap(policy.namespaceInfoMap, &policy.namespaceChanges)
		utilpolicy.UpdateNamespaceMatchLabelMap(policy.namespaceMatchLabelMap, policy.namespacePodMap, &policy.namespaceChanges)
		policy.podChanges.CleanUpItem()
		policy.namespaceChanges.CleanUpItem()
		policy.podChanges.Lock.Unlock()
		policy.namespaceChanges.Lock.Unlock()
	}

	// todo: delete unused iptables, delete unused ipsets(check whether label is deleted)
	switch syncType {
	case SYNCALL:
		glog.V(4).Infof("syncType is SYNCALL, so sync all rules and check unused rule")
		// no map changed so we need to sync the whole iptables rules and ipset rules
		err := policy.initActiveIPSets(iPRangeSet, flannelNetSet)
		if err != nil{
			glog.Errorf("init active IPSets failed %v", err)
		}
		err = policy.ensureIPRangeSetMember(iPRangeSet)
		if err != nil{
			glog.Errorf("ensure ip range member err %v", err)
		}
		err = policy.ensureFlannelNetSetMember(flannelNetSet)
		if err != nil{
			glog.Errorf("ensure flannel net member err %v", err)
		}
		err = policy.syncPolicyRules()
		if err != nil{
			glog.Errorf("sync policy rule failed %v", err)
		}
		err = policy.syncAllPodSets()
		if err != nil{
			glog.Errorf("sync pod ipset failed %v", err)
		}
		err = policy.checkUnusedIPSets()
		if err != nil{
			glog.Errorf("check unused ipsets failed %v", err)
		}
	case SYNCNETWORKPOLICY:
		glog.V(4).Infof("syncType is SYNCNETWORKPOLICY, so sync all rules")
		// if networkPolicy map update, we need to sync the whole iptables rules
		// since syncAllPodSets will sync ipset created by syncPolicyRules
		// so we also neet to sync ipset rules
		// todo: better use incremental update
		utilpolicy.UpdateNetworkPolicyMap(policy.networkPolicyMap, &policy.networkPolicyChanges)
		err := policy.initActiveIPSets(iPRangeSet, flannelNetSet)
		if err != nil{
			glog.Errorf("init active IPSets failed %v", err)
		}
		err = policy.syncPolicyRules()
		if err != nil{
			glog.Errorf("sync policy rule failed %v", err)
		}
		err = policy.syncAllPodSets()
		if err != nil{
			glog.Errorf("sync pod ipset failed %v", err)
		}
	case SYNCPOD:
		glog.V(4).Infof("syncType is SYNCPOD, so sync all pod label sets, namspace sets and namespace label sets")
		policy.podChanges.Lock.Lock()
		utilpolicy.UpdatePodMatchLabelMap(policy.podMatchLabelMap, &policy.podChanges)
		utilpolicy.UpdateNamespacePodMap(policy.namespacePodMap, &policy.podChanges)
		utilpolicy.UpdateNamespaceMatchLabelMapByPod(policy.namespaceMatchLabelMap, policy.namespaceInfoMap, &policy.podChanges)
		err := policy.syncPodSets()
		if err != nil{
			glog.Errorf("sync pod ipset failed %v", err)
		}
		policy.podChanges.CleanUpItem()
		policy.podChanges.Lock.Unlock()

	case SYNCNAMESPACE:
		glog.V(4).Infof("syncType is SYNCNAMESPACE, so sync all namespace label sets")
		policy.namespaceChanges.Lock.Lock()
		utilpolicy.UpdateNamespaceInfoMap(policy.namespaceInfoMap, &policy.namespaceChanges)
		utilpolicy.UpdateNamespaceMatchLabelMap(policy.namespaceMatchLabelMap, policy.namespacePodMap, &policy.namespaceChanges)
		err := policy.syncNamespaceSets()
		if err != nil{
			glog.Errorf("sync pod ipset failed %v", err)
		}
		policy.namespaceChanges.CleanUpItem()
		policy.namespaceChanges.Lock.Unlock()
	}

}

// insert ennPolicy entry in filter tables, e.g
// -N ENN-INPUT
// -N ENN-OUTPUT
// -N ENN-FORWARD
// -A INPUT -j ENN-INPUT
// -A OUTPUT -j ENN-OUTPUT
// -A FORWARD -j ENN-FORWARD
func (policy *EnnPolicy) ensureEnnEntry() error{
	glog.V(4).Infof("start to ensure EnnEntry")
	err := policy.iptablesInterface.NewChain(FILTER_TABLE, ENN_INPUT_CHAIN)
	if err!= nil{
		return err
	}
	err = policy.iptablesInterface.NewChain(FILTER_TABLE, ENN_OUTPUT_CHAIN)
	if err!= nil{
		return err
	}
	err = policy.iptablesInterface.NewChain(FILTER_TABLE, ENN_FORWARD_CHAIN)
	if err!= nil{
		return err
	}

	var args []string

	args = []string{
		"-j", ENN_INPUT_CHAIN,
	}
	err = policy.iptablesInterface.PrependUnique(FILTER_TABLE, INPUT_CHAIN, args...)
	if err!= nil{
		return err
	}

	args = []string{
		"-j", ENN_OUTPUT_CHAIN,
	}
	err = policy.iptablesInterface.PrependUnique(FILTER_TABLE, OUTPUT_CHAIN, args...)
	if err!= nil{
		return err
	}

	args = []string{
		"-j", ENN_FORWARD_CHAIN,
	}
	err = policy.iptablesInterface.PrependUnique(FILTER_TABLE, FORWARD_CHAIN, args...)
	if err!= nil{
		return err
	}

	return nil
}


// insert ipset for ip range e.g:
// Name: ENN-FLANNEL-xxxxxx
// Type: hash:ip
// Revision: 2
// Header: family inet hashsize 1024 maxelem 65536
// Size in memory: 448
// References: 1
// Members:
// 10.244.0.0
// 10.244.0.1
func (policy *EnnPolicy) ensureFlannelNetSet() (*utilIPSet.IPSet, error){
	glog.V(4).Infof("start to ensure flannel net ip set")
	flannelNetName := ennFlannelIPSetName(policy.flannelNet, strconv.Itoa(policy.flannelLen))
	flannelNetSet := &utilIPSet.IPSet{
		Name:    flannelNetName,
		Type:    utilIPSet.TypeHashIP,
	}
	err := policy.ipsetInterface.CreateIPSet(flannelNetSet, true)
	if err!= nil{
		return nil, fmt.Errorf("ensure flannelNetSet error %v", err)
	}
	return flannelNetSet, nil
}

func (policy *EnnPolicy) ensureFlannelNetSetMember(flannelNetSet *utilIPSet.IPSet) error{

	if !policy.acceptFlannel{
		glog.V(4).Infof("accept flannel is set to false, so skip ensure flannelNetSet members")
		return nil
	}

	if policy.flannelLen > 32 || policy.flannelLen < 0 {
		return fmt.Errorf("invalid flannelLen:%d", policy.flannelLen)
	}

	str := strings.Split(policy.flannelNet, "/")
	if len(str) != 2{
		return fmt.Errorf("invalid flannelNet:%s", policy.flannelNet)
	}

	// get flannel net ip
	flannelIP   := str[0]
	ips := strings.Split(flannelIP, ".")
	if len(ips) != 4{
		return fmt.Errorf("invalid flannelNet:%s", policy.flannelNet)
	}
	flannelIpsInt, err := utiltool.IpStringToInt(ips)
	if err != nil{
		return fmt.Errorf("invalid flannelNet:%s", policy.flannelNet)
	}

	// get flannel net sublen
	flannelMask, err := strconv.Atoi(str[1])
	if err != nil{
		return fmt.Errorf("invalid flannelNet:%s", policy.flannelNet)
	}
	//flanelSubLen := 32 - flannelMask
	dockerMask   := flannelMask + policy.flannelLen
	if dockerMask > 32 || dockerMask < 0 {
		return fmt.Errorf("invalid dockerMask, flannelNet:%s, flannelLen:%d", policy.flannelNet, policy.flannelLen)
	}
	dockerSubLen := 32 - dockerMask

	// build flannel/docker ip map
	// add all possible flannel ips and docker ips
	// flannel ip end up with .0 and docker ip end up with .1
	var item int
	for item = 0; item * 8 <= dockerSubLen; item ++{}
	item --
	if item == 0{
		return fmt.Errorf("dockerSubLen is less than 8: %d", dockerSubLen)
	}
	dockerBit := dockerSubLen - item * 8
	step      := utiltool.PowerInt(2, dockerBit)
	stepLimit := utiltool.PowerInt(2, policy.flannelLen)


	flanelIPMap := make(map[string]bool)

	for i := 0; i < stepLimit; i++{
		// add flannel ip (end up with .0)
		ipString, err := utiltool.IpIntToString(flannelIpsInt)
		if err != nil{
			fmt.Errorf("ensureFlannelNetSetMember: invalid ip err: %v", err)
		} else {
			flanelIPMap[ipString] = true
		}
		// add docker ip (end up with .1)
		dockerIpsInt, err := utiltool.IpOperateAdd(flannelIpsInt, 0, 1)
		if err != nil{
			fmt.Errorf("ensureFlannelNetSetMember: ip add step error: %v", err)
		} else {
			ipString, err := utiltool.IpIntToString(dockerIpsInt)
			if err != nil{
				fmt.Errorf("ensureFlannelNetSetMember: invalid ip err: %v", err)
			} else {
				flanelIPMap[ipString] = true
			}
		}
		// get next flannel ip
		flannelIpsInt, err = utiltool.IpOperateAdd(flannelIpsInt, item, step)
		if err != nil{
			fmt.Errorf("ensureFlannelNetSetMember: ip add step error: %v", err)
			break
		}
	}

	kernelSet, err := policy.ipsetInterface.GetIPSet(flannelNetSet.Name)
	if err!= nil{
		return err
	}
	kernelEntries, err := policy.ipsetInterface.ListEntry(kernelSet)
	if err!= nil{
		return err
	}

	//delete unused entries
	for _, kernelEntry := range kernelEntries{
		glog.V(7).Infof("kernel entry is type:%s, ip:%s, port:%s, net:%s",
			kernelEntry.Type, kernelEntry.IP, kernelEntry.Port, kernelEntry.Net)
		ip, err := utilIPSet.EntryToString(kernelEntry)
		if err!= nil{
			glog.Errorf("get entry err ipset:%s, entry type:%s err:%v", kernelSet.Name, kernelEntry.Type, err)
			continue
		}
		_, ok := flanelIPMap[ip]
		if !ok{
			glog.V(6).Infof("find unused entry %s of ipset %s:%s so delete it", ip, kernelSet.Name, kernelSet.Type)
			err := policy.ipsetInterface.DelEntry(kernelSet, kernelEntry, false)
			if err != nil{
				glog.Errorf("ensureFlannelNetSetMember error : %v", err)
				continue
			}
		}
	}
	// add new entries
	kernelEntryIPMap := make(map[string]bool)
	for _, kernelEntry := range kernelEntries{
		kernelEntryIPMap[kernelEntry.IP] = true
	}
	for ip := range flanelIPMap{
		glog.V(7).Infof("podInfoMap ip is %s", ip)
		_, ok := kernelEntryIPMap[ip]
		if !ok{
			glog.V(6).Infof("find new entry %s of ipset %s:%s so add it", ip, kernelSet.Name, kernelSet.Type)
			entry := &utilIPSet.Entry{
				IP:    ip,
				Type:  utilIPSet.TypeHashIP,
			}
			err := policy.ipsetInterface.AddEntry(kernelSet, entry, true)
			if err != nil{
				glog.Errorf("ensureFlannelNetSetMember error : %v", err)
				continue
			}
		}
	}


	return nil
}

// insert ipset for ip range e.g:
// Name: ENN-RANGEIP-xxxxxx
// Type: hash:net
// Revision: 6
// Header: family inet hashsize 1024 maxelem 65536
// Size in memory: 448
// References: 1
// Members:
// 10.244.0.0/16
func (policy *EnnPolicy) ensureIPRangeSet() (*utilIPSet.IPSet, error){
	glog.V(4).Infof("start to ensure IPRangeSet")
	iPRangeName := ennIPRangeIPSetName(policy.iPRange)
	iPRangeSet := &utilIPSet.IPSet{
		Name:    iPRangeName,
		Type:    utilIPSet.TypeHashNet,
	}
	err := policy.ipsetInterface.CreateIPSet(iPRangeSet, true)
	if err!= nil{
		return nil, fmt.Errorf("ensure IPRangeSet error %v", err)
	}
	return iPRangeSet, nil
}

func (policy *EnnPolicy) ensureIPRangeSetMember(iPRangeSet *utilIPSet.IPSet) error{

	entries, err := policy.ipsetInterface.ListEntry(iPRangeSet)
	if err != nil{
		return fmt.Errorf("ensure IPRangeSetMember error %v", err)
	}
	// delete unused entry for IPRangeSet
	for _, entry := range entries{
		net, err := utilIPSet.EntryToString(entry)
		if err != nil{
			return fmt.Errorf("invalid IPRangeSet %v", err)
		}
		if strings.Compare(net, policy.iPRange) != 0{
			glog.V(6).Infof("find unused entry: %s of ipset %s:%s so delete it", net, iPRangeSet.Name, iPRangeSet.Type)
			err := policy.ipsetInterface.DelEntry(iPRangeSet, entry, false)
			if err != nil{
				return fmt.Errorf("ensure IPRangeSetMember error %v", err)
			}
		}
	}

	// add iPRange to IPRangeSet

	if strings.Compare(policy.iPRange, "0.0.0.0/0") == 0{
		glog.V(4).Infof("default ip range is 0.0.0.0/0, so skip add entry (ipset cannot handle 0.0.0.0/0)")
		return nil
	}

	iPRangeEntry := &utilIPSet.Entry{
		Type:    utilIPSet.TypeHashNet,
		Net:     policy.iPRange,
	}
	err = policy.ipsetInterface.AddEntry(iPRangeSet, iPRangeEntry, true)
	if err!= nil{
		return fmt.Errorf("ensure IPRangeSet  add entry error %v", err)
	}
	return nil
}

// initActiveIPSets will init policy.activeIPSets
// and add first ipset "ipRangeSet" into this map
// this map will store ipsets which created by enn-policy
// enn-policy will only sync ipsets which is "active"
func (policy *EnnPolicy) initActiveIPSets(iPRangeSet, flannelNetSet *utilIPSet.IPSet) error{
	// init activeIPSets
	policy.activeIPSets = make(map[string]*utilIPSet.IPSet)
	if iPRangeSet == nil{
		return fmt.Errorf("ipRangeSet is nil")
	}
	if flannelNetSet == nil{
		return fmt.Errorf("flannelNetSet is nil")
	}
	policy.activeIPSets[iPRangeSet.Name]    = iPRangeSet
	policy.activeIPSets[flannelNetSet.Name] = flannelNetSet
	return nil
}

// syncPolicyRules will scan the whole networkPolicyMap and create iptables filter rule
// s for each networkPolicyInfo
// syncPolicyReles will also create ipset for namespace/podSelector/namespaceSelector/ipRange for each networkPolicyInfo if required
// these ipsets will be stored in map and entries will be added in syncPodSets function
func (policy *EnnPolicy) syncPolicyRules() error{

	glog.V(4).Infof("start to sync PolicyRules")
	// should first cleanup iptables created by enn-policy, then insert new policy rules
	// todo: better use iptables-save and iptables-restore
	//err := policy.cleanupPolicy()
	//if err != nil{
	//	return fmt.Errorf("syncPolicyRules err %v", err)
	//}
	var args []string
	policy.existingFilterChains = make(map[utiliptables.Chain]string)

	policy.iptablesData.Reset()
	err := policy.k8siptablesInterface.SaveInto(utiliptables.TableFilter, policy.iptablesData)
	if err != nil { // if we failed to get any rules
		glog.Errorf("Failed to execute iptables-save, syncing all rules: %v", err)
	} else { // otherwise parse the output
		policy.existingFilterChains = utiliptables.GetChainLines(utiliptables.TableFilter, policy.iptablesData.Bytes())
	}

	// Reset all buffers used later.
	// This is to avoid memory reallocations and thus improve performance.
	policy.filterChains.Reset()
	policy.filterRules.Reset()

	// Write table headers.
	writeLine(policy.filterChains, "*filter")

	// Make sure we keep stats for the top-level chains, if they existed
	// (which most should have because we created them above).
	if chain, ok := policy.existingFilterChains[utiliptables.Chain(ENN_INPUT_CHAIN)]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(utiliptables.Chain(ENN_INPUT_CHAIN)))
	}
	if chain, ok := policy.existingFilterChains[utiliptables.Chain(ENN_OUTPUT_CHAIN)]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(utiliptables.Chain(ENN_OUTPUT_CHAIN)))
	}
	if chain, ok := policy.existingFilterChains[utiliptables.Chain(ENN_FORWARD_CHAIN)]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(utiliptables.Chain(ENN_FORWARD_CHAIN)))
	}

	// Accumulate NAT chains to keep.
	policy.activeFilterChains = make(map[utiliptables.Chain]bool) // use a map as a set

	for _, networkPolicy := range policy.networkPolicyMap {
		policyName := networkPolicy.Name
		policyNamespace := networkPolicy.Namespace

		// create ipset for corresponding namespace (NetworkPolicy.metadata.namespace)
		glog.V(4).Infof("networkpolicy %s is defined in namespace %s, so create ipset for this namespace", networkPolicy.Name, networkPolicy.Namespace)
		namespacePodSetName := ennNamespaceIPSetName(policyNamespace)
		namespaceIPSet := &utilIPSet.IPSet{
			Name:    namespacePodSetName,
			Type:    utilIPSet.TypeHashIP,
		}
		err := policy.ipsetInterface.CreateIPSet(namespaceIPSet, true)
		if err != nil{
			glog.Errorf("networkPolicy %s create ipset for namespace %s err %v", policyName, policyNamespace, err)
			continue
		}
		// add IPSet into map if this IPSet is not created
		_, ok := policy.namespacePodSet[policyNamespace]
		if !ok{
			policy.namespacePodSet[policyNamespace] = namespaceIPSet
		}
		policy.activeIPSets[namespaceIPSet.Name] = namespaceIPSet

		// create ipset for corresponding policy podSelector (NetworkPolicy.spec.podSelector)
		// var policyPodSetNames []string
		var xLabel []string
		specPodSelector := &utilpolicy.NamespacedLabelMap{
			Namespace:  policyNamespace,
			Label:      make(map[string]string),
		}
		for labelKey, labelValue := range networkPolicy.PodSelector{
			glog.V(4).Infof("networkpolicy %s defined spec.podSelector %s=%s, so create ipset for the label", networkPolicy.Name, labelKey, labelValue)
			policyPodSetName := ennLabelIPSetName(policyNamespace, "pod", labelKey, labelValue)
			policyIPSet := &utilIPSet.IPSet{
				Name:    policyPodSetName,
				Type:    utilIPSet.TypeHashIP,
			}
			err := policy.ipsetInterface.CreateIPSet(policyIPSet, true)
			if err != nil{
				glog.Errorf("networkPolicy %s:%s create ipset for policy pod match %s=%s err %v",
					policyName,
					policyNamespace,
					labelKey,
					labelValue,
					err,
				)
				continue
			}
			// policyPodSetNames = append(policyPodSetNames, policyPodSetName)
			namespacedLabel := utilpolicy.NamespacedLabel{
				Namespace:   policyNamespace,
				LabelKey:    labelKey,
				LabelValue:  labelValue,
			}
			// add IPSet into map if this IPSet is not created
			_, ok := policy.podLabelSet[namespacedLabel]
			if !ok{
				policy.podLabelSet[namespacedLabel] = policyIPSet
			}
			policy.activeIPSets[policyIPSet.Name] = policyIPSet

			xLabel = append(xLabel, labelKey)
			xLabel = append(xLabel, labelValue)
			specPodSelector.Label[labelKey] = labelValue
		}

		// create podSelector ipset for policy spec, logic of matchLabels should be and
		if len(networkPolicy.PodSelector) > 0 {
			policyPodSetName := ennXLabelIPSetName(policyNamespace, "pod", xLabel)
			policyIPSet := &utilIPSet.IPSet{
				Name:    policyPodSetName,
				Type:    utilIPSet.TypeHashIP,
			}
			err := policy.ipsetInterface.CreateIPSet(policyIPSet, true)
			if err != nil{
				glog.Errorf("networkPolicy %s:%s create ipset for policy spec podSelector",
					policyName,
					policyNamespace,
					err,
				)
			}
			policy.activeIPSets[policyIPSet.Name] = policyIPSet

			namespaceName := types.NamespacedName{Namespace: policyNamespace, Name: policyName}

			policy.podXLabelMap[namespaceName] = specPodSelector

			policy.podXLabelSet[namespaceName] = policyIPSet

			policy.activeIPSets[policyIPSet.Name] = policyIPSet
		}

		// create iptables rules in filter table
		// todo: should better use iptables-save/restore
		for _, policyType := range networkPolicy.PolicyType {

			if policyType == utilpolicy.TypeIngress{
				policy.syncIngressRule(networkPolicy, namespacePodSetName)


			} else if policyType == utilpolicy.TypeEgress{
				policy.syncEgressRule(networkPolicy, namespacePodSetName)

			}
		}
	}

	for chain := range policy.activeFilterChains {
		chainString := string(chain)
		if strings.HasPrefix(chainString, "ENN-PLY-"){

			// accept all the traffic when ctstate is RELATED,ESTABLISHED
			// iptables -t filter -A ENN-PLY-x-xxxxxx -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
			// need to ensure this sure should always be the first rule of chain ENN-PLY-x-xxxxx
			args = []string{
				"-I", chainString,
				"-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED",
				"-j", "ACCEPT",
			}
			writeLine(policy.filterRules, args...)

			// reject other traffic by default
			// iptables -t filter -A ENN-PLY-x-xxxxxx -j REJECT
			// need to ensure this sure should always be the last rule of chain ENN-PLY-x-xxxxx
			args = []string{
				"-A", chainString,
				"-m", "comment", "--comment", `"defualt reject rule"`,
				"-j", "REJECT",
			}
			writeLine(policy.filterRules, args...)
		}
	}

	// Delete chains no longer in use.
	for chain := range policy.existingFilterChains {
		if !policy.activeFilterChains[chain] {
			chainString := string(chain)
			if !strings.HasPrefix(chainString, "ENN-INGRESS-") &&
				!strings.HasPrefix(chainString, "ENN-EGRESS-") &&
				!strings.HasPrefix(chainString, "ENN-PLY-IN-") &&
				!strings.HasPrefix(chainString, "ENN-PLY-E-") &&
				!strings.HasPrefix(chainString, "ENN-DPATCH-")&&
				!strings.HasPrefix(chainString, "ENN-IPCIDR-"){
				// Ignore chains that aren't ours.
				continue
			}
			// We must (as per iptables) write a chain-line for it, which has
			// the nice effect of flushing the chain.  Then we can remove the
			// chain.
			writeLine(policy.filterChains, policy.existingFilterChains[chain])
			writeLine(policy.filterRules, "-X", chainString)
		}
	}

	// Write the end-of-table markers.
	writeLine(policy.filterRules, "COMMIT")

	// Sync rules.
	// NOTE: NoFlushTables is used so we don't flush non-enn-policy chains in the table
	policy.iptablesData.Reset()
	policy.iptablesData.Write(policy.filterChains.Bytes())
	policy.iptablesData.Write(policy.filterRules.Bytes())

	glog.V(5).Infof("Restoring iptables rules: %s", policy.iptablesData.Bytes())
	//fmt.Printf("Restoring iptables rules: %s", policy.iptablesData.Bytes())
	err = policy.k8siptablesInterface.RestoreAll(policy.iptablesData.Bytes(), utiliptables.NoFlushTables, utiliptables.RestoreCounters)
	if err != nil {
		glog.Errorf("Failed to execute iptables-restore: %v", err)

		return err
	}

	return nil
}


func (policy *EnnPolicy) syncIngressRule(networkPolicy *utilpolicy.NetworkPolicyInfo, namespacePodSetName string) {

	glog.V(4).Infof("process ingress sync rules for networkPolicy %s/%s ", networkPolicy.Namespace, networkPolicy.Name)
	var args []string

	// create iptables ingress policy rule for different namespace, like
	// iptables -t filter -N ENN-INGRESS-xxxxxx
	// iptables -t filter -A ENN-FORWARD -m set --match-set [namespaceIPSet] dst -j ENN-INGRESS-xxxxxx
	// iptables -t filter -A ENN-OUTPUT -m set --match-set [namespaceIPSet] dst -j ENN-INGRESS-xxxxxx
	namespaceIngressChainName := ennNamespaceIngressChainName(networkPolicy.Namespace)

	chainName := utiliptables.Chain(namespaceIngressChainName)
	if chain, ok := policy.existingFilterChains[chainName]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
	}

	_, ok := policy.activeFilterChains[chainName]
	if ok{
		glog.V(4).Infof("chainName:%s is already active so skip add rule to OUTPUT and FORWARD chain", string(chainName))
	}else {

		policy.activeFilterChains[chainName] = true
		comment := fmt.Sprintf(`"ingress entry for namespace/%s"`, networkPolicy.Namespace)
		args = []string{
			"-A", string(ENN_FORWARD_CHAIN),
			"-m", "set", "--match-set", namespacePodSetName, "dst",
			"-m", "comment", "--comment", comment,
		}
		writeLine(policy.filterRules, append(args, "-j", namespaceIngressChainName)...)
		args = []string{
			"-A", string(ENN_OUTPUT_CHAIN),
			"-m", "set", "--match-set", namespacePodSetName, "dst",
			"-m", "comment", "--comment", comment,
		}
		writeLine(policy.filterRules, append(args, "-j", namespaceIngressChainName)...)
	}

	// create iptables ingress policy rule for iPRange, like
	// iptables -t filter -N ENN-PLY-IN-xxxxxx
	// iptables -t filter -A ENN-INGRESS-xxxxxx -m set --match-set [flannelRnage] src -j ACCEPT
	// iptables -t filter -A ENN-INGRESS-xxxxxx -m set --match-set [iPRange] src -j ENN-PLY-IN-xxxxxx
	// iptables -t filter -A ENN-INGRESS-xxxxxx -j ACCEPT

	glog.V(4).Infof("process iptables ingress policy rule for iPRange %s/ ", policy.iPRange)
	iPRangeIngressChainName := ennIPRangeIngressChainName(networkPolicy.Namespace, policy.iPRange)
	chainName = utiliptables.Chain(iPRangeIngressChainName)
	if chain, ok := policy.existingFilterChains[chainName]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
	}

	_, ok = policy.activeFilterChains[chainName]
	if ok{
		glog.V(4).Infof("chainName:%s is already active so skip add rule to chain:%s", string(chainName),namespaceIngressChainName)
	} else {

		policy.activeFilterChains[chainName] = true
		if policy.acceptFlannel{
			glog.V(4).Infof("process iptables ingress policy rule for all flannel/docker ips: %s", policy.flannelNet)
			flannelNetName := ennFlannelIPSetName(policy.flannelNet, strconv.Itoa(policy.flannelLen))
			comment := fmt.Sprintf(`"match flannel ip net: %s"`, policy.flannelNet)
			args = []string{
				"-A", namespaceIngressChainName,
				"-m", "set", "--match-set", flannelNetName, "src",
				"-m", "comment", "--comment", comment,
				"-j", "ACCEPT",
			}
			writeLine(policy.filterRules, args...)
		}
		if strings.Compare(policy.iPRange, "0.0.0.0/0") == 0 {
			glog.V(4).Infof("iprange is default value 0.0.0.0/0 so derectly jump to iPRangeChain")
			comment := fmt.Sprintf(`"iprange is default value %s so derectly jump to iPRangeChain"`, policy.iPRange)
			args = []string{
				"-A", namespaceIngressChainName,
				"-m", "comment", "--comment", comment,
			}
			writeLine(policy.filterRules, append(args, "-j", iPRangeIngressChainName)...)
		} else {
			comment := fmt.Sprintf(`"match ip range %s"`, policy.iPRange)
			iPRangeName := ennIPRangeIPSetName(policy.iPRange)
			args = []string{
				"-A", namespaceIngressChainName,
				"-m", "set", "--match-set", iPRangeName, "src",
				"-m", "comment", "--comment", comment,
			}
			writeLine(policy.filterRules, append(args, "-j", iPRangeIngressChainName)...)
			args = []string{
				"-A", namespaceIngressChainName,
				"-m", "comment", "--comment", `"accept other traffic beyond ip range"`,
				"-j", "ACCEPT",
			}
			writeLine(policy.filterRules, args...)
		}
	}

	// through all ingress rules to handle ingress rules in iptables filter table e.g
	// iptables -t filter -A ENN-PLY-IN-xxxxxx -j ENN-DPATCH-Axxxx
	// iptables -t filter -A ENN-PLY-IN-xxxxxx -j ENN-DPATCH-Bxxxx
	// iptables -t filter -A ENN-PLY-IN-xxxxxx -j ENN-DPATCH-Cxxxx
	for _, ingress := range networkPolicy.Ingress{

		// no PodSelector and NamespaceSelector and CIDR defined
		// which means ingress.spec is empty
		// but if only ports defined, should apply rule for all ip with special ports
		if len(ingress.PodSelector) == 0 && len(ingress.NamespaceSelector) == 0 && len(ingress.IPBlock) == 0 {

			if len(ingress.Ports) > 0 {
				policy.dispatchOnlyPorts(
					TYPE_INGRESS,
					networkPolicy,
					ingress.Ports,
					iPRangeIngressChainName,
				)
			}
		}
		// handle iptables rules for podSelect
		for _, podSelect := range ingress.PodSelector{
			policy.dispatchPodSelector(
				TYPE_INGRESS,
				networkPolicy,
				podSelect,
				ingress.Ports,
				iPRangeIngressChainName,
			)
		}
		// handle iptables rules for namespaceSelect
		for _, namespaceSelect := range ingress.NamespaceSelector{
			policy.dispatchNamespaceSelector(
				TYPE_INGRESS,
				networkPolicy,
				namespaceSelect,
				ingress.Ports,
				iPRangeIngressChainName,
			)
		}
		// handle iptables rules for ipBlock
		for _, ipBlock := range ingress.IPBlock{
			policy.dispatchIPBlock(
				TYPE_INGRESS,
				networkPolicy,
				ipBlock,
				ingress.Ports,
				iPRangeIngressChainName,
			)
		}

	}
}

func (policy *EnnPolicy) syncEgressRule(networkPolicy *utilpolicy.NetworkPolicyInfo, namespacePodSetName string) {

	glog.V(4).Infof("process egress sync rules for networkPolicy %s/%s ", networkPolicy.Namespace, networkPolicy.Name)
	var args []string

	// create iptables egress policy rule for different namespace, like
	// iptables -t filter -N ENN-EGRESS-xxxxxx
	// iptables -t filter -A ENN-FORWARD -m set --match-set [namespaceIPSet] src -j ENN-EGRESS-xxxxxx
	// iptables -t filter -A ENN-OUTPUT -m set --match-set [namespaceIPSet] src -j ENN-EGRESS-xxxxxx
	namespaceEgressChainName := ennNamespaceEgressChainName(networkPolicy.Namespace)
	chainName := utiliptables.Chain(namespaceEgressChainName)
	if chain, ok := policy.existingFilterChains[chainName]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
	}

	_, ok := policy.activeFilterChains[chainName]
	if ok{
		glog.V(4).Infof("chainName:%s is already active so skip add rule to OUTPUT and FORWARD chain", string(chainName))
	}else {
		policy.activeFilterChains[chainName] = true
		comment := fmt.Sprintf(`"ingress entry for namespace/%s"`, networkPolicy.Namespace)
		args = []string{
			"-A", string(ENN_FORWARD_CHAIN),
			"-m", "set", "--match-set", namespacePodSetName, "src",
			"-m", "comment", "--comment", comment,
		}
		writeLine(policy.filterRules, append(args, "-j", namespaceEgressChainName)...)
		args = []string{
			"-A", string(ENN_OUTPUT_CHAIN),
			"-m", "set", "--match-set", namespacePodSetName, "src",
			"-m", "comment", "--comment", comment,
		}
		writeLine(policy.filterRules, append(args, "-j", namespaceEgressChainName)...)
	}

	// create iptables egress policy rule for iPRange, like
	// iptables -t filter -N ENN-PLY-E-xxxxxx
	// iptables -t filter -A ENN-INGRESS-xxxxxx -m set --match-set [flannelRnage] dst -j ACCEPT
	// iptables -t filter -A ENN-EGRESS-xxxxxx -m set --match-set [iPRange] dst -j ENN-PLY-E-xxxxxx
	// iptables -t filter -A ENN-EGRESS-xxxxxx -j ACCEPT

	glog.V(4).Infof("process iptables egress policy rule for iPRange %s/ ", policy.iPRange)
	iPRangeEgressChainName := ennIPRangeEgressChainName(networkPolicy.Namespace, policy.iPRange)
	chainName = utiliptables.Chain(iPRangeEgressChainName)
	if chain, ok := policy.existingFilterChains[chainName]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
	}

	_, ok = policy.activeFilterChains[chainName]
	if ok{
		glog.V(4).Infof("chainName:%s is already active so skip add rule to chain:%s", string(chainName),namespaceEgressChainName)
	} else {

		policy.activeFilterChains[chainName] = true
		if policy.acceptFlannel{
			glog.V(4).Infof("process iptables egress policy rule for all flannel/docker ips: %s", policy.flannelNet)
			flannelNetName := ennFlannelIPSetName(policy.flannelNet, strconv.Itoa(policy.flannelLen))
			comment := fmt.Sprintf(`"match flannel ip net: %s"`, policy.flannelNet)
			args = []string{
				"-A", namespaceEgressChainName,
				"-m", "set", "--match-set", flannelNetName, "dst",
				"-m", "comment", "--comment", comment,
				"-j", "ACCEPT",
			}
			writeLine(policy.filterRules, args...)
		}
		if strings.Compare(policy.iPRange, "0.0.0.0/0") == 0 {
			glog.V(4).Infof("iprange is default value 0.0.0.0/0 so derectly jump to iPRangeChain")
			comment := fmt.Sprintf(`"iprange is default value %s so derectly jump to iPRangeChain"`, policy.iPRange)
			args = []string{
				"-A", namespaceEgressChainName,
				"-m", "comment", "--comment", comment,
			}
			writeLine(policy.filterRules, append(args, "-j", iPRangeEgressChainName)...)
		} else {
			comment := fmt.Sprintf(`"match ip range %s"`, policy.iPRange)
			iPRangeName := ennIPRangeIPSetName(policy.iPRange)
			args = []string{
				"-A", namespaceEgressChainName,
				"-m", "set", "--match-set", iPRangeName, "dst",
				"-m", "comment", "--comment", comment,
			}
			writeLine(policy.filterRules, append(args, "-j", iPRangeEgressChainName)...)
			args = []string{
				"-A", namespaceEgressChainName,
				"-m", "comment", "--comment", `"accept other traffic beyond ip range"`,
				"-j", "ACCEPT",
			}
			writeLine(policy.filterRules, args...)
		}
	}

	// through all ingress rules to handle ingress rules in iptables filter table e.g
	// iptables -t filter -A ENN-PLY-E-xxxxxx -j ENN-DPATCH-Axxxx
	// iptables -t filter -A ENN-PLY-E-xxxxxx -j ENN-DPATCH-Bxxxx
	// iptables -t filter -A ENN-PLY-E-xxxxxx -j ENN-DPATCH-Cxxxx
	for _, egress := range networkPolicy.Egress{

		// no PodSelector and NamespaceSelector and CIDR defined
		// which means egress.spec is empty
		// but if only ports defined, should apply rule for all ip with special ports
		if len(egress.PodSelector) == 0 && len(egress.NamespaceSelector) == 0 && len(egress.IPBlock) == 0 {

			if len(egress.Ports) > 0 {
				policy.dispatchOnlyPorts(
					TYPE_EGRESS,
					networkPolicy,
					egress.Ports,
					iPRangeEgressChainName,
				)
			}
		}
		// handle iptables rules for podSelect
		for _, podSelect := range egress.PodSelector{
			policy.dispatchPodSelector(
				TYPE_EGRESS,
				networkPolicy,
				podSelect,
				egress.Ports,
				iPRangeEgressChainName,
			)
		}
		// handle iptables rules for namespaceSelect
		for _, namespaceSelect := range egress.NamespaceSelector{
			policy.dispatchNamespaceSelector(
				TYPE_EGRESS,
				networkPolicy,
				namespaceSelect,
				egress.Ports,
				iPRangeEgressChainName,
			)
		}
		// handle iptables rules for ipBlock
		for _, ipBlock := range egress.IPBlock{
			policy.dispatchIPBlock(
				TYPE_EGRESS,
				networkPolicy,
				ipBlock,
				egress.Ports,
				iPRangeEgressChainName,
			)
		}

	}

}

// dispatchOnlyPorts create iptables for ingress/egress rules only contains ports
// accept traffic with special ports
func (policy *EnnPolicy) dispatchOnlyPorts( ruleType         int,
                                            networkPolicy    *utilpolicy.NetworkPolicyInfo,
                                            Ports            []utilpolicy.PolicyPort,
                                            iPRangeChainName string,
) {

	glog.V(4).Infof("process sync OnlyPorts policy rules for networkPolicy %s/%s ", networkPolicy.Namespace, networkPolicy.Name)
	var args []string

	var policyPodMatchDirect string
	if ruleType == TYPE_INGRESS{
		policyPodMatchDirect = "dst"
		glog.V(4).Infof("OnlyPorts ingress policy rules")
	} else if ruleType == TYPE_EGRESS{
		policyPodMatchDirect = "src"
		glog.V(4).Infof("OnlyPorts egress policy rules")
	} else {
		glog.Errorf("invalid ruleType %d", ruleType)
		return
	}

	dispatchChainName := ennDispatchChainName(
		networkPolicy.Namespace,
		networkPolicy.Name,
		strconv.Itoa(ruleType),
		"onlyPorts",
		"",
		"",
		"",
	)

	chainName := utiliptables.Chain(dispatchChainName)
	if chain, ok := policy.existingFilterChains[chainName]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
	}
	policy.activeFilterChains[chainName] = true

	comment := fmt.Sprintf(`"policy %s:%s entry for only ports"`,
		networkPolicy.Namespace,
		networkPolicy.Name,
	)
	args = []string{
		"-A", iPRangeChainName,
		"-m", "comment", "--comment", comment,
	}
	writeLine(policy.filterRules, append(args, "-j", dispatchChainName)...)


	for _, port := range Ports{
		// spec.podSelector is not defined, so create rule match all pods
		if len(networkPolicy.PodSelector) == 0{
			glog.V(4).Infof("network policy %s/%s spec.podSelector is not defined so create rule match all pod %s",
				networkPolicy.Namespace, networkPolicy.Name, policyPodMatchDirect)
			comment := fmt.Sprintf(`"accept rule selected by policy %s/%s: %s match all pods"`,
				networkPolicy.Namespace,
				networkPolicy.Name,
				policyPodMatchDirect,
			)

			args = []string{
				"-A", dispatchChainName,
				"-m", "comment", "--comment", comment,
				"-p", port.Protocol,
				"--dport", port.Port,
				"-j", "ACCEPT",
			}
			writeLine(policy.filterRules, args...)

		} else {
			// if spec.podSelector is defined, so create rule match pods src with labels
			var xLabel []string
			for labelKey, labelValue := range networkPolicy.PodSelector {
				glog.V(4).Infof("accept rule selected by network policy %s/%s: %s match %s=%s",
					networkPolicy.Namespace, networkPolicy.Name, policyPodMatchDirect, labelKey, labelValue)

				xLabel = append(xLabel, labelKey)
				xLabel = append(xLabel, labelValue)

			}

			comment := fmt.Sprintf(`"accept rule selected by policy %s/%s: %s match policy spec podSelector"`,
				networkPolicy.Namespace,
				networkPolicy.Name,
				policyPodMatchDirect,
			)

			policyPodSetName := ennXLabelIPSetName(networkPolicy.Namespace, "pod", xLabel)

			args = []string{
				"-A", dispatchChainName,
				"-m", "comment", "--comment", comment,
				"-m", "set", "--match-set", policyPodSetName, policyPodMatchDirect,
				"-p", port.Protocol,
				"--dport", port.Port,
				"-j", "ACCEPT",
			}
			writeLine(policy.filterRules, args...)
		}
	}
}

// dispatchPodSelector create iptables for ingress/egress rules based on podSelector
// for ingress rule, iptables will accept traffic from corresponding selected pod set
// for egress rule, iptables will accept traffic to corresponding selected pod set
func (policy *EnnPolicy) dispatchPodSelector(ruleType         int,
                                             networkPolicy    *utilpolicy.NetworkPolicyInfo,
                                             podSelect        utilpolicy.LabelSelector,
                                             Ports            []utilpolicy.PolicyPort,
                                             iPRangeChainName string,
) {

	glog.V(4).Infof("process sync PodSelector policy rules for networkPolicy %s/%s ", networkPolicy.Namespace, networkPolicy.Name)
	var args []string

	var policyPodMatchDirect string
	var selectPodMatchDirect string
	if ruleType == TYPE_INGRESS{
		policyPodMatchDirect = "dst"
		selectPodMatchDirect = "src"
		glog.V(4).Infof("PodSelector ingress policy rules")
	} else if ruleType == TYPE_EGRESS{
		policyPodMatchDirect = "src"
		selectPodMatchDirect = "dst"
		glog.V(4).Infof("PodSelector egress policy rules")
	} else {
		glog.Errorf("invalid ruleType %d", ruleType)
		return
	}

	for podLabelKey, podLabelValue := range podSelect.Label{

		glog.V(4).Infof("network policy %s/%s NetworkPolicyPeer PodSelector %s=%s",
			networkPolicy.Namespace, networkPolicy.Name, podLabelKey, podLabelValue)
		// create corresponding ipset for podSelect
		selectPodSetName := ennLabelIPSetName(networkPolicy.Namespace, "pod", podLabelKey, podLabelValue)
		selectPodIPSet := &utilIPSet.IPSet{
			Name:    selectPodSetName,
			Type:    utilIPSet.TypeHashIP,
		}
		err := policy.ipsetInterface.CreateIPSet(selectPodIPSet, true)
		if err != nil{
			glog.Errorf("networkPolicy %s:%s create ipset for podSelector pod match %s=%s err %v",
				networkPolicy.Namespace,
				networkPolicy.Name,
				podLabelKey,
				podLabelValue,
				err,
			)
			continue
		}
		namespacedLabel := utilpolicy.NamespacedLabel{
			Namespace:   networkPolicy.Namespace,
			LabelKey:    podLabelKey,
			LabelValue:  podLabelValue,
		}
		// add IPSet into map if this IPSet is not created
		_, ok := policy.podLabelSet[namespacedLabel]
		if !ok{
			policy.podLabelSet[namespacedLabel] = selectPodIPSet
		}
		policy.activeIPSets[selectPodIPSet.Name] = selectPodIPSet

		// create dispatch iptables rule
		dispatchChainName := ennDispatchChainName(
			networkPolicy.Namespace,
			networkPolicy.Name,
			strconv.Itoa(ruleType),
			"podSelector",
			"",
			podLabelKey,
			podLabelValue,
		)

		chainName := utiliptables.Chain(dispatchChainName)
		if chain, ok := policy.existingFilterChains[chainName]; ok {
			writeLine(policy.filterChains, chain)
		} else {
			writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
		}
		policy.activeFilterChains[chainName] = true

		comment := fmt.Sprintf(`"policy %s:%s entry for podSelector"`,
			networkPolicy.Namespace,
			networkPolicy.Name,
		)
		args = []string{
			"-A", iPRangeChainName,
			"-m", "comment", "--comment", comment,
		}
		writeLine(policy.filterRules, append(args, "-j", dispatchChainName)...)

		// spec.podSelector is not defined, so create rule match all dst pods for ingress, match all src pods for egress
		if len(networkPolicy.PodSelector) == 0{
			glog.V(4).Infof("network policy %s/%s spec.podSelector is not defined so create rule match all pod %s",
				networkPolicy.Namespace, networkPolicy.Name, policyPodMatchDirect)
			comment := fmt.Sprintf(`"accept rule selected by policy %s/%s: %s pod match %s=%s"`,
				networkPolicy.Namespace,
				networkPolicy.Name,
				selectPodMatchDirect,
				podLabelKey,
				podLabelValue,
			)
			// Ports is not defined
			if len(Ports) == 0{

				args = []string{
					"-A", dispatchChainName,
					"-m", "comment", "--comment", comment,
					"-m", "set", "--match-set", selectPodSetName, selectPodMatchDirect,
					"-j", "ACCEPT",
				}
				writeLine(policy.filterRules, args...)

			}
			for _, port := range Ports{

				args = []string{
					"-A", dispatchChainName,
					"-m", "comment", "--comment", comment,
					"-m", "set", "--match-set", selectPodSetName, selectPodMatchDirect,
					"-p", port.Protocol,
					"--dport", port.Port,
					"-j", "ACCEPT",
				}
				writeLine(policy.filterRules, args...)

			}
		} else {
			// spec.podSelector is defined, so create rule match selected dst pods for ingress, match selected src pods for egress
			var xLabel []string
			for policyLabelKey, policyLabelValue := range networkPolicy.PodSelector {
				glog.V(4).Infof("accept rule selected by network policy %s/%s: %s match %s=%s",
					networkPolicy.Namespace, networkPolicy.Name, policyPodMatchDirect, policyLabelKey, policyLabelValue)

				xLabel = append(xLabel, policyLabelKey)
				xLabel = append(xLabel, policyLabelValue)

			}

			comment := fmt.Sprintf(`"accept rule selected by policy %s/%s: %s match spec podSelector, %s pod match %s=%s"`,
				networkPolicy.Namespace,
				networkPolicy.Name,
				policyPodMatchDirect,
				selectPodMatchDirect,
				podLabelKey,
				podLabelValue,
			)

			policyPodSetName := ennXLabelIPSetName(networkPolicy.Namespace, "pod", xLabel)

			// Ports is not defined
			if len(Ports) == 0 {

				args = []string{
					"-A", dispatchChainName,
					"-m", "comment", "--comment", comment,
					"-m", "set", "--match-set", policyPodSetName, policyPodMatchDirect,
					"-m", "set", "--match-set", selectPodSetName, selectPodMatchDirect,
					"-j", "ACCEPT",
				}
				writeLine(policy.filterRules, args...)

			}
			for _, port := range Ports {

				args = []string{
					"-A", dispatchChainName,
					"-m", "comment", "--comment", comment,
					"-m", "set", "--match-set", policyPodSetName, policyPodMatchDirect,
					"-m", "set", "--match-set", selectPodSetName, selectPodMatchDirect,
					"-p", port.Protocol,
					"--dport", port.Port,
					"-j", "ACCEPT",
				}
				writeLine(policy.filterRules, args...)

			}
		}
	}
}


// dispatchNamespaceSelector create iptables for ingress/egress rules based on namespaceSelector
// for ingress rule, iptables will accept traffic from corresponding selected namespace set
// for egress rule, iptables will accept traffic to corresponding selected namespace set
func (policy *EnnPolicy) dispatchNamespaceSelector(ruleType         int,
                                                   networkPolicy    *utilpolicy.NetworkPolicyInfo,
                                                   namespaceSelect  utilpolicy.LabelSelector,
                                                   Ports            []utilpolicy.PolicyPort,
                                                   iPRangeChainName string,
) {
	glog.V(4).Infof("process sync NamespaceSelector policy rules for networkPolicy %s/%s ", networkPolicy.Namespace, networkPolicy.Name)
	var args []string

	var policyPodMatchDirect string
	var selectPodMatchDirect string
	if ruleType == TYPE_INGRESS{
		policyPodMatchDirect = "dst"
		selectPodMatchDirect = "src"
		glog.V(4).Infof("NamespaceSelector ingress policy rules")
	} else if ruleType == TYPE_EGRESS{
		policyPodMatchDirect = "src"
		selectPodMatchDirect = "dst"
		glog.V(4).Infof("NamespaceSelector egress policy rules")
	} else {
		glog.Errorf("invalid ruleType %d", ruleType)
		return
	}

	for namespaceLabelKey, namespaceLabelValue := range namespaceSelect.Label{
		glog.V(4).Infof("network policy %s/%s NetworkPolicyPeer NamespaceSelector %s=%s",
			networkPolicy.Namespace, networkPolicy.Name, namespaceLabelKey, namespaceLabelValue)
		// create corresponding ipset for namespaceSelect
		selectPodSetName := ennNSLabelIPSetName(namespaceLabelKey, namespaceLabelValue)
		selectPodIPSet := &utilIPSet.IPSet{
			Name:    selectPodSetName,
			Type:    utilIPSet.TypeHashIP,
		}
		err := policy.ipsetInterface.CreateIPSet(selectPodIPSet, true)
		if err != nil{
			glog.Errorf("networkPolicy %s:%s create ipset for namespaceSelector match %s=%s err %v",
				networkPolicy.Namespace,
				networkPolicy.Name,
				namespaceLabelKey,
				namespaceLabelValue,
				err,
			)
			continue
		}
		label := utilpolicy.Label{
			LabelKey:    namespaceLabelKey,
			LabelValue:  namespaceLabelValue,
		}
		// add IPSet into namespacePodSet map if this IPSet is not created
		_, ok := policy.namespacePodLabelSet[label]
		if !ok{
			policy.namespacePodLabelSet[label] = selectPodIPSet
		}

		policy.activeIPSets[selectPodIPSet.Name] = selectPodIPSet

		// create dispatch iptables rule
		dispatchChainName := ennDispatchChainName(
			networkPolicy.Namespace,
			networkPolicy.Name,
			strconv.Itoa(ruleType),
			"namespaceSelector",
			"",
			namespaceLabelKey,
			namespaceLabelValue,
		)

		chainName := utiliptables.Chain(dispatchChainName)
		if chain, ok := policy.existingFilterChains[chainName]; ok {
			writeLine(policy.filterChains, chain)
		} else {
			writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
		}
		policy.activeFilterChains[chainName] = true

		comment := fmt.Sprintf(`"policy %s:%s entry for namespaceSelector"`,
			networkPolicy.Namespace,
			networkPolicy.Name,
		)
		args = []string{
			"-A", iPRangeChainName,
			"-m", "comment", "--comment", comment,
		}
		writeLine(policy.filterRules, append(args, "-j", dispatchChainName)...)

		// spec.podSelector is not defined, so create rule match all dst pods for ingress, match all src pods for egress
		if len(networkPolicy.PodSelector) == 0{
			glog.V(4).Infof("network policy %s/%s spec.podSelector is not defined so create rule match all pod %s",
				networkPolicy.Namespace, networkPolicy.Name, policyPodMatchDirect)
			comment := fmt.Sprintf(`"accept rule selected by policy %s/%s: %s namespace match %s=%s"`,
				networkPolicy.Namespace,
				networkPolicy.Name,
				selectPodMatchDirect,
				namespaceLabelKey,
				namespaceLabelValue,
			)
			// Ports is not defined
			if len(Ports) == 0{

				args = []string{
					"-A", dispatchChainName,
					"-m", "comment", "--comment", comment,
					"-m", "set", "--match-set", selectPodSetName, selectPodMatchDirect,
					"-j", "ACCEPT",
				}
				writeLine(policy.filterRules, args...)

			}
			for _, port := range Ports{

				args = []string{
					"-A", dispatchChainName,
					"-m", "comment", "--comment", comment,
					"-m", "set", "--match-set", selectPodSetName, selectPodMatchDirect,
					"-p", port.Protocol,
					"--dport", port.Port,
					"-j", "ACCEPT",
				}
				writeLine(policy.filterRules, args...)

			}
		} else {
			// spec.podSelector is defined, so create rule match selected dst pods for ingress, match selected src pods for egress
			var xLabel []string
			for policyLabelKey, policyLabelValue := range networkPolicy.PodSelector {
				glog.V(4).Infof("accept rule selected by network policy %s/%s: %s match %s=%s",
					networkPolicy.Namespace, networkPolicy.Name, policyPodMatchDirect, policyLabelKey, policyLabelValue)

				xLabel = append(xLabel, policyLabelKey)
				xLabel = append(xLabel, policyLabelValue)

			}

			comment := fmt.Sprintf(`"accept rule selected by policy %s/%s: %s match policy spec podSelector, %s namespace match %s=%s"`,
				networkPolicy.Namespace,
				networkPolicy.Name,
				policyPodMatchDirect,
				selectPodMatchDirect,
				namespaceLabelKey,
				namespaceLabelValue,
			)

			policyPodSetName := ennXLabelIPSetName(networkPolicy.Namespace, "pod", xLabel)

			// Ports is not defined
			if len(Ports) == 0 {

				args = []string{
					"-A", dispatchChainName,
					"-m", "comment", "--comment", comment,
					"-m", "set", "--match-set", policyPodSetName, policyPodMatchDirect,
					"-m", "set", "--match-set", selectPodSetName, selectPodMatchDirect,
					"-j", "ACCEPT",
				}
				writeLine(policy.filterRules, args...)

			}
			for _, port := range Ports {

				args = []string{
					"-A", dispatchChainName,
					"-m", "comment", "--comment", comment,
					"-m", "set", "--match-set", policyPodSetName, policyPodMatchDirect,
					"-m", "set", "--match-set", selectPodSetName, selectPodMatchDirect,
					"-p", port.Protocol,
					"--dport", port.Port,
					"-j", "ACCEPT",
				}
				writeLine(policy.filterRules, args...)

			}
		}
	}
}

// dispatchIPBlock create iptables for ingress/egress rules based on ipBlock
// for ingress rule, iptables will accept traffic from corresponding selected cidr
// for egress rule, iptables will accept traffic to corresponding selected cidr
func (policy *EnnPolicy) dispatchIPBlock(ruleType         int,
                                         networkPolicy    *utilpolicy.NetworkPolicyInfo,
                                         ipBlock          utilpolicy.CIDRRange,
                                         Ports            []utilpolicy.PolicyPort,
                                         iPRangeChainName string,
) {
	glog.V(4).Infof("process sync IPBlock policy rules for networkPolicy %s/%s ", networkPolicy.Namespace, networkPolicy.Name)
	var args []string

	var policyPodMatchDirect string
	var cidrDirect string
	if ruleType == TYPE_INGRESS{
		policyPodMatchDirect = "dst"
		cidrDirect = "-s"
		glog.V(4).Infof("IPBlock ingress policy rules")
	} else if ruleType == TYPE_EGRESS{
		policyPodMatchDirect = "src"
		cidrDirect = "-d"
		glog.V(4).Infof("IPBlock egress policy rules")
	} else {
		glog.Errorf("invalid ruleType %d", ruleType)
		return
	}

	glog.V(4).Infof("network olicy %s/%s NetworkPolicyPeer IPBlock CIDR %s",
		networkPolicy.Namespace, networkPolicy.Name, ipBlock.CIDR)
	// create dispatch iptables rule
	dispatchChainName := ennDispatchChainName(
		networkPolicy.Namespace,
		networkPolicy.Name,
		strconv.Itoa(ruleType),
		"ipBlock",
		ipBlock.CIDR,
		"",
		"",
	)

	ipBlockChainName := ennIPBlockChainName(
		networkPolicy.Namespace,
		networkPolicy.Name,
		strconv.Itoa(ruleType),
		ipBlock.CIDR,
	)

	chainName := utiliptables.Chain(dispatchChainName)
	if chain, ok := policy.existingFilterChains[chainName]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
	}
	policy.activeFilterChains[chainName] = true


	chainName = utiliptables.Chain(ipBlockChainName)
	if chain, ok := policy.existingFilterChains[chainName]; ok {
		writeLine(policy.filterChains, chain)
	} else {
		writeLine(policy.filterChains, utiliptables.MakeChainLine(chainName))
	}
	policy.activeFilterChains[chainName] = true

	comment := fmt.Sprintf(`"policy %s:%s entry for ipBlock"`,
		networkPolicy.Namespace,
		networkPolicy.Name,
	)
	args = []string{
		"-A", iPRangeChainName,
		"-m", "comment", "--comment", comment,
	}
	writeLine(policy.filterRules, append(args, "-j", dispatchChainName)...)

	// spec.podSelector is not defined, so create rule match all dst pods for ingress, match all src pods for egress
	if len(networkPolicy.PodSelector) == 0{
		glog.V(4).Infof("network policy %s/%s spec.podSelector is not defined so create rule match all pod %s",
			networkPolicy.Namespace, networkPolicy.Name, policyPodMatchDirect)
		comment := fmt.Sprintf(`"accept rule selected by policy %s/%s: %s cidr %s"`,
			networkPolicy.Namespace,
			networkPolicy.Name,
			cidrDirect,
			ipBlock.CIDR,
		)
		// Ports is not defined
		if len(Ports) == 0{
			args = []string{
				"-A", dispatchChainName,
				"-m", "comment", "--comment", comment,
				cidrDirect, ipBlock.CIDR,
				"-j", ipBlockChainName,
			}
			writeLine(policy.filterRules, args...)

		}
		for _, port := range Ports{
			args = []string{
				"-A", dispatchChainName,
				"-m", "comment", "--comment", comment,
				cidrDirect, ipBlock.CIDR,
				"-p", port.Protocol,
				"--dport", port.Port,
				"-j", ipBlockChainName,
			}
			writeLine(policy.filterRules, args...)
		}
	} else {
		// spec.podSelector is defined, so create rule match selected dst pods for ingress, match selected src pods for egress
		var xLabel []string
		for policyLabelKey, policyLabelValue := range networkPolicy.PodSelector {
			glog.V(4).Infof("accept rule selected by network policy %s/%s: %s match %s=%s",
				networkPolicy.Namespace, networkPolicy.Name, policyPodMatchDirect, policyLabelKey, policyLabelValue)

			xLabel = append(xLabel, policyLabelKey)
			xLabel = append(xLabel, policyLabelValue)

		}

		comment := fmt.Sprintf(`"accept rule selected by policy %s/%s: %s match policy spec podSelector, %s cidr %s"`,
			networkPolicy.Namespace,
			networkPolicy.Name,
			policyPodMatchDirect,
			cidrDirect,
			ipBlock.CIDR,
		)

		policyPodSetName := ennXLabelIPSetName(networkPolicy.Namespace, "pod", xLabel)

		// Ports is not defined
		if len(Ports) == 0 {
			args = []string{
				"-A", dispatchChainName,
				"-m", "comment", "--comment", comment,
				"-m", "set", "--match-set", policyPodSetName, policyPodMatchDirect,
				cidrDirect, ipBlock.CIDR,
				"-j", ipBlockChainName,
			}
			writeLine(policy.filterRules, args...)

		}
		for _, port := range Ports {
			args = []string{
				"-A", dispatchChainName,
				"-m", "comment", "--comment", comment,
				"-m", "set", "--match-set", policyPodSetName, policyPodMatchDirect,
				cidrDirect, ipBlock.CIDR,
				"-p", port.Protocol,
				"--dport", port.Port,
				"-j", ipBlockChainName,
			}
			writeLine(policy.filterRules, args...)

		}
	}

	policy.handleIPBlockChain(ruleType, networkPolicy, ipBlock, ipBlockChainName)
}

func (policy *EnnPolicy) handleIPBlockChain(ruleType         int,
                                            networkPolicy    *utilpolicy.NetworkPolicyInfo,
                                            ipBlock          utilpolicy.CIDRRange,
                                            ipBlockChainName string,
) {
	glog.V(4).Infof("process handle IPBlock cidr rules for networkPolicy %s/%s ", networkPolicy.Namespace, networkPolicy.Name)
	var args []string

	var cidrDirect string
	if ruleType == TYPE_INGRESS{
		cidrDirect = "src"
		glog.V(4).Infof("cidr ingress policy rules")
	} else if ruleType == TYPE_EGRESS{
		cidrDirect = "dst"
		glog.V(4).Infof("cidr egress policy rules")
	} else {
		glog.Errorf("invalid ruleType %d", ruleType)
		return
	}

	// create corresponding ipset for except cidr
	exceptCIDRSetName := ennExceptCIDRIPSetName(networkPolicy.Namespace, networkPolicy.Name, strconv.Itoa(ruleType), ipBlock.CIDR)
	exceptCIDRIPSet := &utilIPSet.IPSet{
		Name:    exceptCIDRSetName,
		Type:    utilIPSet.TypeHashNet,
	}
	err := policy.ipsetInterface.CreateIPSet(exceptCIDRIPSet, true)
	if err != nil{
		glog.Errorf("networkPolicy %s:%s create ipset for except cidr: %s err: %v",
			networkPolicy.Namespace,
			networkPolicy.Name,
			ipBlock.CIDR,
			err,
		)
		return
	}

	policy.activeIPSets[exceptCIDRIPSet.Name] = exceptCIDRIPSet

	err = policy.syncIPSetEntryForNet(exceptCIDRIPSet, ipBlock.ExceptCIDR)
	if err != nil{
		glog.Errorf("sync cidr ipset: %s err: %v", exceptCIDRSetName, err)
	}

	// create corresponding iptables rule
	comment := fmt.Sprintf(`"reject rule selected by policy %s/%s: %s excpet cidr"`,
		networkPolicy.Namespace,
		networkPolicy.Name,
		cidrDirect,
	)

	args = []string{
		"-A", ipBlockChainName,
		"-m", "comment", "--comment", comment,
		"-m", "set", "--match-set", exceptCIDRSetName, cidrDirect,
		"-j", "REJECT",
	}

	writeLine(policy.filterRules, args...)

	comment = fmt.Sprintf(`"accept default traffic of cidr %s"`, ipBlock.CIDR)

	args = []string{
		"-A", ipBlockChainName,
		"-m", "comment", "--comment", comment,
		"-j", "ACCEPT",
	}

	writeLine(policy.filterRules, args...)

}

// syncAllPodSets will sync all the ipsets defined in networkPolicy
// these ipsets include ipset for namespace, ipset for labels of namespace, ipset for labels of pod
func (policy *EnnPolicy) syncAllPodSets() error{

	glog.V(4).Infof("start to sync All PodSets")
	// sync ipset entry for each namespace
	for namespace, ipset := range policy.namespacePodSet{

		var podInfoMap utilpolicy.PodInfoMap

		podInfoMap, ok := policy.namespacePodMap[namespace]
		if !ok{
			glog.Errorf("cannot find any pod in namespace %s", namespace)
		} else{
			err := policy.syncIPSetEntry(ipset, podInfoMap)
			if err != nil{
				glog.Errorf("sync entry for ipset:%s of namespace %s failed %s", ipset.Name, namespace, err)
			}
		}
	}

	// sync ipset entry for pod label
	for namespacedLabel, ipset := range policy.podLabelSet{

		var podInfoMap utilpolicy.PodInfoMap

		podInfoMap, ok := policy.podMatchLabelMap[namespacedLabel]
		if !ok{
			glog.Errorf("cannot find any pod in namespace %s label %s=%s",
				namespacedLabel.Namespace,
				namespacedLabel.LabelKey,
				namespacedLabel.LabelValue,
			)
		} else{
			err := policy.syncIPSetEntry(ipset, podInfoMap)
			if err != nil{
				glog.Errorf("sync entry for ipset:%s of namespace %s label %s=%s failed %s",
					ipset.Name,
					namespacedLabel.Namespace,
					namespacedLabel.LabelKey,
					namespacedLabel.LabelValue,
					err,
				)
			}
		}
	}

	// sync ipset entry for namespace label
	for label, ipset := range policy.namespacePodLabelSet{

		var podInfoMap utilpolicy.PodInfoMap

		podInfoMap, ok := policy.namespaceMatchLabelMap[label]
		if !ok{
			glog.Errorf("cannot find any pod in namespace label %s=%s",
				label.LabelKey,
				label.LabelValue,
			)
		} else{
			err := policy.syncIPSetEntry(ipset, podInfoMap)
			if err != nil{
				glog.Errorf("sync entry for ipset:%s of namespace label %s=%s failed %s",
					ipset.Name,
					label.LabelKey,
					label.LabelValue,
					err,
				)
			}
		}
	}

	// sync ipset entry for spec podSelector
	for namespacedName, ipset := range policy.podXLabelSet{

		namespacedLabelMap, ok := policy.podXLabelMap[namespacedName]
		if !ok{
			glog.Errorf("cannot find namepacedLabelMap for namespaceName %s:%s", namespacedName.Namespace, namespacedName.Name)
			continue
		}

		labelLen := len(namespacedLabelMap.Label)
		// if spec podSelector is not defined, skip
		if labelLen == 0{
			continue
		}
		var podInfoMaps []utilpolicy.PodInfoMap
		podInfoMaps = make([]utilpolicy.PodInfoMap, labelLen)

		podInfoMapLen := 0
		for key, value := range namespacedLabelMap.Label{
			namespacedLabel := utilpolicy.NamespacedLabel{
				Namespace:   namespacedLabelMap.Namespace,
				LabelKey:    key,
				LabelValue:  value,
			}
			xPodInfoMap, ok := policy.podMatchLabelMap[namespacedLabel]
			// maybe some key/value pairs are invalid
			if !ok{
				glog.Errorf("cannot find any pod in namespace %s label %s=%s",
					namespacedLabel.Namespace,
					namespacedLabel.LabelKey,
					namespacedLabel.LabelValue,
				)
			} else {
				podInfoMaps[podInfoMapLen] = make(utilpolicy.PodInfoMap)
				for ip, infoMap := range xPodInfoMap{
					podInfoMaps[podInfoMapLen][ip] = infoMap
				}
				podInfoMapLen++
			}
		}
		// there is no valid key/value pairs for this ipset, so skip
		if podInfoMapLen == 0{
			glog.Errorf("invalid spec.podSelector for namespaceName %s:%s",namespacedName.Namespace, namespacedName.Name)
			continue
		}

		// AND operation for podInfoMaps
		podInfoMap := podInfoMaps[0]
		for i := 1; i < podInfoMapLen; i++{
			for ip := range podInfoMap{
				_, ok := podInfoMaps[i][ip]
				if !ok{
					delete(podInfoMap, ip)
				}
			}
		}

		err := policy.syncIPSetEntry(ipset, podInfoMap)
		if err != nil{
			glog.Errorf("sync entry for ipset:%s of namespaceName %s:%s failed %s",
				ipset.Name,
				namespacedName.Namespace,
				namespacedName.Name,
				err,
			)
		}

	}

	return nil
}

// when pod is add/delete/update, we call syncPodSets
// the pod set for pod label, namespace label and namespace could be changed
// so we need to sync the corresponding ipsets
func (policy *EnnPolicy) syncPodSets() error{

	glog.V(4).Infof("start to sync changed PodSets")
	// sync ipset entry for each pod label
	for _, podSetChange := range policy.podChanges.PodItems{
		for namespacedLabel, _ := range podSetChange.Previous{

			_, hasCurrent := podSetChange.Current[namespacedLabel]
			if hasCurrent{
				// find same namespacedLabel which means pod info is updated
				// so just sync in current map, do not need to sync ipset in previous map twice
				glog.V(6).Infof("skip sync podChange previous map for %q, sync in current map", namespacedLabel)
				continue
			}
			policy.trySyncPodLabelSet(namespacedLabel)
			policy.trySyncPodXLabelSet(namespacedLabel)

		}
		for namespacedLabel, _ := range podSetChange.Current{

			policy.trySyncPodLabelSet(namespacedLabel)
			policy.trySyncPodXLabelSet(namespacedLabel)
		}
	}

	// sync ipset entry for each namespace
	// the process is just as the same as pod label sync
	for _, namespaceSetChange := range policy.podChanges.NamespacePodItems{
		for namespace, _ := range namespaceSetChange.Previous{

			_, hasCurrent := namespaceSetChange.Current[namespace]
			if hasCurrent{
				// find same namespace which means pod info is updated
				// so just sync in current map, do not need to sync ipset in previous map twice
				glog.V(6).Infof("skip sync podChange previous map for %s, sync in current map", namespace)
				continue
			}
			policy.trySyncNamespacePodSet(namespace)
			// for each namespace of the updated pod, should sync namespacePodLabelSet also
//			for label := range policy.namespaceMatchLabelMap{
//				policy.trySyncNamespacePodLabelSet(label.LabelKey, label.LabelValue)
//			}

		}
		for namespace, _ := range namespaceSetChange.Current{

			policy.trySyncNamespacePodSet(namespace)
//			for label := range policy.namespaceMatchLabelMap{
//				policy.trySyncNamespacePodLabelSet(label.LabelKey, label.LabelValue)
//			}
		}
	}

	// sync ipset entry for each namespace labels
	// the process is just as the same as pod label sync
	for _, namespaceLabelSetChange := range policy.podChanges.NamespaceLabelItems{
		for namespace, _ := range namespaceLabelSetChange.Previous{

			_, hasCurrent := namespaceLabelSetChange.Current[namespace]
			if hasCurrent{
				// find same namespace which means pod info is updated
				// so just sync in current map, do not need to sync ipset in previous map twice
				glog.V(6).Infof("skip sync podChange previous map for %s, sync in current map", namespace)
				continue
			}
			// sync each namespace label
			namespaceInfo, ok := policy.namespaceInfoMap[namespace]
			if !ok{
				glog.Errorf("cannot find namespace Info in namespaceInfoMap, namespace is %s", namespace)
				continue
			}
			for k, v := range namespaceInfo.Labels{
				policy.trySyncNamespacePodLabelSet(k, v)
			}
		}
		for namespace, _ := range namespaceLabelSetChange.Current{

			namespaceInfo, ok := policy.namespaceInfoMap[namespace]
			if !ok{
				glog.Errorf("cannot find namespace Info in namespaceInfoMap, namespace is %s", namespace)
				continue
			}
			for k, v := range namespaceInfo.Labels{
				policy.trySyncNamespacePodLabelSet(k, v)
			}
		}
	}

	return nil
}

// when namespace is add/delete/update, we call syncNamespaceSets
// the namespace label ipset and namespace ipset could be changed
// syncNamespaceSets will sync ipsets for the changed namespace
func (policy *EnnPolicy) syncNamespaceSets() error{

	glog.V(4).Infof("start to sync changed namespace")
	for namespacedName, namespaceChange := range policy.namespaceChanges.Items{

		if namespaceChange.Previous != nil{

			// use a temp map to store all labels which need to be synced,
			// we need to sync both previous.Labels(delete unused ipset entries)
			// and current.Labels(add new ipset entries)
			syncLabel := make(map[utilpolicy.Label]bool)
			for k, v := range namespaceChange.Previous.Labels{
				label := utilpolicy.Label{
					LabelKey: k,
					LabelValue: v,
				}
				syncLabel[label] = true
			}
			// if we only get previous namespaceInfo, which means namespace is deleted
			// so deleted all the ipset corresponding with this namespace
			// including label ipset and namespace ipset
			// also delete element in corresponding maps
			if namespaceChange.Current == nil{
				policy.tryDeleteNamespacePodSet(namespaceChange.Previous.Name)
				// todo: also need to sync NamespacePodLabelSet
			} else{
				// means namespace updated, need to compare label maps
				// and delete unused label ipsets which in previous labelMap but not in current labelMap
				// sync new label ipsets which in current labelMap but not in previous labelMap
//				for k, v := range namespaceChange.Current.Labels {
//
//						policy.trySyncNamespacePodLabelSet(k, v)
//
//				}
				for k, v := range namespaceChange.Current.Labels {
					label := utilpolicy.Label{
						LabelKey: k,
						LabelValue: v,
					}
					syncLabel[label] = true
				}
			}
			for label, _ := range syncLabel{

				policy.trySyncNamespacePodLabelSet(label.LabelKey, label.LabelValue)
			}
		} else{
			if namespaceChange.Current == nil{
				glog.Errorf("invalid namespaceChange Map namespace:%s, both privous and current is nil", namespacedName.Name)
				continue
			}
			// if we only get current namespaceInfo, which means namespace is added
			// ipset for namespace is add in syncPolicyRules() and synced in syncPodSets()
			// so we just need to sync ipset for labels of namespace
			for k, v := range namespaceChange.Current.Labels{
				policy.trySyncNamespacePodLabelSet(k, v)
			}
		}
	}

	return nil
}

func (policy *EnnPolicy) syncIPSetEntry(ipset *utilIPSet.IPSet, podInfoMap utilpolicy.PodInfoMap) error{

	glog.V(4).Infof("start to sync entry for ipset %s:%s", ipset.Name, ipset.Type)
	kernelSet, err := policy.ipsetInterface.GetIPSet(ipset.Name)
	if err!= nil{
		return err
	}
	kernelEntries, err := policy.ipsetInterface.ListEntry(kernelSet)
	if err!= nil{
		return err
	}

	//delete unused entries
	for _, kernelEntry := range kernelEntries{
		glog.V(7).Infof("kernel entry is type:%s, ip:%s, port:%s, net:%s",
			kernelEntry.Type, kernelEntry.IP, kernelEntry.Port, kernelEntry.Net)
		ip, err := utilIPSet.EntryToString(kernelEntry)
		if err!= nil{
			glog.Errorf("get entry err ipset:%s, entry type:%s err:%v", kernelSet.Name, kernelEntry.Type, err)
			continue
		}
		_, ok := podInfoMap[ip]
		if !ok{
			glog.V(6).Infof("find unused entry %s of ipset %s:%s so delete it", ip, kernelSet.Name, kernelSet.Type)
			err := policy.ipsetInterface.DelEntry(kernelSet, kernelEntry, false)
			if err != nil{
				glog.Errorf("syncIPSetEntry error : %v", err)
				continue
			}
		}
	}
	// add new entries
	kernelEntryIPMap := make(map[string]bool)
	for _, kernelEntry := range kernelEntries{
		kernelEntryIPMap[kernelEntry.IP] = true
	}
	for ip := range podInfoMap{
		glog.V(7).Infof("podInfoMap ip is %s", ip)
		_, ok := kernelEntryIPMap[ip]
		if !ok{
			glog.V(6).Infof("find new entry %s of ipset %s:%s so add it", ip, kernelSet.Name, kernelSet.Type)
			entry := &utilIPSet.Entry{
				IP:    ip,
				Type:  utilIPSet.TypeHashIP,
			}
			err := policy.ipsetInterface.AddEntry(kernelSet, entry, true)
			if err != nil{
				glog.Errorf("syncIPSetEntry error : %v", err)
				continue
			}
		}
	}

	return nil
}

func (policy *EnnPolicy) syncIPSetEntryForNet(ipset *utilIPSet.IPSet, nets []string) error{


	glog.V(4).Infof("start to sync entry for ipset %s:%s", ipset.Name, ipset.Type)
	kernelSet, err := policy.ipsetInterface.GetIPSet(ipset.Name)
	if err!= nil{
		return err
	}
	kernelEntries, err := policy.ipsetInterface.ListEntry(kernelSet)
	if err!= nil{
		return err
	}

	netsMap := make(map[string]bool)
	for _, net := range nets{
		netsMap[net] = true
	}
	kernelEntryIPMap := make(map[string]bool)
	for _, kernelEntry := range kernelEntries{
		kernelEntryIPMap[kernelEntry.Net] = true
	}

	//delete unused entries
	for _, kernelEntry := range kernelEntries{
		glog.V(7).Infof("kernel entry is type:%s, ip:%s, port:%s, net:%s",
			kernelEntry.Type, kernelEntry.IP, kernelEntry.Port, kernelEntry.Net)
		net, err := utilIPSet.EntryToString(kernelEntry)
		if err!= nil{
			glog.Errorf("get entry err ipset:%s, entry type:%s err:%v", kernelSet.Name, kernelEntry.Type, err)
			continue
		}
		_, ok := netsMap[net]
		if !ok{
			glog.V(6).Infof("find unused entry: %s of ipset %s:%s so delete it", net, kernelSet.Name, kernelSet.Type)
			err := policy.ipsetInterface.DelEntry(kernelSet, kernelEntry, false)
			if err != nil{
				glog.Errorf("syncIPSetEntry error : %v", err)
				continue
			}
		}
	}

	// add new entries
	for net := range netsMap{
		glog.V(7).Infof("new hash/net is %s", net)
		_, ok := kernelEntryIPMap[net]
		if !ok{
			glog.V(6).Infof("find new entry: %s of ipset %s:%s so add it", net, kernelSet.Name, kernelSet.Type)
			entry := &utilIPSet.Entry{
				Net:   net,
				Type:  utilIPSet.TypeHashNet,
			}
			err := policy.ipsetInterface.AddEntry(kernelSet, entry, true)
			if err != nil{
				glog.Errorf("syncIPSetEntry error : %v", err)
				continue
			}
		}
	}

	return nil
}

// trySyncPodXLabelSet will try sync ipset entries with given namespacedLabel
// when a pod is added/deleted/updated, trySyncPodXLabelSet will scan the whole podXLabelMap
// and find whether there is a namespacedLabelMap item contains corresponding namespacedLabel
// if find a item, sync corresponding ipset
func (policy *EnnPolicy) trySyncPodXLabelSet(namespaceLabel utilpolicy.NamespacedLabel){

	glog.V(4).Infof("try to sync PodXLabelSet for namespacedLabel %s:%s=%s", namespaceLabel.Namespace, namespaceLabel.LabelKey, namespaceLabel.LabelValue)

	// scan all podXLabelMap to find namespacedLabelMap contains namespacedLabel(namespaceLabel.Namespace/namespaceLabel.LabelKey/namespaceLabel.LabelValue)
	for namespacedName, namespacedLabelMap := range policy.podXLabelMap{

		if strings.Compare(namespacedLabelMap.Namespace, namespaceLabel.Namespace) == 0{
			containLabel := false
			for key, value := range namespacedLabelMap.Label{
				if strings.Compare(key,namespaceLabel.LabelKey) == 0 && strings.Compare(value,namespaceLabel.LabelValue) == 0{

					containLabel = true
					break;

				}
			}

			if !containLabel{
				continue
			}

			// 1. if find, we should get the ipset name
			ipset, ok := policy.podXLabelSet[namespacedName]
			if !ok{
				glog.Errorf("trySyncPodXLabelSet: cannot find podXLabelSet for namespaceName %s:%s",
					namespacedName.Namespace,
					namespacedName.Name,
				)
				continue
			}

			// check whether ipset is active, delete inactive ipset from corresponding map
			_, ok = policy.activeIPSets[ipset.Name]
			if !ok{
				glog.V(6).Infof("ipset: %s is not active, so delete this ipset from podXLabelSet", ipset.Name)
				delete(policy.podXLabelSet, namespacedName)
				continue
			}

			// 2. get the podInfoMaps for each namespacedLabel
			labelLen := len(namespacedLabelMap.Label)
			// if spec podSelector is not defined, skip
			if labelLen == 0{
				continue
			}
			var podInfoMaps []utilpolicy.PodInfoMap
			podInfoMaps = make([]utilpolicy.PodInfoMap, labelLen)

			podInfoMapLen := 0
			for key, value := range namespacedLabelMap.Label{
				namespacedLabel := utilpolicy.NamespacedLabel{
					Namespace:   namespacedLabelMap.Namespace,
					LabelKey:    key,
					LabelValue:  value,
				}
				xPodInfoMap, ok := policy.podMatchLabelMap[namespacedLabel]
				// maybe some key/value pairs are invalid
				if !ok{
					glog.Errorf("trySyncPodXLabelSet: cannot find any pod in namespace %s label %s=%s",
						namespacedLabel.Namespace,
						namespacedLabel.LabelKey,
						namespacedLabel.LabelValue,
					)
				} else {
					podInfoMaps[podInfoMapLen] = make(utilpolicy.PodInfoMap)
					for ip, infoMap := range xPodInfoMap{
						podInfoMaps[podInfoMapLen][ip] = infoMap
					}
					podInfoMapLen++
				}
			}

			// 3. AND operation for podInfoMaps
			podInfoMap := podInfoMaps[0]
			for i := 1; i < podInfoMapLen; i++{
				for ip := range podInfoMap{
					_, ok := podInfoMaps[i][ip]
					if !ok{
						delete(podInfoMap, ip)
					}
				}
			}

			// 4. sync corresponding ipset
			err := policy.syncIPSetEntry(ipset, podInfoMap)
			if err != nil{
				glog.Errorf("trySyncPodXLabelSet: sync entry for ipset:%s of namespaceName %s:%s failed %s",
					ipset.Name,
					namespacedName.Namespace,
					namespacedName.Name,
					err,
				)
			}
		}
	}
}

// trySyncPodLabelSet will try sync ipset entries with given namespaceLabel
// namespaceLabel is constructed by pod namespace and pod labels
// if ipset is not active, will delete this ipset from podLabelSet map
// so this ipset will never be synced untill it turns to be active again
// function checkUnusedIPSets() will delete these inactive ipset from kernel
func (policy *EnnPolicy) trySyncPodLabelSet(namespaceLabel utilpolicy.NamespacedLabel){

	glog.V(4).Infof("try to sync PodLabelSet for namespacedLabel %s:%s=%s", namespaceLabel.Namespace, namespaceLabel.LabelKey, namespaceLabel.LabelValue)
	ipset, has := policy.podLabelSet[namespaceLabel]
	if !has{
		glog.V(7).Infof("cannot find namespaceLabel %s:%s=%s in podLabelSet, maybe this namespaceLabel is not used in policy",
			namespaceLabel.Namespace, namespaceLabel.LabelKey, namespaceLabel.LabelValue)
		return
	}
	// check whether ipset is active, delete inactive ipset from corresponding map
	_, ok := policy.activeIPSets[ipset.Name]
	if !ok{
		glog.V(6).Infof("ipset: %s is not active, so delete this ipset from podLabelSet", ipset.Name)
		delete(policy.podLabelSet, namespaceLabel)
		return
	}
	// this pod set is defined in networkPolicy, and is active, so sync this ipset
	// since podMatchLabelMap is updated, so we just sync entry from podMatchLabelMap
	podInfoMap, ok := policy.podMatchLabelMap[namespaceLabel]
	if !ok{
		glog.Errorf("cannot find any pod in namespace %s label %s=%s",
			namespaceLabel.Namespace,
			namespaceLabel.LabelKey,
			namespaceLabel.LabelValue,
		)
	}else{
		err := policy.syncIPSetEntry(ipset, podInfoMap)
		if err != nil{
			glog.Errorf("sync entry for ipset:%s of namespace %s label %s=%s failed %s",
				ipset.Name,
				namespaceLabel.Namespace,
				namespaceLabel.LabelKey,
				namespaceLabel.LabelValue,
				err,
			)
		}
	}
}

// trySyncNamespacePodLabelSet will try sync ipset entries with given namespace label
// if ipset is not active, will delete this ipset from namespacePodLabelSet map
// so this ipset will never be synced untill it turns to be active again
// function checkUnusedIPSets() will delete these inactive ipset from kernel
func (policy *EnnPolicy) trySyncNamespacePodLabelSet(k, v string){

	glog.V(4).Infof("try to sync NamespacePodLabelSet for namespacedLabel %s=%s", k, v)
	label := utilpolicy.Label{
		LabelKey:     k,
		LabelValue:   v,
	}
	ipset, has := policy.namespacePodLabelSet[label]
	if !has{
		glog.V(7).Infof("cannot find namespaceLabel %s=%s in namespacePodLabelSet, maybe this namespaceLabel is not used in policy",
			k, v)
		return
	}
	// check whether ipset is active, delete inactive ipset from corresponding map
	_, ok := policy.activeIPSets[ipset.Name]
	if !ok{
		glog.V(6).Infof("ipset: %s is not active, so delete this ipset from namespacePodLabelSet", ipset.Name)
		delete(policy.namespacePodLabelSet, label)
		return
	}
	// this pod set is defined in networkPolicy, and is active, so sync this ipset
	// since namespaceMatchLabelMap is updated, so we just sync entry from namespaceMatchLabelMap
	podInfoMap, ok := policy.namespaceMatchLabelMap[label]
	if !ok{
		glog.Errorf("cannot find any pod in namespace label %s=%s",
			label.LabelKey,
			label.LabelValue,
		)
	}else{
		err := policy.syncIPSetEntry(ipset, podInfoMap)
		if err != nil{
			glog.Errorf("sync entry for ipset:%s of namespace label %s=%s failed %s",
				ipset.Name,
				label.LabelKey,
				label.LabelValue,
				err,
			)
		}
	}
}

func (policy *EnnPolicy) tryDeleteNamespacePodLabelSet(k, v string){

	glog.V(4).Infof("try to delete NamespacePodLabelSet for namespacedLabel %s=%s", k, v)
	label := utilpolicy.Label{
		LabelKey:     k,
		LabelValue:   v,
	}
	ipset, has := policy.namespacePodLabelSet[label]
	if !has{
		glog.V(7).Infof("cannot find namespaceLabel %s=%s in namespacePodLabelSet, maybe this namespaceLabel is not used in policy",
			k, v)
		return
	}
	err := policy.ipsetInterface.DestroyIPSet(ipset)
	if err != nil{
		glog.Errorf("delete ipset for namespaceLabel %s=%s fail, %s", k, v, err)
	}
	delete(policy.namespacePodLabelSet, label)
}

// trySyncNamespacePodLabelSet will try sync ipset entries with given namespace
// if ipset is not active, will delete this ipset from namespacePodSet map
// so this ipset will never be synced untill it turns to be active again
// function checkUnusedIPSets() will delete these inactive ipset from kernel
func (policy *EnnPolicy) trySyncNamespacePodSet(namespace string){

	glog.V(4).Infof("try to sync NamespacePodSet for namespace %s", namespace)
	ipset, has := policy.namespacePodSet[namespace]
	if !has{
		glog.V(7).Infof("cannot find namespace %s in namespacePodSet, maybe this namespace is not used in policy", namespace)
		return
	}
	// check whether ipset is active, delete inactive ipset from corresponding map
	_, ok := policy.activeIPSets[ipset.Name]
	if !ok{
		glog.V(6).Infof("ipset: %s is not active, so delete this ipset from namespacePodSet", ipset.Name)
		delete(policy.namespacePodSet, namespace)
		return
	}
	// this pod set is defined in networkPolicy, and is active, so sync this ipset
	// since namespacePodMap is updated, so we just sync entry from namespacePodMap
	podInfoMap, ok := policy.namespacePodMap[namespace]
	if !ok{
		glog.Errorf("cannot find any pod in namespace %s", namespace)
	} else{
		err := policy.syncIPSetEntry(ipset, podInfoMap)
		if err != nil{
			glog.Errorf("sync entry for ipset:%s of namespace %s failed %s", ipset.Name, namespace, err)
		}
	}
}

func (policy *EnnPolicy) tryDeleteNamespacePodSet(namespace string){

	glog.V(4).Infof("try to delete NamespacePodSet for namespace %s", namespace)
	ipset, has := policy.namespacePodSet[namespace]
	if !has{
		glog.V(7).Infof("cannot find namespace %s in namespacePodSet, maybe this namespace is not used in policy", namespace)
		return
	}
	err := policy.ipsetInterface.DestroyIPSet(ipset)
	if err != nil{
		glog.Errorf("delete ipset for namespace %s fail, %s", namespace, err)
	}
	delete(policy.namespacePodSet, namespace)
}

// scan the whole ipsets list to find whether there is unused ipset
func (policy *EnnPolicy) checkUnusedIPSets() error{

	glog.V(4).Infof("start to check unsued ipsets")
	ipsetsName, err := policy.ipsetInterface.ListIPSetsName()
	if err != nil{
		return fmt.Errorf("check unused ipsets error %v", err)
	}
	for _, ipsetName := range ipsetsName{
		// check whech ipset is created by enn-policy
		if strings.HasPrefix(ipsetName, "ENN"){
			_, ok := policy.activeIPSets[ipsetName]
			if !ok{
				ipset, err := policy.ipsetInterface.GetIPSet(ipsetName)
				if err!= nil{
					glog.Errorf("check unused ipsets error %v", err)
					continue
				}
				glog.V(4).Infof("find unusd ipset %s type %s", ipset.Name, ipset.Type)
				err = policy.ipsetInterface.DestroyIPSet(ipset)
				if err!= nil{
					glog.Errorf("check unused ipsets error %v", err)
				}
			}
		}
	}

	for namespaceLabel, ipset := range policy.podLabelSet{
		ipsetName := ipset.Name
		_, ok := policy.activeIPSets[ipsetName]
		if !ok{
			glog.V(4).Infof("find inavtive ipset:%s in podLabelSet so delete it", ipsetName)
			delete(policy.podLabelSet, namespaceLabel)
		}
	}

	for label, ipset := range policy.namespacePodLabelSet{
		ipsetName := ipset.Name
		_, ok := policy.activeIPSets[ipsetName]
		if !ok{
			glog.V(4).Infof("find inavtive ipset:%s in namespacePodLabelSet so delete it", ipsetName)
			delete(policy.namespacePodLabelSet, label)
		}
	}

	for namespace, ipset := range policy.namespacePodSet{
		ipsetName := ipset.Name
		_, ok := policy.activeIPSets[ipsetName]
		if !ok{
			glog.V(4).Infof("find inavtive ipset:%s in namespacePodSet so delete it", ipsetName)
			delete(policy.namespacePodSet, namespace)
		}
	}

	for namespacedName, ipset := range policy.podXLabelSet{
		ipsetName := ipset.Name
		_, ok := policy.activeIPSets[ipsetName]
		if !ok{
			glog.V(4).Infof("find inavtive ipset:%s in podXLabelSet so delete it", ipsetName)
			delete(policy.podXLabelSet, namespacedName)
		}
	}

	return nil
}

func (policy *EnnPolicy) CleanupLeftovers(){
	// should first cleanup iptables then cleanup ipsets
	// because ipset cannot be deleted if it's used in iptables (reference > 0)
	// should cleanup policy rules before cleanup enn entry
	// because enn entry chains cannot be deleted when they are still referred by other chain
	//policy.cleanupPolicy()
	//policy.cleanupEntry()
	policy.cleanupIPTables()
	policy.cleanupIPSets()
}

func (policy *EnnPolicy) cleanupIPTables() error{

	// cleanup -A INPUT -j ENN-INPUT
	inputLists, err := policy.iptablesInterface.List(FILTER_TABLE, INPUT_CHAIN)
	if err != nil{
		glog.Errorf("cleanup input entry failed %v", err)
	} else {
		for i, input := range inputLists {
			if strings.Contains(input, "ENN") {
				arg := strconv.Itoa(i)
				policy.iptablesInterface.Delete(FILTER_TABLE, INPUT_CHAIN, arg)
				if err != nil {
					glog.Errorf("cleanup entry failed %v", err)
				}
			}
		}
	}

	// cleanup -A OUTPUT -j ENN-OUTPUT
	inputLists, err = policy.iptablesInterface.List(FILTER_TABLE, OUTPUT_CHAIN)
	if err != nil{
		glog.Errorf("cleanup output entry failed %v", err)
	} else {
		for i, input := range inputLists {
			if strings.Contains(input, "ENN") {
				arg := strconv.Itoa(i)
				policy.iptablesInterface.Delete(FILTER_TABLE, OUTPUT_CHAIN, arg)
				if err != nil {
					glog.Errorf("cleanup entry failed %v", err)
				}
			}
		}
	}

	// cleanup -A FORWARD -j ENN-FORWARD
	inputLists, err = policy.iptablesInterface.List(FILTER_TABLE, FORWARD_CHAIN)
	if err != nil{
		glog.Errorf("cleanup forward entry failed %v", err)
	} else {
		for i, input := range inputLists {
			if strings.Contains(input, "ENN") {
				arg := strconv.Itoa(i)
				policy.iptablesInterface.Delete(FILTER_TABLE, FORWARD_CHAIN, arg)
				if err != nil {
					glog.Errorf("cleanup entry failed %v", err)
				}
			}
		}
	}

	// cleanup chain start with ENN
	iptablesData := bytes.NewBuffer(nil)
	if err := policy.k8siptablesInterface.SaveInto(utiliptables.TableFilter, iptablesData); err != nil {
		glog.Errorf("Failed to execute iptables-save for %s: %v", utiliptables.TableFilter, err)
		return err
	} else {
		existingFilterChains := utiliptables.GetChainLines(utiliptables.TableFilter, iptablesData.Bytes())
		filterChains := bytes.NewBuffer(nil)
		filterRules := bytes.NewBuffer(nil)
		writeLine(filterChains, "*filter")

		for chain := range existingFilterChains {
			chainString := string(chain)
			if strings.HasPrefix(chainString, "ENN-"){
				writeLine(filterChains, existingFilterChains[chain]) // flush
				writeLine(filterRules, "-X", chainString)         // delete
			}
		}

		writeLine(filterRules, "COMMIT")
		filterLines := append(filterChains.Bytes(), filterRules.Bytes()...)
		// Write it.
		if err := policy.k8siptablesInterface.Restore(utiliptables.TableFilter, filterLines, utiliptables.NoFlushTables, utiliptables.RestoreCounters); err != nil {
			glog.Errorf("Failed to execute iptables-restore for %s: %v", utiliptables.TableFilter, err)
			return err
		}
	}

	return nil
}

func (policy *EnnPolicy) cleanupPolicy() error{

	glog.V(4).Infof("start to cleanupPolicy")

	ennChainName  := make([]string, 0)
	ennPolicyName := make([]string, 0)

	filterLists, err := policy.iptablesInterface.ListChains(FILTER_TABLE)
	if err != nil {
		return fmt.Errorf("cleanupPolicy err %s", err)
	}

	for _, chainName := range filterLists{
		if strings.HasPrefix(chainName, "ENN"){
			ennChainName = append(ennChainName, chainName)
			if !strings.HasPrefix(chainName, ENN_INPUT_CHAIN) &&
				!strings.HasPrefix(chainName, ENN_OUTPUT_CHAIN) &&
				!strings.HasPrefix(chainName, ENN_FORWARD_CHAIN){
				ennPolicyName = append(ennPolicyName, chainName)
			}
		}
	}

	// first flush all chains created by enn-policy
	for _, chain := range ennChainName{
		err := policy.iptablesInterface.ClearChain(FILTER_TABLE, chain)
		if err != nil{
			glog.Errorf("cleanupPolicy flush chain %s err %v", chain, err)
		}
	}
	// then delete all policy chains created by enn-policy
	// (except entry chains e.g ENN-INPUT, ENN-OUTPUT, ENN-FORWARD)
	for _, chain := range ennPolicyName {
		err := policy.iptablesInterface.DeleteChain(FILTER_TABLE, chain)
		if err != nil{
			glog.Errorf("cleanupPolicy delete chain %s err %v", chain, err)
		}
	}

	return nil
}

func (policy *EnnPolicy) cleanupEntry(){

	glog.V(4).Infof("start to cleanupEntry")
	// cleanup -A INPUT -j ENN-INPUT
	// cleanup -N ENN-INPUT
	inputLists, err := policy.iptablesInterface.List(FILTER_TABLE, INPUT_CHAIN)
	if err != nil{
		glog.Errorf("cleanup input entry failed %v", err)
	} else {
		for i, input := range inputLists {
			if strings.Contains(input, "ENN") {
				arg := strconv.Itoa(i)
				policy.iptablesInterface.Delete(FILTER_TABLE, INPUT_CHAIN, arg)
				if err != nil {
					glog.Errorf("cleanup entry failed %v", err)
				}
			}
		}
	}
	err = policy.iptablesInterface.DeleteChain(FILTER_TABLE, ENN_INPUT_CHAIN)
	if err != nil{
		glog.Errorf("cleanup input entry failed %v", err)
	}
	// cleanup -A OUTPUT -j ENN-OUTPUT
	// cleanup -N ENN-OUTPUT
	inputLists, err = policy.iptablesInterface.List(FILTER_TABLE, OUTPUT_CHAIN)
	if err != nil{
		glog.Errorf("cleanup output entry failed %v", err)
	} else {
		for i, input := range inputLists {
			if strings.Contains(input, "ENN") {
				arg := strconv.Itoa(i)
				policy.iptablesInterface.Delete(FILTER_TABLE, OUTPUT_CHAIN, arg)
				if err != nil {
					glog.Errorf("cleanup entry failed %v", err)
				}
			}
		}
	}
	err = policy.iptablesInterface.DeleteChain(FILTER_TABLE, ENN_OUTPUT_CHAIN)
	if err != nil{
		glog.Errorf("cleanup ouput entry failed %v", err)
	}
	// cleanup -A FORWARD -j ENN-FORWARD
	// cleanup -N ENN-FORWARD
	inputLists, err = policy.iptablesInterface.List(FILTER_TABLE, FORWARD_CHAIN)
	if err != nil{
		glog.Errorf("cleanup forward entry failed %v", err)
	} else {
		for i, input := range inputLists {
			if strings.Contains(input, "ENN") {
				arg := strconv.Itoa(i)
				policy.iptablesInterface.Delete(FILTER_TABLE, FORWARD_CHAIN, arg)
				if err != nil {
					glog.Errorf("cleanup entry failed %v", err)
				}
			}
		}
	}
	err = policy.iptablesInterface.DeleteChain(FILTER_TABLE, ENN_FORWARD_CHAIN)
	if err != nil{
		glog.Errorf("cleanup forward entry failed %v", err)
	}
}

func (policy *EnnPolicy) cleanupIPSets(){

	glog.V(4).Infof("start to cleanupIPSets")
	ipsetsName, err := policy.ipsetInterface.ListIPSetsName()
	if err != nil{
		glog.Errorf("cleanupIPSets error %v", err)
	}
	for _, ipsetName := range ipsetsName{
		if strings.HasPrefix(ipsetName, "ENN"){
			ipset, err := policy.ipsetInterface.GetIPSet(ipsetName)
			if err!= nil{
				glog.Errorf("cleanupIPSets get ipset %s error %v", ipsetName, err)
				continue
			}
			err = policy.ipsetInterface.DestroyIPSet(ipset)
			if err!= nil{
				glog.Errorf("cleanupIPSets destroy ipset %s error %v", ipsetName, err)
			}
		}
	}
}

func ennNamespaceIngressChainName(namespace string) string{
	hash := sha256.Sum256([]byte(namespace))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-INGRESS-" + encoded[:16]
}

func ennNamespaceEgressChainName(namespace string) string{
	hash := sha256.Sum256([]byte(namespace))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-EGRESS-" + encoded[:16]
}

func ennIPRangeIngressChainName(namespace string, ipRange string) string{
	hash := sha256.Sum256([]byte(namespace + ipRange))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-PLY-IN-" + encoded[:16]
}

func ennIPRangeEgressChainName(namespace string, ipRange string) string{
	hash := sha256.Sum256([]byte(namespace + ipRange))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-PLY-E-" + encoded[:16]
}

// [match] separate different dispatch kind (onlyPorts/podSelector/namespaceSelector/ipBlock)
func ennDispatchChainName(namespace string, policyName string, inORe string, match string, cidr string, labelKey string, labelValue string) string{
	hash := sha256.Sum256([]byte(namespace + policyName + inORe + match + cidr + labelKey + labelValue))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-DPATCH-" + encoded[:16]
}

func ennIPBlockChainName(namespace string, policyName string, inORe string, cidr string) string{
	hash := sha256.Sum256([]byte(namespace + policyName + inORe + cidr))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-IPCIDR-" + encoded[:16]
}

func ennNamespaceIPSetName(namespace string) string{
	hash := sha256.Sum256([]byte(namespace))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-NS-" + encoded[:16]
}

func ennIPRangeIPSetName(ipRange string) string{
	hash := sha256.Sum256([]byte(ipRange))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-RANGEIP-" + encoded[:16]
}

// [kind] separate namespacedLabel from different kind (pod/namespace)
func ennLabelIPSetName(namespace string, kind string, labelKey string, labelValue string) string{
	hash := sha256.Sum256([]byte(namespace + kind + labelKey + labelValue))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-PODSET-" + encoded[:16]
}

// ipset name for possible flannel ips and docker ips
func ennFlannelIPSetName(flannelNet string, flannelLen string) string{
	hash := sha256.Sum256([]byte(flannelNet + flannelLen))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-FLANNEL-" + encoded[:16]
}

// [kind] separate namespacedLabel from different kind (pod/namespace)
// [labels] should come in pairs (key/value)
func ennXLabelIPSetName(namespace string, kind string, labels []string) string{
	sort.Strings(labels)
	var xLabel = ""
	for _, label := range labels{
		xLabel += label
	}
	hash := sha256.Sum256([]byte(namespace + kind + xLabel))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-PODSET-" + encoded[:16]
}

func ennNSLabelIPSetName(labelKey string, labelValue string) string{
	hash := sha256.Sum256([]byte(labelKey + labelValue))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-NSSET-" + encoded[:16]
}

func ennExceptCIDRIPSetName(namespace string, policyName string, inORe string, cidr string) string{
	hash := sha256.Sum256([]byte(namespace + policyName + inORe + cidr))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "ENN-IPTSET-" + encoded[:16]
}

// Join all words with spaces, terminate with newline and write to buf.
func writeLine(buf *bytes.Buffer, words ...string) {
	// We avoid strings.Join for performance reasons.
	for i := range words {
		buf.WriteString(words[i])
		if i < len(words)-1 {
			buf.WriteByte(' ')
		} else {
			buf.WriteByte('\n')
		}
	}
}

