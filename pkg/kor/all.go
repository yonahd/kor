package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
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

func getUnusedHpas(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	hpaDiff, err := processNamespaceHpas(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "hpas", namespace, err)
	}
	namespaceHpaDiff := ResourceDiff{"Hpa", hpaDiff}
	return namespaceHpaDiff
}

func getUnusedPvcs(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	pvcDiff, err := processNamespacePvcs(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pvcs", namespace, err)
	}
	namespacePvcDiff := ResourceDiff{"Pvc", pvcDiff}
	return namespacePvcDiff
}

func getUnusedIngresses(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	ingressDiff, err := processNamespaceIngresses(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "ingresses", namespace, err)
	}
	namespaceIngressDiff := ResourceDiff{"Ingress", ingressDiff}
	return namespaceIngressDiff
}

func getUnusedPdbs(kubeClient *kubernetes.Clientset, namespace string) ResourceDiff {
	pdbDiff, err := processNamespacePdbs(kubeClient, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pdbs", namespace, err)
	}
	namespacePdbDiff := ResourceDiff{"Pdb", pdbDiff}
	return namespacePdbDiff
}

func GetUnusedAll(includeExcludeLists IncludeExcludeLists, clientset *kubernetes.Clientset) {
	var namespaces []string

	namespaces = SetNamespaceList(includeExcludeLists, clientset)
	for _, namespace := range namespaces {
		var allDiffs []ResourceDiff
		namespaceCMDiff := getUnusedCMs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceCMDiff)
		namespaceSVCDiff := getUnusedSVCs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSVCDiff)
		namespaceSecretDiff := getUnusedSecrets(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSecretDiff)
		namespaceSADiff := getUnusedServiceAccounts(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSADiff)
		namespaceDeploymentDiff := getUnusedDeployments(clientset, namespace)
		allDiffs = append(allDiffs, namespaceDeploymentDiff)
		namespaceStatefulsetDiff := getUnusedStatefulsets(clientset, namespace)
		allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		namespaceRoleDiff := getUnusedRoles(clientset, namespace)
		allDiffs = append(allDiffs, namespaceRoleDiff)
		namespaceHpaDiff := getUnusedHpas(clientset, namespace)
		allDiffs = append(allDiffs, namespaceHpaDiff)
		namespacePvcDiff := getUnusedPvcs(clientset, namespace)
		allDiffs = append(allDiffs, namespacePvcDiff)
		namespaceIngressDiff := getUnusedIngresses(clientset, namespace)
		allDiffs = append(allDiffs, namespaceIngressDiff)
		namespacePdbDiff := getUnusedPdbs(clientset, namespace)
		allDiffs = append(allDiffs, namespacePdbDiff)
		output := FormatOutputAll(namespace, allDiffs)
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedAllSendToSlackWebhook(includeExcludeLists IncludeExcludeLists, clientset *kubernetes.Clientset, slackWebhookURL string) {
	var namespaces []string

	namespaces = SetNamespaceList(includeExcludeLists, clientset)

	var outputBuffer bytes.Buffer

	for _, namespace := range namespaces {
		var allDiffs []ResourceDiff
		namespaceCMDiff := getUnusedCMs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceCMDiff)
		namespaceSVCDiff := getUnusedSVCs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSVCDiff)
		namespaceSecretDiff := getUnusedSecrets(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSecretDiff)
		namespaceSADiff := getUnusedServiceAccounts(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSADiff)
		namespaceDeploymentDiff := getUnusedDeployments(clientset, namespace)
		allDiffs = append(allDiffs, namespaceDeploymentDiff)
		namespaceStatefulsetDiff := getUnusedStatefulsets(clientset, namespace)
		allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		namespaceRoleDiff := getUnusedRoles(clientset, namespace)
		allDiffs = append(allDiffs, namespaceRoleDiff)
		namespaceHpaDiff := getUnusedHpas(clientset, namespace)
		allDiffs = append(allDiffs, namespaceHpaDiff)
		namespacePvcDiff := getUnusedPvcs(clientset, namespace)
		allDiffs = append(allDiffs, namespacePvcDiff)
		namespaceIngressDiff := getUnusedIngresses(clientset, namespace)
		allDiffs = append(allDiffs, namespaceIngressDiff)
		output := FormatOutputAll(namespace, allDiffs)

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
	}

	if err := SendToSlackWebhook(slackWebhookURL, outputBuffer.String()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send payload to Slack: %v\n", err)
	}
}

