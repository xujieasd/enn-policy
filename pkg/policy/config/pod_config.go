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

// PodHandler is an abstract interface of objects which receive
// notifications about Pod object changes.
type PodHandler interface {
	// OnPodAdd is called whenever creation of new Pod object
	// is observed.
	OnPodAdd(pod *api.Pod)
	// OnPodUpdate is called whenever modification of an existing
	// Pod object is observed.
	OnPodUpdate(oldPod, pod *api.Pod)
	// OnPodDelete is called whenever deletion of an existing Pod
	// object is observed.
	OnPodDelete(pod *api.Pod)
	// OnPodSynced is called once all the initial even handlers were
	// called and the state is fully propagated to local cache.
	OnPodSynced()
}

// PodConfig tracks a set of Pod configurations.
// It accepts "set", "add" and "remove" operations of Pods via channels, and invokes registered handlers on change.
type PodConfig struct {
	lister        listers.PodLister
	listerSynced  cache.InformerSynced
	eventHandlers []PodHandler
}

// NewPodConfig creates a new PodConfig.
func NewPodConfig(podInformer coreinformers.PodInformer, resyncPeriod time.Duration) *PodConfig {
	result := &PodConfig{
		lister:       podInformer.Lister(),
		listerSynced: podInformer.Informer().HasSynced,
	}

	podInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddPod,
			UpdateFunc: result.handleUpdatePod,
			DeleteFunc: result.handleDeletePod,
		},
		resyncPeriod,
	)

	return result
}

// RegisterEventHandler registers a handler which is called on every Pod change.
func (c *PodConfig) RegisterEventHandler(handler PodHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run starts the goroutine responsible for calling
// registered handlers.
func (c *PodConfig) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

	glog.V(2).Info("Starting Pod config controller")
	defer glog.V(2).Info("Shutting down Pod config controller")

	if !waitForCacheSync("Pod config", stopCh, c.listerSynced) {
		return
	}

	for i := range c.eventHandlers {
		glog.V(3).Infof("Calling handler.OnPodSynced()")
		c.eventHandlers[i].OnPodSynced()
	}

	<-stopCh
}

func (c *PodConfig) handleAddPod(obj interface{}) {
	pod, ok := obj.(*api.Pod)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnPodAdd")
		c.eventHandlers[i].OnPodAdd(pod)
	}
}

func (c *PodConfig) handleUpdatePod(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*api.Pod)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	pod, ok := newObj.(*api.Pod)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnPodUpdate")
		c.eventHandlers[i].OnPodUpdate(oldPod, pod)
	}
}

func (c *PodConfig) handleDeletePod(obj interface{}) {
	pod, ok := obj.(*api.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if pod, ok = tombstone.Obj.(*api.Pod); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	for i := range c.eventHandlers {
		glog.V(4).Infof("Calling handler.OnPodDelete")
		c.eventHandlers[i].OnPodDelete(pod)
	}
}
