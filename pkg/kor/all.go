package kor

import (
	"encoding/json"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"os"
)

type GetUnusedResourceJSONResponse struct {
	ResourceType string              `json:"resourceType"`
	Namespaces   map[string][]string `json:"namespaces"`
}

type ResourceDiff struct {
	resourceType string
	diff         []string
}

func getUnusedCMs(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	cmDiff, err := processNamespaceCM(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "configmaps", namespace, err)
	}
	namespaceCMDiff := ResourceDiff{"ConfigMap", cmDiff}
	return namespaceCMDiff
}

func getUnusedSVCs(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	svcDiff, err := ProcessNamespaceServices(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "services", namespace, err)
	}
	namespaceSVCDiff := ResourceDiff{"Service", svcDiff}
	return namespaceSVCDiff
}

func getUnusedSecrets(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	secretDiff, err := processNamespaceSecret(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "secrets", namespace, err)
	}
	namespaceSecretDiff := ResourceDiff{"Secret", secretDiff}
	return namespaceSecretDiff
}

func getUnusedServiceAccounts(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	saDiff, err := processNamespaceSA(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "serviceaccounts", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"ServiceAccount", saDiff}
	return namespaceSADiff
}

func getUnusedDeployments(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	deployDiff, err := ProcessNamespaceDeployments(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "deployments", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"Deployment", deployDiff}
	return namespaceSADiff
}

func getUnusedStatefulsets(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	stsDiff, err := ProcessNamespaceStatefulsets(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "statefulsets", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"Statefulset", stsDiff}
	return namespaceSADiff
}

func getUnusedRoles(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	roleDiff, err := processNamespaceRoles(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "roles", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"Role", roleDiff}
	return namespaceSADiff
}

func GetUnusedAll(namespace string, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string
	var allDiffs []ResourceDiff

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)
	for _, namespace := range namespaces {
		namespaceCMDiff := getUnusedCMs(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceCMDiff)
		namespaceSVCDiff := getUnusedSVCs(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceSVCDiff)
		namespaceSecretDiff := getUnusedSecrets(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceSecretDiff)
		namespaceSADiff := getUnusedServiceAccounts(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceSADiff)
		namespaceDeploymentDiff := getUnusedDeployments(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceDeploymentDiff)
		namespaceStatefulsetDiff := getUnusedStatefulsets(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		namespaceRoleDiff := getUnusedRoles(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceRoleDiff)
		output := FormatOutputAll(namespace, allDiffs)
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedAllJSON(namespace string, kubeconfig string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)

	// Create the JSON response object
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		var allDiffs []ResourceDiff

		namespaceCMDiff := getUnusedCMs(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceCMDiff)

		namespaceSVCDiff := getUnusedSVCs(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceSVCDiff)

		namespaceSecretDiff := getUnusedSecrets(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceSecretDiff)

		namespaceSADiff := getUnusedServiceAccounts(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceSADiff)

		namespaceDeploymentDiff := getUnusedDeployments(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceDeploymentDiff)

		namespaceStatefulsetDiff := getUnusedStatefulsets(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceStatefulsetDiff)

		namespaceRoleDiff := getUnusedRoles(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceRoleDiff)

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

	return string(jsonResponse), nil
}
