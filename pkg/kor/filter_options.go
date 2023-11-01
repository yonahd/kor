package kor

import (
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// FilterOptions represents the flags and options for filtering unused Kubernetes resources, such as pods, services, or configmaps.
// A resource is considered unused if it meets the following conditions:
//   - Its age (measured from the last modified time) is within the range specified by older-than and newer-than flags.
//     If older-than or newer-than is zero, no age limit is applied.
//     If both flags are set, an error is returned.
//   - Its size (measured in bytes) is within the range specified by MinSize and MaxSize flags.
//     If MinSize or MaxSize is zero, no size limit is applied.
//   - It does not have any labels that match the ExcludeLabels flag. The ExcludeLabels flag supports '=', '==', and '!=' operators,
//     and multiple label pairs can be separated by commas. For example, -l key1=value1,key2!=value2.
type FilterOptions struct {
	// OlderThan is the minimum age of the resources to be considered unused
	OlderThan string
	// NewerThan is the maximum age of the resources to be considered unused
	NewerThan string
	// ExcludeLabels is a label selector to exclude resources with matching labels
	ExcludeLabels string
}

// NewFilterOptions returns a new FilterOptions instance with default values
func NewFilterOptions() *FilterOptions {
	return &FilterOptions{
		OlderThan:     "",
		NewerThan:     "",
		ExcludeLabels: "",
	}
}

// Validate makes sure provided values for FilterOptions are valid
func (o *FilterOptions) Validate() error {
	if _, err := labels.Parse(o.ExcludeLabels); err != nil {
		return err
	}

	// Parse the older-than flag value into a time.Duration value
	if o.OlderThan != "" {
		olderThan, err := time.ParseDuration(o.OlderThan)
		if err != nil {
			return err
		}
		if olderThan < 0 {
			return errors.New("OlderThan must be a non-negative duration")
		}
	}

	// Parse the newer-than flag value into a time.Duration value
	if o.NewerThan != "" {
		newerThan, err := time.ParseDuration(o.NewerThan)
		if err != nil {
			return err
		}
		if newerThan < 0 {
			return errors.New("NewerThan must be a non-negative duration")
		}
	}

	return nil
}

// HasExcludedLabel parses the excluded selector into a label selector object
func HasExcludedLabel(resourcelabels map[string]string, excludeSelector string) (bool, error) {
	if excludeSelector == "" {
		return false, nil
	}
	exclude, err := labels.Parse(excludeSelector)
	if err != nil {
		return false, err
	}

	labelSet := labels.Set(resourcelabels)
	return exclude.Matches(labelSet), nil
}

// HasIncludedAge checks if a resource has an age that matches the included criteria specified by the filter options
// A resource is considered to have an included age if its age (measured from the last modified time) is within the
// range specified by older-than and newer-than flags.
// If older-than or newer-than is zero, no age limit is applied.
// If both flags are set, an error is returned.
func HasIncludedAge(creationTime metav1.Time, opts *FilterOptions) (bool, error) {
	if opts.OlderThan == "" && opts.NewerThan == "" {
		return true, nil
	}
	// The function returns an error if both flags are set is because it does not make sense to
	// query for resources that are both older than and newer than a certain duration.
	// For example, if you set --older-than=1h and --newer-than=30m, you are asking for resources
	// that are older than 1 hour and newer than 30 minutes, which is impossible!
	if opts.OlderThan != "" && opts.NewerThan != "" {
		return false, errors.New("invalid flags: older-than and newer-than cannot be used together")
	}

	// Parse the older-than flag value into a time.Duration value
	if opts.OlderThan != "" {
		olderThan, err := time.ParseDuration(opts.OlderThan)
		if err != nil {
			return false, err
		}
		return time.Since(creationTime.Time) > olderThan, nil
	}

	// Parse the newer-than flag value into a time.Duration value
	if opts.NewerThan != "" {
		newerThan, err := time.ParseDuration(opts.NewerThan)
		if err != nil {
			return false, err
		}
		return time.Since(creationTime.Time) < newerThan, nil
	}

	return true, nil
}
