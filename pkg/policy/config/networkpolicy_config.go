package config

import (
	api "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/networking/v1"
	listers "k8s.io/client-go/listers/networking/v1"

	"time"
	"github.com/golang/glog"
	"fmt"
)

// NetworkPolicyHandler is an abstract interface of objects which receive
// notifications about NetworkPolicy object changes.
type NetworkPolicyHandler interface {
	// NetworkPolicyAdd is called whenever creation of new NetworkPolicy object
	// is observed.
	OnNetworkPolicyAdd(networkPolicy *api.NetworkPolicy)
	// OnNetworkPolicyUpdate is called whenever modification of an existing
	// NetworkPolicy object is observed.
	OnNetworkPolicyUpdate(oldNetworkPolicy, networkPolicy *api.NetworkPolicy)
	// OnNetworkPolicyDelete is called whenever deletion of an existing NetworkPolicy
	// object is observed.
	OnNetworkPolicyDelete(networkPolicy *api.NetworkPolicy)
	// OnNetworkPolicySynced is called once all the initial even handlers were
	// called and the state is fully propagated to local cache.
	OnNetworkPolicySynced()
}

// NetworkPolicyConfig tracks a set of NetworkPolicy configurations.
// It accepts "set", "add" and "remove" operations of NetworkPolicy via channels, and invokes registered handlers on change.
type NetworkPolicyConfig struct {
	lister        listers.NetworkPolicyLister
	listerSynced  cache.InformerSynced
	eventHandlers []NetworkPolicyHandler
}

// NetworkPolicyConfig creates a new NetworkPolicyConfig.
func NewNetworkPolicyConfig(networkPolicyInformer coreinformers.NetworkPolicyInformer, resyncPeriod time.Duration) *NetworkPolicyConfig {
	result := &NetworkPolicyConfig{
		lister:       networkPolicyInformer.Lister(),
		listerSynced: networkPolicyInformer.Informer().HasSynced,
	}

	networkPolicyInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddNetworkPolicy,
			UpdateFunc: result.handleUpdateNetworkPolicy,
			DeleteFunc: result.handleDeleteNetworkPolicy,
		},
		resyncPeriod,
	)

	return result
}

// RegisterEventHandler registers a handler which is called on every NetworkPolicy change.
func (c *NetworkPolicyConfig) RegisterEventHandler(handler NetworkPolicyHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run starts the goroutine responsible for calling
// registered handlers.
func (c *NetworkPolicyConfig) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

	glog.V(2).Info("Starting NetworkPolicy config controller")
	defer glog.V(2).Info("Shutting down NetworkPolicy config controller")

	if !waitForCacheSync("NetworkPolicy config", stopCh, c.listerSynced) {
		return
	}

	for i := range c.eventHandlers {
		glog.V(3).Infof("Calling handler.OnNetworkPolicySynced()")
		c.eventHandlers[i].OnNetworkPolicySynced()
	}

	<-stopCh
}

func (c *NetworkPolicyConfig) handleAddNetworkPolicy(obj interface{}) {
	networkPolicy, ok := obj.(*api.NetworkPolicy)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnNetworkPolicyAdd")
		c.eventHandlers[i].OnNetworkPolicyAdd(networkPolicy)
	}
}

func (c *NetworkPolicyConfig) handleUpdateNetworkPolicy(oldObj, newObj interface{}) {
	oldNetworkPolicy, ok := oldObj.(*api.NetworkPolicy)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	networkPolicy, ok := newObj.(*api.NetworkPolicy)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnNetworkPolicyUpdate")
		c.eventHandlers[i].OnNetworkPolicyUpdate(oldNetworkPolicy, networkPolicy)
	}
}

func (c *NetworkPolicyConfig) handleDeleteNetworkPolicy(obj interface{}) {
	networkPolicy, ok := obj.(*api.NetworkPolicy)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if networkPolicy, ok = tombstone.Obj.(*api.NetworkPolicy); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnNetworkPolicyDelete")
		c.eventHandlers[i].OnNetworkPolicyDelete(networkPolicy)
	}
}