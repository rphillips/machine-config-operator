package kubeletconfig

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	osev1 "github.com/openshift/api/config/v1"
)

///////////////////////////////////////////////////////////////////////////////
// Features
///////////////////////////////////////////////////////////////////////////////

const (
	clusterFeatureInstanceName = "cluster"
)

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
	rv := make(map[string]bool)
	features, err := ctrl.featLister.Get(clusterFeatureInstanceName)
	if err != nil {
		return &rv, err
	}
	set, ok := osev1.FeatureSets[features.Spec.FeatureSet]
	if !ok {
		return &rv, fmt.Errorf("Enabled FeatureSet %v does not have a corresponding config", features.Spec.FeatureSet)
	}
	for _, featEnabled := range set.Enabled {
		rv[featEnabled] = true
	}
	for _, featDisabled := range set.Disabled {
		rv[featDisabled] = false
	}
	return &rv, nil
}
