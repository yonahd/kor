package filters

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// FilterFunc is a filter that is a function
// If the resource is legal, return true
// example:
// deployment.Spec.Replicas > 0; return true
// meta.GetLabels()["kor/used"] == "true"; return true
type FilterFunc func(object runtime.Object, opts *Options) bool

// Framework is a filter framework
type Framework interface {
	// Run runs all the filters in the framework
	// If the resource is legal, return true
	Run(opts *Options, disable ...string) (bool, error)
	// AddFilter adds a filter to the framework
	AddFilter(name string, f FilterFunc) Framework
	// SetRegistry sets the registry of the framework
	SetRegistry(r Registry) Framework
	// SetObject sets the object of the framework
	SetObject(object runtime.Object) Framework
	// RunFilter runs a filter in the framework
	// name not found, return true
	RunFilter(name string, opts *Options) (bool, error)
}
