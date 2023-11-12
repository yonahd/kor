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
		switch resource {
		case "cm", "configmap", "configmaps":
			namespaceCMDiff := getUnusedCMs(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespaceCMDiff)
		case "svc", "service", "services":
			namespaceSVCDiff := getUnusedSVCs(clientset, namespace)
			allDiffs = append(allDiffs, namespaceSVCDiff)
		case "scrt", "secret", "secrets":
			namespaceSecretDiff := getUnusedSecrets(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespaceSecretDiff)
		case "sa", "serviceaccount", "serviceaccounts":
			namespaceSADiff := getUnusedServiceAccounts(clientset, namespace)
			allDiffs = append(allDiffs, namespaceSADiff)
		case "deploy", "deployment", "deployments":
			namespaceDeploymentDiff := getUnusedDeployments(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespaceDeploymentDiff)
		case "sts", "statefulset", "statefulsets":
			namespaceStatefulsetDiff := getUnusedStatefulSets(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		case "role", "roles":
			namespaceRoleDiff := getUnusedRoles(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespaceRoleDiff)
		case "hpa", "horizontalpodautoscaler", "horizontalpodautoscalers":
			namespaceHpaDiff := getUnusedHpas(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespaceHpaDiff)
		case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
			namespacePvcDiff := getUnusedPvcs(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespacePvcDiff)
		case "ing", "ingress", "ingresses":
			namespaceIngressDiff := getUnusedIngresses(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespaceIngressDiff)
		case "pdb", "poddisruptionbudget", "poddisruptionbudgets":
			namespacePdbDiff := getUnusedPdbs(clientset, namespace, filterOpts)
			allDiffs = append(allDiffs, namespacePdbDiff)
		default:
			fmt.Printf("resource type %q is not supported\n", resource)
		}
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
		output := FormatOutputAll("", crdDiff)
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
		output := FormatOutputAll(namespace, allDiffs)
		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")

		resourceMap := make(map[string][]string)
		for _, diff := range allDiffs {
			resourceMap[diff.resourceType] = diff.diff
		}
		response[namespace] = resourceMap
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
