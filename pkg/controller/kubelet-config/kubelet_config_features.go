package kubeletconfig

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	osev1 "github.com/openshift/api/config/v1"
)

///////////////////////////////////////////////////////////////////////////////
// Features
///////////////////////////////////////////////////////////////////////////////

func (ctrl *Controller) featureWorker() {
	for ctrl.processNextFeatureWorkItem() {
	}
}

func (ctrl *Controller) processNextFeatureWorkItem() bool {
	key, quit := ctrl.featureQueue.Get()
	if quit {
		return false
	}
	defer ctrl.featureQueue.Done(key)

	err := ctrl.syncFeatureHandler(key.(string))
	ctrl.handleErr(err, key)
	return true
}

func (ctrl *Controller) syncFeatureHandler(feat string) error {
	return nil
}

func (ctrl *Controller) enqueueFeature(feat *osev1.Features) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(feat)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", feat, err))
		return
	}
	ctrl.featureQueue.Add(key)
}

func (ctrl *Controller) updateFeature(old, cur interface{}) {
	oldFeature := old.(*osev1.Features)
	newFeature := cur.(*osev1.Features)
	if !reflect.DeepEqual(oldFeature.Spec, newFeature.Spec) {
		glog.V(4).Infof("Update Feature %s", newFeature.Name)
		ctrl.enqueueFeature(newFeature)
	}
}

func (ctrl *Controller) addFeature(obj interface{}) {
	features := obj.(*osev1.Features)
	glog.V(4).Infof("Adding Feature %s", features.Name)
	ctrl.enqueueFeature(features)
}

func (ctrl *Controller) deleteFeature(obj interface{}) {
	features, ok := obj.(*osev1.Features)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		features, ok = tombstone.Obj.(*osev1.Features)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a KubeletConfig %#v", obj))
			return
		}
	}
	glog.V(4).Infof("Deleted Feature %s and restored default config", features.Name)
}

func (ctrl *Controller) getFeatures() (*map[string]bool, error) {
	features, err := ctrl.featLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	rv := make(map[string]bool)
	for _, feature := range features {
		for _, featEnabled := range feature.Spec.Enabled {
			rv[featEnabled] = true
		}
		for _, featDisabled := range feature.Spec.Disabled {
			rv[featDisabled] = false
		}
	}
	return &rv, nil
}