func GetUnusedAllSendToSlackAsFile(includeExcludeLists IncludeExcludeLists, clientset *kubernetes.Clientset, slackChannel string, slackAuthToken string) {
	var namespaces []string

	namespaces = SetNamespaceList(includeExcludeLists, clientset)

	var outputBuffer bytes.Buffer

	for _, namespace := range namespaces {
		var allDiffs []ResourceDiff
		namespaceCMDiff := getUnusedCMs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceCMDiff)
		namespaceSVCDiff := getUnusedSVCs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSVCDiff)
		namespaceSecretDiff := getUnusedSecrets(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSecretDiff)
		namespaceSADiff := getUnusedServiceAccounts(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSADiff)
		namespaceDeploymentDiff := getUnusedDeployments(clientset, namespace)
		allDiffs = append(allDiffs, namespaceDeploymentDiff)
		namespaceStatefulsetDiff := getUnusedStatefulsets(clientset, namespace)
		allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		namespaceRoleDiff := getUnusedRoles(clientset, namespace)
		allDiffs = append(allDiffs, namespaceRoleDiff)
		namespaceHpaDiff := getUnusedHpas(clientset, namespace)
		allDiffs = append(allDiffs, namespaceHpaDiff)
		namespacePvcDiff := getUnusedPvcs(clientset, namespace)
		allDiffs = append(allDiffs, namespacePvcDiff)
		namespaceIngressDiff := getUnusedIngresses(clientset, namespace)
		allDiffs = append(allDiffs, namespaceIngressDiff)
		output := FormatOutputAll(namespace, allDiffs)

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
	}

	outputFilePath, _ := writeOutputToFile(outputBuffer)

	if err := SendFileToSlack(outputFilePath, "Unused All", slackChannel, slackAuthToken); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send output to Slack: %v\n", err)
	}
}

func GetUnusedAllStructured(includeExcludeLists IncludeExcludeLists, clientset *kubernetes.Clientset, outputFormat string) (string, error) {
	var namespaces []string

	namespaces = SetNamespaceList(includeExcludeLists, clientset)

	// Create the JSON response object
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		var allDiffs []ResourceDiff

		namespaceCMDiff := getUnusedCMs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceCMDiff)

		namespaceSVCDiff := getUnusedSVCs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSVCDiff)

		namespaceSecretDiff := getUnusedSecrets(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSecretDiff)

		namespaceSADiff := getUnusedServiceAccounts(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSADiff)

		namespaceDeploymentDiff := getUnusedDeployments(clientset, namespace)
		allDiffs = append(allDiffs, namespaceDeploymentDiff)

		namespaceStatefulsetDiff := getUnusedStatefulsets(clientset, namespace)
		allDiffs = append(allDiffs, namespaceStatefulsetDiff)

		namespaceRoleDiff := getUnusedRoles(clientset, namespace)
		allDiffs = append(allDiffs, namespaceRoleDiff)

		namespaceHpaDiff := getUnusedHpas(clientset, namespace)
		allDiffs = append(allDiffs, namespaceHpaDiff)

		namespacePvcDiff := getUnusedPvcs(clientset, namespace)
		allDiffs = append(allDiffs, namespacePvcDiff)

		namespaceIngressDiff := getUnusedIngresses(clientset, namespace)
		allDiffs = append(allDiffs, namespaceIngressDiff)

		namespacePdbDiff := getUnusedPdbs(clientset, namespace)
		allDiffs = append(allDiffs, namespacePdbDiff)

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
