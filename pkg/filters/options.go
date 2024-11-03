package filters

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Options represents the flags and options for filtering unused Kubernetes resources, such as pods, services, or configmaps.
// A resource is considered unused if it meets the following conditions:
//   - Its age (measured from the last modified time) is within the range specified by older-than and newer-than flags.
//     If older-than or newer-than is zero, no age limit is applied.
//     If both flags are set, an error is returned.
//   - Its size (measured in bytes) is within the range specified by MinSize and MaxSize flags.
//     If MinSize or MaxSize is zero, no size limit is applied.
//   - It does not have any labels that match the ExcludeLabels flag. The ExcludeLabels flag supports '=', '==', and '!=' operators,
//     and multiple label pairs can be separated by commas. For example, -l key1=value1,key2!=value2.
type Options struct {
	// OlderThan is the minimum age of the resources to be considered unused
	OlderThan string
	// NewerThan is the maximum age of the resources to be considered unused
	NewerThan string
	// ExcludeLabels is a label selector to exclude resources with matching labels
	// IncludeLabels conflicts with it, and when setting IncludeLabels, ExcludeLabels is ignored and set to empty
	ExcludeLabels []string
	// IncludeLabels is a label selector to include resources with matching labels
	IncludeLabels string
	// ExcludeNamespaces is a namespace selector to exclude resources in matching namespaces
	// IncludeNamespaces conflicts with it, and when setting IncludeNamespaces, ExcludeNamespaces is ignored and set to empty
	ExcludeNamespaces []string
	// IncludeNamespaces is a namespace selector to include resources in matching namespaces
	IncludeNamespaces []string

	namespace             []string
	once                  sync.Once
	IncludeThirdPartyCrds []string
}

// NewFilterOptions returns a new FilterOptions instance with default values
func NewFilterOptions() *Options {
	return &Options{
		OlderThan: "",
		NewerThan: "",
	}
}

func parseLabels(labelsStr string) (labels.Set, error) {
	labelMap := map[string]string{}

	labelPairs := strings.Split(labelsStr, ",")

	for _, pair := range labelPairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format: %s", pair)
		}
		labelMap[parts[0]] = parts[1]
	}

	return labels.Set(labelMap), nil
}

// Validate makes sure provided values for FilterOptions are valid
func (o *Options) Validate() error {

	// Parse and validate the labels
	for _, labelStr := range o.ExcludeLabels {
		if _, err := parseLabels(labelStr); err != nil {
			return err
		}
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

// Modify modifies the options
func (o *Options) Modify() {
	o.modifyLabels()
}

// Namespaces returns the namespaces, only called once
func (o *Options) Namespaces(clientset kubernetes.Interface) []string {
	o.once.Do(func() {
		namespaces := make([]string, 0)
		namespacesMap := make(map[string]bool)
		if len(o.IncludeNamespaces) > 0 && len(o.ExcludeNamespaces) > 0 {
			fmt.Fprintf(os.Stderr, "Exclude namespaces can't be used together with include namespaces. Ignoring --exclude-namespaces (-e) flag\n")
			o.ExcludeNamespaces = nil
		}
		includeNamespaces := o.IncludeNamespaces
		excludeNamespaces := o.ExcludeNamespaces

		if len(o.IncludeNamespaces) > 0 {

			for _, ns := range includeNamespaces {

				_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
				if err == nil {
					namespacesMap[ns] = true
				} else {
					fmt.Fprintf(os.Stderr, "namespace [%s] not found\n", ns)
				}
			}
		} else {
			namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to retrieve namespaces: %v\n", err)
				return
			}

			for _, ns := range namespaceList.Items {
				namespacesMap[ns.Name] = false
			}

			for _, ns := range namespaceList.Items {
				namespacesMap[ns.Name] = true
			}
			for _, ns := range excludeNamespaces {
				if _, exists := namespacesMap[ns]; exists {
					namespacesMap[ns] = false
				}
			}
		}
		for ns := range namespacesMap {
			if namespacesMap[ns] {
				namespaces = append(namespaces, ns)
			}
		}
		o.namespace = namespaces
	})
	return o.namespace
}

func (o *Options) CleanRepeatedCrds() []string {
	if len(o.IncludeThirdPartyCrds) > 0 {
		keys := make(map[string]bool)
		includecrdsNew := make([]string, 0)

		for _, entry := range o.IncludeThirdPartyCrds {
			if _, value := keys[entry]; !value {
				keys[entry] = true
				includecrdsNew = append(includecrdsNew, entry)
			}
			o.IncludeThirdPartyCrds = includecrdsNew
		}
	}
	return o.IncludeThirdPartyCrds
}

func (o *Options) modifyLabels() {
	if o.IncludeLabels != "" {
		if len(o.ExcludeLabels) > 0 {
			fmt.Fprintf(os.Stderr, "Exclude labels can't be used together with include labels. Ignoring --exclude-labels (-l) flag\n")
		}
		o.ExcludeLabels = nil
	}
}
