package kor

import (
	"encoding/json"
	"regexp"
	"sort"
)

type ExceptionResource struct {
	Namespace    string
	ResourceName string
	MatchRegex   bool
}
type IncludeExcludeLists struct {
	IncludeListStr string
	ExcludeListStr string
}

type Config struct {
	ExceptionClusterRoles    []ExceptionResource `json:"exceptionClusterRoles"`
	ExceptionConfigMaps      []ExceptionResource `json:"exceptionConfigMaps"`
	ExceptionCrds            []ExceptionResource `json:"exceptionCrds"`
	ExceptionDaemonSets      []ExceptionResource `json:"exceptionDaemonSets"`
	ExceptionRoles           []ExceptionResource `json:"exceptionRoles"`
	ExceptionSecrets         []ExceptionResource `json:"exceptionSecrets"`
	ExceptionServiceAccounts []ExceptionResource `json:"exceptionServiceAccounts"`
	ExceptionServices        []ExceptionResource `json:"exceptionServices"`
	ExceptionStorageClasses  []ExceptionResource `json:"exceptionStorageClasses"`
	ExceptionJobs            []ExceptionResource `json:"exceptionJobs"`
	ExceptionPdbs            []ExceptionResource `json:"exceptionPdbs"`
	ExceptionRoleBindings    []ExceptionResource `json:"exceptionRoleBindings"`
	// Add other configurations if needed
}

func RemoveDuplicatesAndSort(slice []string) []string {
	uniqueSet := make(map[string]bool)
	for _, item := range slice {
		uniqueSet[item] = true
	}
	uniqueSlice := make([]string, 0, len(uniqueSet))
	for item := range uniqueSet {
		uniqueSlice = append(uniqueSlice, item)
	}
	sort.Strings(uniqueSlice)
	return uniqueSlice
}

// TODO create formatter by resource "#", "Resource Name", "Namespace"
// TODO Functions that use this object are accompanied by repeated data acquisition operations and can be optimized.
func CalculateResourceDifference(usedResourceNames []string, allResourceNames []string) []string {
	var difference []string
	for _, name := range allResourceNames {
		found := false
		for _, usedName := range usedResourceNames {
			if name == usedName {
				found = true
				break
			}
		}
		if !found {
			difference = append(difference, name)
		}
	}
	return difference
}

func isResourceException(resourceName, namespace string, exceptions []ExceptionResource) (bool, error) {
	var match bool
	for _, e := range exceptions {
		if e.ResourceName == resourceName && e.Namespace == namespace {
			match = true
			break
		}

		if e.MatchRegex {
			namespaceRegexp, err := regexp.Compile(e.Namespace)
			if err != nil {
				return false, err
			}
			nameRegexp, err := regexp.Compile(e.ResourceName)
			if err != nil {
				return false, err
			}
			if nameRegexp.MatchString(resourceName) && namespaceRegexp.MatchString(namespace) {
				match = true
				break
			}
		}
	}
	return match, nil
}

func unmarshalConfig(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func contains(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

func resourceInfoContains(slice []ResourceInfo, item string) bool {
	for _, element := range slice {
		if element.Name == item {
			return true
		}
	}
	return false
}

// Convert a slice of names into a map for fast lookup
func convertNamesToPresenseMap(names []string, _ []string, err error) (map[string]bool, error) {
	if err != nil {
		return nil, err
	}

	namesMap := make(map[string]bool)
	for _, n := range names {
		namesMap[n] = true
	}

	return namesMap, nil
}
