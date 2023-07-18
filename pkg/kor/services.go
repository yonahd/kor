package kor

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

var exceptionServices = []ExceptionResource{
	{ResourceName: "default", Namespace: "*"},
}

func GetEndpointsWithoutSubsets(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
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

func processNamespaceSerices(clientset *kubernetes.Clientset, namespace string) (string, error) {
	usedServices, err := GetEndpointsWithoutSubsets(clientset, namespace)
	if err != nil {
		return "", err
	}

	return FormatOutput(namespace, usedServices, "Services"), nil

}

func GetUnusedServices(namespace string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient()

	namespaces = SetNamespaceList(namespace, kubeClient)

	for _, namespace := range namespaces {
		output, err := processNamespaceSerices(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		fmt.Println(output)
		fmt.Println()
	}
}
