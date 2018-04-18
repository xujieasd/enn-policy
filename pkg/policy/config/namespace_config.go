package config

import (
	api "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	listers "k8s.io/client-go/listers/core/v1"

	"time"
	"github.com/golang/glog"
	"fmt"
)

// NamespaceHandler is an abstract interface of objects which receive
// notifications about Namespace object changes.
type NamespaceHandler interface {
	// OnNamespaceAdd is called whenever creation of new Namespace object
	// is observed.
	OnNamespaceAdd(namespace *api.Namespace)
	// OnNamespaceUpdate is called whenever modification of an existing
	// Namespace object is observed.
	OnNamespaceUpdate(oldNamespace, namespace *api.Namespace)
	// OnNamespaceDelete is called whenever deletion of an existing Namespace
	// object is observed.
	OnNamespaceDelete(namespace *api.Namespace)
	// OnNamespaceSynced is called once all the initial even handlers were
	// called and the state is fully propagated to local cache.
	OnNamespaceSynced()
}

// NamespaceConfig tracks a set of Namespace configurations.
// It accepts "set", "add" and "remove" operations of Namespaces via channels, and invokes registered handlers on change.
type NamespaceConfig struct {
	lister        listers.NamespaceLister
	listerSynced  cache.InformerSynced
	eventHandlers []NamespaceHandler
}

// NewNamespaceConfig creates a new NamespaceConfig.
func NewNamespaceConfig(namespaceInformer coreinformers.NamespaceInformer, resyncPeriod time.Duration) *NamespaceConfig {
	result := &NamespaceConfig{
		lister:       namespaceInformer.Lister(),
		listerSynced: namespaceInformer.Informer().HasSynced,
	}

	namespaceInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddNamespace,
			UpdateFunc: result.handleUpdateNamespace,
			DeleteFunc: result.handleDeleteNamespace,
		},
		resyncPeriod,
	)

	return result
}

// RegisterEventHandler registers a handler which is called on every Namespace change.
func (c *NamespaceConfig) RegisterEventHandler(handler NamespaceHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run starts the goroutine responsible for calling
// registered handlers.
func (c *NamespaceConfig) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

	glog.V(2).Info("Starting Namespace config controller")
	defer glog.V(2).Info("Shutting down Namespace config controller")

	if !waitForCacheSync("Namespace config", stopCh, c.listerSynced) {
		return
	}

	for i := range c.eventHandlers {
		glog.V(3).Infof("Calling handler.OnNamespaceSynced()")
		c.eventHandlers[i].OnNamespaceSynced()
	}

	<-stopCh
}

func (c *NamespaceConfig) handleAddNamespace(obj interface{}) {
	namespace, ok := obj.(*api.Namespace)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnNamespaceAdd")
		c.eventHandlers[i].OnNamespaceAdd(namespace)
	}
}

func (c *NamespaceConfig) handleUpdateNamespace(oldObj, newObj interface{}) {
	oldNamespace, ok := oldObj.(*api.Namespace)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	namespace, ok := newObj.(*api.Namespace)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnNamespaceUpdate")
		c.eventHandlers[i].OnNamespaceUpdate(oldNamespace, namespace)
	}
}

func (c *NamespaceConfig) handleDeleteNamespace(obj interface{}) {
	namespace, ok := obj.(*api.Namespace)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if namespace, ok = tombstone.Obj.(*api.Namespace); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnNamespaceDelete")
		c.eventHandlers[i].OnNamespaceDelete(namespace)
	}
}
