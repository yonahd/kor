package filters

import (
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	LabelFilterName    = "label"
	AgeFilterName      = "age"
	KorLabelFilterName = "korlabel"
)

// KorLabelFilter is a filter that filters out resources that are ["kor/used"] != "true"
func KorLabelFilter(object runtime.Object, opts *Options) bool {
	if meta, ok := object.(metav1.Object); ok {
		if meta.GetLabels()["kor/used"] == "true" {
			return true
		}
	}
	return false
}

// LabelFilter is a filter that filters out resources by label
func LabelFilter(object runtime.Object, opts *Options) bool {
	if meta, ok := object.(metav1.Object); ok {
		if has, err := HasExcludedLabel(meta.GetLabels(), opts.ExcludeLabels); err == nil {
			return has
		}
	}
	return false
}

// AgeFilter is a filter that filters out resources by age
func AgeFilter(object runtime.Object, opts *Options) bool {
	if meta, ok := object.(metav1.Object); ok {
		if has, err := HasIncludedAge(meta.GetCreationTimestamp(), opts); err == nil {
			return !has
		}
	}
	return false
}

// HasExcludedLabel parses the excluded selector into a label selector object
func HasExcludedLabel(resourcelabels map[string]string, excludeSelector []string) (bool, error) {
	excludes := make([]labels.Selector, 0)

	if len(excludeSelector) == 0 {
		return false, nil
	}

	for _, labelStr := range excludeSelector {
		exclude, err := labels.Parse(labelStr)
		if err != nil {
			return false, err
		}
		excludes = append(excludes, exclude)
	}

	labelSet := labels.Set(resourcelabels)
	for _, exclude := range excludes {
		if exclude.Matches(labelSet) {
			return true, nil
		}
	}
	return false, nil
}

// HasIncludedAge checks if a resource has an age that matches the included criteria specified by the filter options
// A resource is considered to have an included age if its age (measured from the last modified time) is within the
// range specified by older-than and newer-than flags.
// If older-than or newer-than is zero, no age limit is applied.
// If both flags are set, an error is returned.
func HasIncludedAge(creationTime metav1.Time, filterOpts *Options) (bool, error) {
	if filterOpts.OlderThan == "" && filterOpts.NewerThan == "" {
		return true, nil
	}
	// The function returns an error if both flags are set is because it does not make sense to
	// query for resources that are both older than and newer than a certain duration.
	// For example, if you set --older-than=1h and --newer-than=30m, you are asking for resources
	// that are older than 1 hour and newer than 30 minutes, which is impossible!
	if filterOpts.OlderThan != "" && filterOpts.NewerThan != "" {
		return false, errors.New("invalid flags: older-than and newer-than cannot be used together")
	}

	// Parse the older-than flag value into a time.Duration value
	if filterOpts.OlderThan != "" {
		olderThan, err := time.ParseDuration(filterOpts.OlderThan)
		if err != nil {
			return false, err
		}
		return time.Since(creationTime.Time) > olderThan, nil
	}

	// Parse the newer-than flag value into a time.Duration value
	if filterOpts.NewerThan != "" {
		newerThan, err := time.ParseDuration(filterOpts.NewerThan)
		if err != nil {
			return false, err
		}
		return time.Since(creationTime.Time) < newerThan, nil
	}

	return true, nil
}
