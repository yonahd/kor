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

func retrieveNoNamespaceDiff(apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts Opts, resourceList []string) ([]ResourceDiff, []string) {
	var noNamespaceDiff []ResourceDiff
	for counter, resource := range resourceList {
		if resource == "crd" || resource == "customresourcedefinition" || resource == "customresourcedefinitions" {
			crdDiff := getUnusedCrds(apiExtClient, dynamicClient)
			noNamespaceDiff = append(noNamespaceDiff, crdDiff)
			updatedResourceList := append(resourceList[:counter], resourceList[counter+1:]...)
			return noNamespaceDiff, updatedResourceList
		} else {
			resourceList[counter] = resource
		}
	}
	return noNamespaceDiff, resourceList

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
		case "hpa", "horizontalpodautoscaler", "horizontalpodautoscalers":
			diffResult = getUnusedHpas(clientset, namespace, filterOpts)
		case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
			diffResult = getUnusedPvcs(clientset, namespace, filterOpts)
		case "ing", "ingress", "ingresses":
			diffResult = getUnusedIngresses(clientset, namespace, filterOpts)
		case "pdb", "poddisruptionbudget", "poddisruptionbudgets":
			diffResult = getUnusedPdbs(clientset, namespace, filterOpts)
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

	crdDiff, resourceList := retrieveNoNamespaceDiff(apiExtClient, dynamicClient, outputFormat, opts, resourceList)
	if len(crdDiff) != 0 {
		output := FormatOutputAll("", crdDiff, opts)
		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")

		resourceMap := make(map[string][]string)
		for _, diff := range crdDiff {
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
