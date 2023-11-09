package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func retrieveNamespaceDiffs(clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, namespace string, resourceList []string, filterOpts *FilterOptions) []ResourceDiff {
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
		case "crd", "customresourcedefinition", "customresourcedefinitions":
			namespaceCrdDiff := getUnusedCrds(apiExtClient, dynamicClient)
			allDiffs = append(allDiffs, namespaceCrdDiff)
		default:
			fmt.Printf("resource type %q is not supported\n", resource)
		}
	}
	return allDiffs
}

func GetUnusedMulti(includeExcludeLists IncludeExcludeLists, filterOpts *FilterOptions, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, resourceNames, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	resourceList := strings.Split(resourceNames, ",")

	for _, namespace := range namespaces {
		allDiffs := retrieveNamespaceDiffs(clientset, apiExtClient, dynamicClient, namespace, resourceList, filterOpts)

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

	unusedMulti, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedMulti, nil
}
