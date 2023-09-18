package kor

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func retrieveNamespaceDiffs(kubeClient *kubernetes.Clientset, namespace string, resourceList []string) []ResourceDiff {
	var allDiffs []ResourceDiff
	for _, resource := range resourceList {
		switch resource {
		case "cm", "configmap", "configmaps":
			namespaceCMDiff := getUnusedCMs(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceCMDiff)
		case "svc", "service", "services":
			namespaceSVCDiff := getUnusedSVCs(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceSVCDiff)
		case "scrt", "secret", "secrets":
			namespaceSecretDiff := getUnusedSecrets(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceSecretDiff)
		case "sa", "serviceaccount", "serviceaccounts":
			namespaceSADiff := getUnusedServiceAccounts(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceSADiff)
		case "deploy", "deployment", "deployments":
			namespaceDeploymentDiff := getUnusedDeployments(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceDeploymentDiff)
		case "sts", "statefulset", "statefulsets":
			namespaceStatefulsetDiff := getUnusedStatefulSets(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		case "role", "roles":
			namespaceRoleDiff := getUnusedRoles(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceRoleDiff)
		case "hpa", "horizontalpodautoscaler", "horizontalpodautoscalers":
			namespaceHpaDiff := getUnusedHpas(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceHpaDiff)
		case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
			namespacePvcDiff := getUnusedPvcs(kubeClient, namespace)
			allDiffs = append(allDiffs, namespacePvcDiff)
		case "ing", "ingress", "ingresses":
			namespaceIngressDiff := getUnusedIngresses(kubeClient, namespace)
			allDiffs = append(allDiffs, namespaceIngressDiff)
		case "pdb", "poddisruptionbudget", "poddisruptionbudgets":
			namespacePdbDiff := getUnusedPdbs(kubeClient, namespace)
			allDiffs = append(allDiffs, namespacePdbDiff)
		default:
			fmt.Printf("resource type %q is not supported\n", resource)
		}
	}
	return allDiffs
}

func GetUnusedMulti(includeExcludeLists IncludeExcludeLists, kubeconfig, resourceNames string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	resourceList := strings.Split(resourceNames, ",")
	namespaces = SetNamespaceList(includeExcludeLists, kubeClient)

	for _, namespace := range namespaces {
		allDiffs := retrieveNamespaceDiffs(kubeClient, namespace, resourceList)
		output := FormatOutputAll(namespace, allDiffs)
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedMultiStructured(includeExcludeLists IncludeExcludeLists, kubeconfig, outputFormat, resourceNames string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	resourceList := strings.Split(resourceNames, ",")
	namespaces = SetNamespaceList(includeExcludeLists, kubeClient)

	// Create the JSON response object
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		allDiffs := retrieveNamespaceDiffs(kubeClient, namespace, resourceList)
		// Store the unused resources for each resource type in the JSON response
		resourceMap := make(map[string][]string)
		for _, diff := range allDiffs {
			resourceMap[diff.resourceType] = diff.diff
		}
		response[namespace] = resourceMap
	}

	// Convert the response object to JSON
	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	if outputFormat == "yaml" {
		yamlResponse, err := yaml.JSONToYAML(jsonResponse)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		return string(yamlResponse), nil
	} else {
		return string(jsonResponse), nil
	}
}
