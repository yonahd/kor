package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func retrieveNamespaceDiffs(clientset kubernetes.Interface, namespace string, resourceList []string) []ResourceDiff {
	var allDiffs []ResourceDiff
	for _, resource := range resourceList {
		switch resource {
		case "cm", "configmap", "configmaps":
			namespaceCMDiff := getUnusedCMs(clientset, namespace)
			allDiffs = append(allDiffs, namespaceCMDiff)
		case "svc", "service", "services":
			namespaceSVCDiff := getUnusedSVCs(clientset, namespace)
			allDiffs = append(allDiffs, namespaceSVCDiff)
		case "scrt", "secret", "secrets":
			namespaceSecretDiff := getUnusedSecrets(clientset, namespace)
			allDiffs = append(allDiffs, namespaceSecretDiff)
		case "sa", "serviceaccount", "serviceaccounts":
			namespaceSADiff := getUnusedServiceAccounts(clientset, namespace)
			allDiffs = append(allDiffs, namespaceSADiff)
		case "deploy", "deployment", "deployments":
			namespaceDeploymentDiff := getUnusedDeployments(clientset, namespace)
			allDiffs = append(allDiffs, namespaceDeploymentDiff)
		case "sts", "statefulset", "statefulsets":
			namespaceStatefulsetDiff := getUnusedStatefulSets(clientset, namespace)
			allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		case "role", "roles":
			namespaceRoleDiff := getUnusedRoles(clientset, namespace)
			allDiffs = append(allDiffs, namespaceRoleDiff)
		case "hpa", "horizontalpodautoscaler", "horizontalpodautoscalers":
			namespaceHpaDiff := getUnusedHpas(clientset, namespace)
			allDiffs = append(allDiffs, namespaceHpaDiff)
		case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
			namespacePvcDiff := getUnusedPvcs(clientset, namespace)
			allDiffs = append(allDiffs, namespacePvcDiff)
		case "ing", "ingress", "ingresses":
			namespaceIngressDiff := getUnusedIngresses(clientset, namespace)
			allDiffs = append(allDiffs, namespaceIngressDiff)
		case "pdb", "poddisruptionbudget", "poddisruptionbudgets":
			namespacePdbDiff := getUnusedPdbs(clientset, namespace)
			allDiffs = append(allDiffs, namespacePdbDiff)
		default:
			fmt.Printf("resource type %q is not supported\n", resource)
		}
	}
	return allDiffs
}

func GetUnusedMulti(includeExcludeLists IncludeExcludeLists, kubeconfig, resourceNames string, slackOpts SlackOpts) {
	var clientset kubernetes.Interface
	var namespaces []string

	var outputBuffer bytes.Buffer

	clientset = GetKubeClient(kubeconfig)

	resourceList := strings.Split(resourceNames, ",")
	namespaces = SetNamespaceList(includeExcludeLists, clientset)

	for _, namespace := range namespaces {
		allDiffs := retrieveNamespaceDiffs(clientset, namespace, resourceList)
		output := FormatOutputAll(namespace, allDiffs)

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
	}

	if slackOpts != (SlackOpts{}) {
		if err := SendToSlack(SlackMessage{}, slackOpts, outputBuffer.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send message to slack: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(outputBuffer.String())
	}
}

func GetUnusedMultiStructured(includeExcludeLists IncludeExcludeLists, kubeconfig, outputFormat, resourceNames string) (string, error) {
	var clientset kubernetes.Interface
	var namespaces []string

	clientset = GetKubeClient(kubeconfig)

	resourceList := strings.Split(resourceNames, ",")
	namespaces = SetNamespaceList(includeExcludeLists, clientset)

	// Create the JSON response object
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		allDiffs := retrieveNamespaceDiffs(clientset, namespace, resourceList)
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
