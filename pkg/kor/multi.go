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

func retrieveResourceDiffs(resourceList []string, diffRetriever func(resource string) ResourceDiff) []ResourceDiff {
	resourceJobs := make([]func() ResourceDiff, 0, len(resourceList))
	for _, resource := range resourceList {
		resourceName := resource
		resourceJobs = append(resourceJobs, func() ResourceDiff {
			return diffRetriever(resourceName)
		})
	}

	return runResourceDiffJobs(resourceJobs, getMaxParallelResourceWorkers())
}

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

func retrieveNoNamespaceDiff(clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, resourceList []string, filterOpts *filters.Options, opts common.Opts) ([]ResourceDiff, []string) {
	var noNamespaceJobs []func() ResourceDiff
	markedForRemoval := make([]bool, len(resourceList))
	updatedResourceList := resourceList

	for counter, resource := range resourceList {
		canonicalType := getCanonicalResourceType(resource)
		switch canonicalType {
		case "customresourcedefinition":
			noNamespaceJobs = append(noNamespaceJobs, func() ResourceDiff {
				return getUnusedCrds(apiExtClient, dynamicClient, filterOpts)
			})
			markedForRemoval[counter] = true
		case "persistentvolume":
			noNamespaceJobs = append(noNamespaceJobs, func() ResourceDiff {
				return getUnusedPvs(clientset, filterOpts)
			})
			markedForRemoval[counter] = true
		case "clusterrole":
			noNamespaceJobs = append(noNamespaceJobs, func() ResourceDiff {
				return getUnusedClusterRoles(clientset, filterOpts)
			})
			markedForRemoval[counter] = true
		case "clusterrolebinding":
			noNamespaceJobs = append(noNamespaceJobs, func() ResourceDiff {
				return getUnusedClusterRoleBindings(clientset, filterOpts, opts)
			})
			markedForRemoval[counter] = true
		case "storageclass":
			noNamespaceJobs = append(noNamespaceJobs, func() ResourceDiff {
				return getUnusedStorageClasses(clientset, filterOpts)
			})
			markedForRemoval[counter] = true
		case "volumeattachment":
			noNamespaceJobs = append(noNamespaceJobs, func() ResourceDiff {
				return getUnusedVolumeAttachments(clientset, filterOpts)
			})
			markedForRemoval[counter] = true
		case "priorityclass":
			noNamespaceJobs = append(noNamespaceJobs, func() ResourceDiff {
				return getUnusedPriorityClasses(clientset, filterOpts)
			})
			markedForRemoval[counter] = true
		}
	}

	noNamespaceDiff := runResourceDiffJobs(noNamespaceJobs, getMaxParallelResourceWorkers())

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
	return retrieveResourceDiffs(resourceList, func(resource string) ResourceDiff {
		canonicalType := getCanonicalResourceType(resource)
		switch canonicalType {
		case "configmap":
			return getUnusedCMs(clientset, namespace, filterOpts, opts)
		case "service":
			return getUnusedSVCs(clientset, namespace, filterOpts, opts)
		case "secret":
			return getUnusedSecrets(clientset, namespace, filterOpts, opts)
		case "serviceaccount":
			return getUnusedServiceAccounts(clientset, namespace, filterOpts, opts)
		case "deployment":
			return getUnusedDeployments(clientset, namespace, filterOpts, opts)
		case "statefulset":
			return getUnusedStatefulSets(clientset, namespace, filterOpts, opts)
		case "role":
			return getUnusedRoles(clientset, namespace, filterOpts, opts)
		case "horizontalpodautoscaler":
			return getUnusedHpas(clientset, namespace, filterOpts, opts)
		case "persistentvolumeclaim":
			return getUnusedPvcs(clientset, namespace, filterOpts, opts)
		case "ingress":
			return getUnusedIngresses(clientset, namespace, filterOpts, opts)
		case "poddisruptionbudget":
			return getUnusedPdbs(clientset, namespace, filterOpts, opts)
		case "pod":
			return getUnusedPods(clientset, namespace, filterOpts, opts)
		case "job":
			return getUnusedJobs(clientset, namespace, filterOpts, opts)
		case "replicaset":
			return getUnusedReplicaSets(clientset, namespace, filterOpts, opts)
		case "daemonset":
			return getUnusedDaemonSets(clientset, namespace, filterOpts, opts)
		case "networkpolicy":
			return getUnusedNetworkPolicies(clientset, namespace, filterOpts, opts)
		case "rolebinding":
			return getUnusedRoleBindings(clientset, namespace, filterOpts, opts)
		default:
			fmt.Printf("resource type %q is not supported\n", resource)
			return ResourceDiff{}
		}
	})
}

func GetUnusedMulti(resourceNames string, filterOpts *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	resourceList := strings.Split(resourceNames, ",")
	namespaces := filterOpts.Namespaces(clientset)
	resources := make(map[string]map[string][]ResourceInfo)
	var err error

	if opts.GroupBy == "namespace" {
		resources[""] = make(map[string][]ResourceInfo)
	}

	noNamespaceDiff, resourceList := retrieveNoNamespaceDiff(clientset, apiExtClient, dynamicClient, resourceList, filterOpts, opts)
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
