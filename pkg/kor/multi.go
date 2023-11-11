package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
)

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

func GetUnusedMulti(includeExcludeLists IncludeExcludeLists, resourceNames string, filterOpts *FilterOptions, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	resourceList := strings.Split(resourceNames, ",")
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)
	var err error

	for _, namespace := range namespaces {
		allDiffs := retrieveNamespaceDiffs(clientset, namespace, resourceList, filterOpts)

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

	unusedMulti, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedMulti, nil
}
