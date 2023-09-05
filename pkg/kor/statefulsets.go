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

func getStatefulsetsWithoutReplicas(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	statefulsetsList, err := kubeClient.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var statefulsetsWithoutReplicas []string

	for _, statefulset := range statefulsetsList.Items {
		if *statefulset.Spec.Replicas == 0 {
			statefulsetsWithoutReplicas = append(statefulsetsWithoutReplicas, statefulset.Name)
		}
	}

	return statefulsetsWithoutReplicas, nil
}

func ProcessNamespaceStatefulsets(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	usedServices, err := getStatefulsetsWithoutReplicas(clientset, namespace)
	if err != nil {
		return nil, err
	}

	return usedServices, nil

}

func GetUnusedStatefulsets(includeExcludeLists IncludeExcludeLists, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(includeExcludeLists, kubeClient)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceStatefulsets(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Statefulsets")
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedStatefulsetsStructured(includeExcludeLists IncludeExcludeLists, kubeconfig string, outputFormat string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)
	namespaces = SetNamespaceList(includeExcludeLists, kubeClient)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceStatefulsets(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		resourceMap := make(map[string][]string)
		resourceMap["Statefulsets"] = diff
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
