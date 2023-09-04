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

func getEndpointsWithoutSubsets(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	endpointsList, err := kubeClient.CoreV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var endpointsWithoutSubsets []string

	for _, endpoints := range endpointsList.Items {
		if len(endpoints.Subsets) == 0 {
			endpointsWithoutSubsets = append(endpointsWithoutSubsets, endpoints.Name)
		}
	}

	return endpointsWithoutSubsets, nil
}

func ProcessNamespaceServices(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	usedServices, err := getEndpointsWithoutSubsets(clientset, namespace)
	if err != nil {
		return nil, err
	}

	return usedServices, nil

}

func GetUnusedServices(includeExcludeLists IncludeExcludeLists, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(includeExcludeLists, kubeClient)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceServices(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Services")
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedServicesJSON(includeExcludeLists IncludeExcludeLists, kubeconfig string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)
	namespaces = SetNamespaceList(includeExcludeLists, kubeClient)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceServices(kubeClient, namespace)
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

	return string(jsonResponse), nil
}

func GetUnusedServicesYAML(includeExcludeLists IncludeExcludeLists, kubeconfig string) (string, error) {
	jsonResponse, err := GetUnusedServicesJSON(includeExcludeLists, kubeconfig)
	if err != nil {
		fmt.Println(err)
	}

	yamlResponse, err := yaml.JSONToYAML([]byte(jsonResponse))
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	return (string(yamlResponse)), nil
}
