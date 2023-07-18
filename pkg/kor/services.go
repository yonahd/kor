package kor

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

var exceptionServices = []ExceptionResource{
	{ResourceName: "default", Namespace: "*"},
}

func GetEndpointsWithoutSubsets(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	endpointsList, err := clientset.CoreV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{})
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
	var kubeconfig string
	var namespaces []string

	kubeconfig = GetKubeConfigPath()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	if namespace != "" {
		namespaces = append(namespaces, namespace)
	} else {
		namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to retrieve namespaces: %v\n", err)
			os.Exit(1)
		}
		for _, ns := range namespaceList.Items {
			namespaces = append(namespaces, ns.Name)
		}
	}

	for _, namespace := range namespaces {
		output, err := processNamespaceSerices(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		fmt.Println(output)
		fmt.Println()
	}
}
