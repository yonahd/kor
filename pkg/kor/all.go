package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
)

type GetUnusedResourceJSONResponse struct {
	ResourceType string              `json:"resourceType"`
	Namespaces   map[string][]string `json:"namespaces"`
}

type ResourceDiff struct {
	resourceType string
	diff         []string
}

func getUnusedCMs(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	cmDiff, err := processNamespaceCM(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "configmaps", namespace, err)
	}
	namespaceCMDiff := ResourceDiff{"ConfigMap", cmDiff}
	return namespaceCMDiff
}

func getUnusedSVCs(clientset kubernetes.Interface, namespace string) ResourceDiff {
	svcDiff, err := ProcessNamespaceServices(clientset, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "services", namespace, err)
	}
	namespaceSVCDiff := ResourceDiff{"Service", svcDiff}
	return namespaceSVCDiff
}

func getUnusedSecrets(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	secretDiff, err := processNamespaceSecret(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "secrets", namespace, err)
	}
	namespaceSecretDiff := ResourceDiff{"Secret", secretDiff}
	return namespaceSecretDiff
}

func getUnusedServiceAccounts(clientset kubernetes.Interface, namespace string) ResourceDiff {
	saDiff, err := processNamespaceSA(clientset, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "serviceaccounts", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"ServiceAccount", saDiff}
	return namespaceSADiff
}

func getUnusedDeployments(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	deployDiff, err := ProcessNamespaceDeployments(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "deployments", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"Deployment", deployDiff}
	return namespaceSADiff
}

func getUnusedStatefulSets(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	stsDiff, err := ProcessNamespaceStatefulSets(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "statefulSets", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"StatefulSet", stsDiff}
	return namespaceSADiff
}

func getUnusedRoles(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	roleDiff, err := processNamespaceRoles(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "roles", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"Role", roleDiff}
	return namespaceSADiff
}

func getUnusedHpas(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	hpaDiff, err := processNamespaceHpas(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "hpas", namespace, err)
	}
	namespaceHpaDiff := ResourceDiff{"Hpa", hpaDiff}
	return namespaceHpaDiff
}

func getUnusedPvcs(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	pvcDiff, err := processNamespacePvcs(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pvcs", namespace, err)
	}
	namespacePvcDiff := ResourceDiff{"Pvc", pvcDiff}
	return namespacePvcDiff
}

func getUnusedIngresses(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	ingressDiff, err := processNamespaceIngresses(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "ingresses", namespace, err)
	}
	namespaceIngressDiff := ResourceDiff{"Ingress", ingressDiff}
	return namespaceIngressDiff
}

func getUnusedPdbs(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ResourceDiff {
	pdbDiff, err := processNamespacePdbs(clientset, namespace, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pdbs", namespace, err)
	}
	namespacePdbDiff := ResourceDiff{"Pdb", pdbDiff}
	return namespacePdbDiff
}

func GetUnusedAll(includeExcludeLists IncludeExcludeLists, opts *FilterOptions, clientset kubernetes.Interface, outputFormat string, slackOpts SlackOpts) (string, error) {
	var outputBuffer bytes.Buffer

	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		var allDiffs []ResourceDiff
		namespaceCMDiff := getUnusedCMs(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespaceCMDiff)
		namespaceSVCDiff := getUnusedSVCs(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSVCDiff)
		namespaceSecretDiff := getUnusedSecrets(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespaceSecretDiff)
		namespaceSADiff := getUnusedServiceAccounts(clientset, namespace)
		allDiffs = append(allDiffs, namespaceSADiff)
		namespaceDeploymentDiff := getUnusedDeployments(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespaceDeploymentDiff)
		namespaceStatefulsetDiff := getUnusedStatefulSets(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		namespaceRoleDiff := getUnusedRoles(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespaceRoleDiff)
		namespaceHpaDiff := getUnusedHpas(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespaceHpaDiff)
		namespacePvcDiff := getUnusedPvcs(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespacePvcDiff)
		namespaceIngressDiff := getUnusedIngresses(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespaceIngressDiff)
		namespacePdbDiff := getUnusedPdbs(clientset, namespace, opts)
		allDiffs = append(allDiffs, namespacePdbDiff)

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

	unusedAll, err := unusedResourceFormatter(outputFormat, outputBuffer, slackOpts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedAll, nil
}
