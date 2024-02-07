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
)

func retrieveNoNamespaceDiff(clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, resourceList []string, filterOpts *FilterOptions) ([]ResourceDiff, []string) {
	var noNamespaceDiff []ResourceDiff
	markedForRemoval := make([]bool, len(resourceList))
	updatedResourceList := resourceList

	for counter, resource := range resourceList {
		switch resource {
		case "crd", "customresourcedefinition", "customresourcedefinitions":
			crdDiff := getUnusedCrds(apiExtClient, dynamicClient)
			noNamespaceDiff = append(noNamespaceDiff, crdDiff)
			markedForRemoval[counter] = true
		case "pv", "persistentvolume", "persistentvolumes":
			pvDiff := getUnusedPvs(clientset, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, pvDiff)
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

func retrieveNamespaceDiffs(clientset kubernetes.Interface, namespace string, resourceList []string, filterOpts *FilterOptions) []ResourceDiff {
	var allDiffs []ResourceDiff
	for _, resource := range resourceList {
		var diffResult ResourceDiff
		switch resource {
		case "cm", "configmap", "configmaps":
			diffResult = getUnusedCMs(clientset, namespace, filterOpts)
		case "svc", "service", "services":
			diffResult = getUnusedSVCs(clientset, namespace, filterOpts)
		case "scrt", "secret", "secrets":
			diffResult = getUnusedSecrets(clientset, namespace, filterOpts)
		case "sa", "serviceaccount", "serviceaccounts":
			diffResult = getUnusedServiceAccounts(clientset, namespace, filterOpts)
		case "deploy", "deployment", "deployments":
			diffResult = getUnusedDeployments(clientset, namespace, filterOpts)
		case "sts", "statefulset", "statefulsets":
			diffResult = getUnusedStatefulSets(clientset, namespace, filterOpts)
		case "role", "roles":
			diffResult = getUnusedRoles(clientset, namespace, filterOpts)
		case "clusterrole", "clusterroles":
			diffResult = getUnusedClusterRoles(clientset, namespace, filterOpts)
		case "hpa", "horizontalpodautoscaler", "horizontalpodautoscalers":
			diffResult = getUnusedHpas(clientset, namespace, filterOpts)
		case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
			diffResult = getUnusedPvcs(clientset, namespace, filterOpts)
		case "ing", "ingress", "ingresses":
			diffResult = getUnusedIngresses(clientset, namespace, filterOpts)
		case "pdb", "poddisruptionbudget", "poddisruptionbudgets":
			diffResult = getUnusedPdbs(clientset, namespace, filterOpts)
		case "po", "pod", "pods":
			diffResult = getUnusedPods(clientset, namespace, filterOpts)
		case "job", "jobs":
			diffResult = getUnusedJobs(clientset, namespace, filterOpts)
		case "rs", "replicaset", "replicasets":
			diffResult = getUnusedReplicaSets(clientset, namespace, filterOpts)
		default:
			fmt.Printf("resource type %q is not supported\n", resource)
		}
		allDiffs = append(allDiffs, diffResult)
	}
	return allDiffs
}

func GetUnusedMulti(includeExcludeLists IncludeExcludeLists, resourceNames string, filterOpts *FilterOptions, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts Opts) (string, error) {
	var allDiffs []ResourceDiff
	var outputBuffer bytes.Buffer
	var unusedMulti string
	resourceList := strings.Split(resourceNames, ",")
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)
	var err error

	noNamespaceDiff, resourceList := retrieveNoNamespaceDiff(clientset, apiExtClient, dynamicClient, resourceList, filterOpts)
	if len(noNamespaceDiff) != 0 {
		for _, diff := range noNamespaceDiff {
			if len(diff.diff) != 0 {
				output := FormatOutputAll("", []ResourceDiff{diff}, opts)
				outputBuffer.WriteString(output)
				outputBuffer.WriteString("\n")

				resourceMap := make(map[string][]string)
				resourceMap[diff.resourceType] = diff.diff
				response[""] = resourceMap
			}
		}

		resourceMap := make(map[string][]string)
		for _, diff := range noNamespaceDiff {
			resourceMap[diff.resourceType] = diff.diff
		}
		response[""] = resourceMap

	}

	for _, namespace := range namespaces {
		allDiffs = retrieveNamespaceDiffs(clientset, namespace, resourceList, filterOpts)

		if opts.DeleteFlag {
			for _, diff := range allDiffs {
				if diff.diff, err = DeleteResource(diff.diff, clientset, namespace, diff.resourceType, opts.NoInteractive); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete %s %s in namespace %s: %v\n", diff.resourceType, diff.diff, namespace, err)
				}
			}

		}
		output := FormatOutputAll(namespace, allDiffs, opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			for _, diff := range allDiffs {
				resourceMap[diff.resourceType] = diff.diff
			}
			response[namespace] = resourceMap
		}

	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedMulti, err = unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedMulti, nil
}
