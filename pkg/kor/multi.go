package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func getCanonicalResourceType(resourceName string) string {
	resourceName = strings.ToLower(resourceName)

	if _, exists := ResourceKindList[resourceName]; exists {
		return resourceName
	}

	for singular, resourceKind := range ResourceKindList {
		if resourceKind.Plural == resourceName {
			return singular
		}
		for _, shortName := range resourceKind.ShortNames {
			if shortName == resourceName {
				return singular
			}
		}
	}

	return resourceName
}

func retrieveNoNamespaceDiff(clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, resourceList []string, filterOpts *filters.Options) ([]ResourceDiff, []string) {
	var noNamespaceDiff []ResourceDiff
	markedForRemoval := make([]bool, len(resourceList))
	updatedResourceList := resourceList

	for counter, resource := range resourceList {
		canonicalType := getCanonicalResourceType(resource)
		switch canonicalType {
		case "customresourcedefinition":
			crdDiff := getUnusedCrds(apiExtClient, dynamicClient, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, crdDiff)
			markedForRemoval[counter] = true
		case "persistentvolume":
			pvDiff := getUnusedPvs(clientset, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, pvDiff)
			markedForRemoval[counter] = true
		case "clusterrole":
			clusterRoleDiff := getUnusedClusterRoles(clientset, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, clusterRoleDiff)
			markedForRemoval[counter] = true
		case "storageclass":
			storageClassDiff := getUnusedStorageClasses(clientset, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, storageClassDiff)
			markedForRemoval[counter] = true
		case "volumeattachment":
			vattsDiff := getUnusedVolumeAttachments(clientset, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, vattsDiff)
			markedForRemoval[counter] = true
		}
	}

	// Remove elements marked for removal
	var clearedResourceList []string
	for i, marked := range markedForRemoval {
		if !marked {
			clearedResourceList = append(clearedResourceList, updatedResourceList[i])
		}
	}

	return noNamespaceDiff, clearedResourceList
}

func retrieveNamespaceDiffs(clientset kubernetes.Interface, namespace string, resourceList []string, filterOpts *filters.Options, opts common.Opts) []ResourceDiff {
	var allDiffs []ResourceDiff
	for _, resource := range resourceList {
		var diffResult ResourceDiff
		canonicalType := getCanonicalResourceType(resource)
		switch canonicalType {
		case "configmap":
			diffResult = getUnusedCMs(clientset, namespace, filterOpts, opts)
		case "service":
			diffResult = getUnusedSVCs(clientset, namespace, filterOpts, opts)
		case "secret":
			diffResult = getUnusedSecrets(clientset, namespace, filterOpts, opts)
		case "serviceaccount":
			diffResult = getUnusedServiceAccounts(clientset, namespace, filterOpts, opts)
		case "deployment":
			diffResult = getUnusedDeployments(clientset, namespace, filterOpts, opts)
		case "statefulset":
			diffResult = getUnusedStatefulSets(clientset, namespace, filterOpts, opts)
		case "role":
			diffResult = getUnusedRoles(clientset, namespace, filterOpts, opts)
		case "horizontalpodautoscaler":
			diffResult = getUnusedHpas(clientset, namespace, filterOpts, opts)
		case "persistentvolumeclaim":
			diffResult = getUnusedPvcs(clientset, namespace, filterOpts, opts)
		case "ingress":
			diffResult = getUnusedIngresses(clientset, namespace, filterOpts, opts)
		case "poddisruptionbudget":
			diffResult = getUnusedPdbs(clientset, namespace, filterOpts, opts)
		case "pod":
			diffResult = getUnusedPods(clientset, namespace, filterOpts, opts)
		case "job":
			diffResult = getUnusedJobs(clientset, namespace, filterOpts, opts)
		case "replicaset":
			diffResult = getUnusedReplicaSets(clientset, namespace, filterOpts, opts)
		case "daemonset":
			diffResult = getUnusedDaemonSets(clientset, namespace, filterOpts, opts)
		case "networkpolicy":
			diffResult = getUnusedNetworkPolicies(clientset, namespace, filterOpts, opts)
		case "rolebinding":
			diffResult = getUnusedRoleBindings(clientset, namespace, filterOpts, opts)
		default:
			fmt.Printf("resource type %q is not supported\n", resource)
		}
		allDiffs = append(allDiffs, diffResult)
	}
	return allDiffs
}

func GetUnusedMulti(resourceNames string, filterOpts *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	resourceList := strings.Split(resourceNames, ",")
	namespaces := filterOpts.Namespaces(clientset)
	resources := make(map[string]map[string][]ResourceInfo)
	var err error

	if opts.GroupBy == "namespace" {
		resources[""] = make(map[string][]ResourceInfo)
	}

	noNamespaceDiff, resourceList := retrieveNoNamespaceDiff(clientset, apiExtClient, dynamicClient, resourceList, filterOpts)
	if len(noNamespaceDiff) != 0 {
		for _, diff := range noNamespaceDiff {
			if len(diff.diff) != 0 {
				if opts.DeleteFlag {
					if diff.diff, err = DeleteResource(diff.diff, clientset, "", diff.resourceType, opts.NoInteractive); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to delete %s %s: %v\n", diff.resourceType, diff.diff, err)
					}
				}
				switch opts.GroupBy {
				case "namespace":
					resources[""][diff.resourceType] = diff.diff
				case "resource":
					appendResources(resources, diff.resourceType, "", diff.diff)
				}
			}
		}
	}

	for _, namespace := range namespaces {
		allDiffs := retrieveNamespaceDiffs(clientset, namespace, resourceList, filterOpts, opts)
		if opts.GroupBy == "namespace" {
			resources[namespace] = make(map[string][]ResourceInfo)
		}

		for _, diff := range allDiffs {
			if opts.DeleteFlag {
				if diff.diff, err = DeleteResource(diff.diff, clientset, namespace, diff.resourceType, opts.NoInteractive); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete %s %s in namespace %s: %v\n", diff.resourceType, diff.diff, namespace, err)
				}
			}
			switch opts.GroupBy {
			case "namespace":
				resources[namespace][diff.resourceType] = diff.diff
			case "resource":
				appendResources(resources, diff.resourceType, namespace, diff.diff)
			}
		}
	}

	var outputBuffer bytes.Buffer
	var jsonResponse []byte
	switch outputFormat {
	case "table":
		outputBuffer = FormatOutput(resources, opts)
	case "json", "yaml":
		var err error
		if jsonResponse, err = json.MarshalIndent(resources, "", "  "); err != nil {
			return "", err
		}
	}

	unusedMulti, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedMulti, nil
}
