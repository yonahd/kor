package kor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func ProcessNamespaceServices(clientset kubernetes.Interface, namespace string) ([]string, error) {
	endpointsList, err := clientset.CoreV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var endpointsWithoutSubsets []string

	for _, endpoints := range endpointsList.Items {
		if endpoints.Labels["kor/used"] == "true" {
			continue
		}

		if len(endpoints.Subsets) == 0 {
			endpointsWithoutSubsets = append(endpointsWithoutSubsets, endpoints.Name)
		}
	}

	return endpointsWithoutSubsets, nil
}

func GetUnusedServices(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceServices(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Services")
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedServicesStructured(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string) (string, error) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceServices(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		resourceMap := make(map[string][]string)
		resourceMap["Services"] = diff
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
