package kor

import (
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// FilterOptions represents the flags and options for filtering unused Kubernetes resources, such as pods, services, or configmaps.
// A resource is considered unused if it meets the following conditions:
//   - Its age (measured from the last modified time) is within the range specified by MinAge and MaxAge flags.
//     If MinAge or MaxAge is zero, no age limit is applied.
//   - Its size (measured in bytes) is within the range specified by MinSize and MaxSize flags.
//     If MinSize or MaxSize is zero, no size limit is applied.
//   - It does not have any labels that match the ExcludeLabels flag. The ExcludeLabels flag supports '=', '==', and '!=' operators,
//     and multiple label pairs can be separated by commas. For example, -l key1=value1,key2!=value2.
type FilterOptions struct {
	// MinAge in the minimum age of the resources to be considered unused
	MinAge time.Duration
	// MaxAge in the maximum age of the resources to be considered unused
	MaxAge time.Duration
	// MinSize is the minimum size of the resources to be considered unused
	MinSize uint64
	// MaxSize is the maximum size of the resources to be considered unused
	MaxSize uint64
	// ExcludeLabels is a label selector to exclude resources with matching labels
	ExcludeLabels string
}

// NewFilterOptions returns a new FilterOptions instance with default values
func NewFilterOptions() *FilterOptions {
	return &FilterOptions{
		MinAge:        0,
		MaxAge:        0,
		MinSize:       0,
		MaxSize:       0,
		ExcludeLabels: "",
	}
}

// Validate makes sure provided values for FilterOptions are valid
func (o *FilterOptions) Validate() error {
	if _, err := labels.Parse(o.ExcludeLabels); err != nil {
		return err
	}

	if o.MinAge < 0 {
		return errors.New("MinAge must be a non-negative duration")
	}
	if o.MaxAge < 0 {
		return errors.New("MaxAge must be a non-negative duration")
	}
	if o.MaxAge < o.MinAge {
		return errors.New("MaxAge must greater or equal than MinAge")
	}

	return nil
}

// HasExcludedLabel parses the excluded selector into a label selector object
func HasExcludedLabel(resourcelabels map[string]string, excludeSelector string) (bool, error) {
	exclude, err := labels.Parse(excludeSelector)
	if err != nil {
		return false, err
	}

	labelSet := labels.Set(resourcelabels)
	return exclude.Matches(labelSet), nil
}

// HasIncludedAge checks if a resource has an age that matches the exclude criteria specified by the filter options
// A resource is considered to have an excluded age if its age (measured from the last modified time) is within the
// range specified by MinAge and MaxAge flags.
// If MinAge or MaxAge is zero, no age limit is applied.
func HasIncludedAge(creationTime metav1.Time, opts *FilterOptions) bool {
	if opts.MinAge > 0 && opts.MaxAge > 0 {
		return (time.Since(creationTime.Time) > opts.MinAge) && (time.Since(creationTime.Time) < opts.MaxAge)
	}

	if opts.MinAge > 0 {
		return time.Since(creationTime.Time) > opts.MinAge
	}
	if opts.MaxSize > 0 {
		return time.Since(creationTime.Time) < opts.MaxAge
	}

	return false
}
