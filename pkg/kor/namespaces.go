package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"sigs.k8s.io/yaml"
)

type processFn func(kubernetes.Interface, string) ([]string, error)

func GetUnusedNamespaces(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, slackOpts SlackOpts) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)

	var outputBuffer bytes.Buffer

	for _, namespace := range namespaces {
		diff, err := processNamespaceNS(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Namespaces")

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

func GetUnusedNamespacesStructured(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string) (string, error) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceNS(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		resourceMap := make(map[string][]string)
		resourceMap["Namespaces"] = diff
		response[namespace] = resourceMap
	}

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

func processNamespaceNS(clientset kubernetes.Interface, namespace string) ([]string, error) {
	usedNamespace, err := retrieveUsedNS(clientset, namespace)
	if err != nil {
		return nil, err
	}
	diff := CalculateResourceDifference(usedNamespace, []string{namespace})
	return diff, nil
}

func retrieveUsedNS(clientset kubernetes.Interface, namespace string) ([]string, error) {
	processFunctions := []processFn{
		processNamespaceCM,
		processNamespaceHpas,
		processNamespaceIngresses,
		processNamespacePdbs,
		processNamespaceSecret,
		ProcessNamespaceServices,
		processNamespaceSA,
		ProcessNamespaceDeployments,
		processNamespacePvcs,
		processNamespaceRoles,
		ProcessNamespaceStatefulSets,
	}
	for _, fn := range processFunctions {
		usedResources, err := fn(clientset, namespace)
		if err != nil {
			return nil, err
		}
		if len(usedResources) > 0 {
			return []string{namespace}, nil
		}
	}

	ns, err := clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if ns.Labels["kor/used"] == "true" {
		return []string{namespace}, nil
	}

	return []string{}, nil
}
