package app

import (
	"enn-policy/app/options"
	policyConfig "enn-policy/pkg/policy/config"
	ennPolicy "enn-policy/pkg/policy"
	policyUtil "enn-policy/pkg/policy/util"
	"enn-policy/pkg/util/ipset"
	"enn-policy/pkg/util/iptables"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/informers"
	"k8s.io/apimachinery/pkg/util/wait"
	utilexec "k8s.io/utils/exec"
	utiliptables "enn-policy/pkg/util/k8siptables"
	utildbus "enn-policy/pkg/util/dbus"

	"time"
	"sync"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const Version    = "0.6"
const LastUpdate = "2018.4.11"

type EnnPolicyServer struct {
	Policy                     *ennPolicy.EnnPolicy
	Config                     *options.EnnPolicyConfig
	Client                     *kubernetes.Clientset

	ConfigSyncPeriod           time.Duration
	NetworkPolicyEventHandler  policyConfig.NetworkPolicyHandler
	PodEventHandler            policyConfig.PodHandler
	NamespaceEventHandler      policyConfig.NamespaceHandler
}

func NewEnnPolicyServer(
    policy                     *ennPolicy.EnnPolicy,
    config                     *options.EnnPolicyConfig,
    client                     *kubernetes.Clientset,
    networkPolicyEventHandler  policyConfig.NetworkPolicyHandler,
    podEventHandler            policyConfig.PodHandler,
    namespaceEventHandler      policyConfig.NamespaceHandler,
)(*EnnPolicyServer, error){
	return &EnnPolicyServer{
		Policy:                     policy,
		Config:                     config,
		Client:                     client,
		NetworkPolicyEventHandler:  networkPolicyEventHandler,
		PodEventHandler:            podEventHandler,
		NamespaceEventHandler:      namespaceEventHandler,
	},nil
}

func NewEnnPolicyServerDefault(config *options.EnnPolicyConfig)(*EnnPolicyServer, error){

	if config.Kubeconfig == "" && config.Master == "" {
		glog.Warningf("Neither --kubeconfig nor --master was specified.  Using default API client.  This might not work.")
		/*todo need modify default config path*/
		config.Kubeconfig = "/var/lib/enn-policy/kubeconfig"
	}

	clientconfig, err := clientcmd.BuildConfigFromFlags(config.Master, config.Kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clientconfig)
	if err != nil {
		glog.Fatalf("Invalid API configuration: %v", err)
		panic(err.Error())
	}

	node, err := policyUtil.GetNode(clientset,config.HostnameOverride)
	if err != nil{
		return nil, fmt.Errorf("NewEnnPolicy failure: GetNode fall: %s", err.Error())
	}
	hostname := node.Name

	nodeIP, err := policyUtil.InternalGetNodeHostIP(node)
	if err != nil{
		return nil, fmt.Errorf("NewEnnPolicy failure: GetNodeIP fall: %s", err.Error())
	}

	execerInterface := utilexec.New()
	ipsetInterface := ipset.NewEnnIPSet(execerInterface)
	iptablesInterface := iptables.NewEnnIPTables()

	/*k8s iptables util*/
	protocol := utiliptables.ProtocolIpv4
	var k8siptInterface utiliptables.Interface
	var dbus utildbus.Interface

	dbus = utildbus.New()
	k8siptInterface = utiliptables.New(execerInterface, dbus, protocol)

	if err != nil {
		return nil, fmt.Errorf("iptable init failed %s" + err.Error())
	}
	policy, err := ennPolicy.NewEnnPolicy(
		clientset,
		config,
		hostname,
		nodeIP,
		execerInterface,
		ipsetInterface,
		iptablesInterface,
		k8siptInterface,
	)
	if err != nil{
		return nil, err
	}

	var networkPolicyEventHandler policyConfig.NetworkPolicyHandler
	var podEventHandler policyConfig.PodHandler
	var namespaceEventHandler policyConfig.NamespaceHandler

	networkPolicyEventHandler = policy
	podEventHandler = policy
	namespaceEventHandler = policy

	return NewEnnPolicyServer(
		policy,
		config,
		clientset,
		networkPolicyEventHandler,
		podEventHandler,
		namespaceEventHandler,
	)
}

func CleanUpAndExit() {
	execerInterface := utilexec.New()
	ipsetInterface := ipset.NewEnnIPSet(execerInterface)
	iptablesInterface := iptables.NewEnnIPTables()

	/*k8s iptables util*/
	protocol := utiliptables.ProtocolIpv4
	var k8siptInterface utiliptables.Interface
	var dbus utildbus.Interface

	dbus = utildbus.New()
	k8siptInterface = utiliptables.New(execerInterface, dbus, protocol)

	policy, err := ennPolicy.FakePolicy(
		execerInterface,
		ipsetInterface,
		iptablesInterface,
		k8siptInterface,
	)
	if err != nil{
		glog.Errorf("create fake policy error %v", err)
		return
	}
	policy.CleanupLeftovers()
}

func ShowVersion() {
	fmt.Printf("version: %s\n", Version)
	fmt.Printf("last update time: %s\n", LastUpdate)
}

func (s *EnnPolicyServer) Run() error{

	glog.V(0).Infof("start to run enn policy")
	glog.V(0).Infof("enn-policy version is: %s", Version)
	glog.V(0).Infof("enn-policy last update time is: %s", LastUpdate)
	var StopCh chan struct{}
	var wg sync.WaitGroup

	informerFactory := informers.NewSharedInformerFactory(s.Client, s.ConfigSyncPeriod)


	// Create configs (i.e. Watches for Services and Endpoints)
	// Note: RegisterHandler() calls need to happen before creation of Sources because sources
	// only notify on changes, and the initial update (on process start) may be lost if no handlers
	// are registered yet.
	networkPolicyConfig := policyConfig.NewNetworkPolicyConfig(informerFactory.Networking().V1().NetworkPolicies(), s.ConfigSyncPeriod)
	networkPolicyConfig.RegisterEventHandler(s.NetworkPolicyEventHandler)
	go networkPolicyConfig.Run(wait.NeverStop)

	podConfig := policyConfig.NewPodConfig(informerFactory.Core().V1().Pods(), s.ConfigSyncPeriod)
	podConfig.RegisterEventHandler(s.PodEventHandler)
	go podConfig.Run(wait.NeverStop)

	namespaceConfig := policyConfig.NewNamespaceConfig(informerFactory.Core().V1().Namespaces(), s.ConfigSyncPeriod)
	namespaceConfig.RegisterEventHandler(s.NamespaceEventHandler)
	go namespaceConfig.Run(wait.NeverStop)

	// This has to start after the calls to NewServiceConfig and NewEndpointsConfig because those
	// functions must configure their shared informer event handlers first.
	go informerFactory.Start(wait.NeverStop)

	StopCh = make(chan struct{})

	wg.Add(1)
	go s.Policy.SyncLoop(StopCh, &wg)


	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	glog.V(0).Infof("get sys terminal and exit enn policy")
	StopCh <- struct{}{}

	wg.Wait()

	return nil
}

