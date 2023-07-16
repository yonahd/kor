package kor

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

var exceptionServiceAccounts = []ExceptionResource{
	{ResourceName: "default", Namespace: "*"},
}

func retrieveUsedSA(clientset *kubernetes.Clientset, namespace string) ([]string, error) {

	podServiceAccounts := []string{}

	// Retrieve pods in the specified namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Extract service account names from pods
	for _, pod := range pods.Items {
		if pod.Spec.ServiceAccountName != "" {
			podServiceAccounts = append(podServiceAccounts, pod.Spec.ServiceAccountName)
		}
	}

	for _, resource := range exceptionServiceAccounts {
		if resource.Namespace == namespace || resource.Namespace == "*" {
			podServiceAccounts = append(podServiceAccounts, resource.ResourceName)
		}
	}

	return podServiceAccounts, nil
}

func retrieveServiceAccountNames(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	serviceaccounts, err := clientset.CoreV1().ServiceAccounts(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(serviceaccounts.Items))
	for _, serviceaccount := range serviceaccounts.Items {
		names = append(names, serviceaccount.Name)
	}
	return names, nil
}

func processNamespaceSA(clientset *kubernetes.Clientset, namespace string) (string, error) {
	usedServiceAccounts, err := retrieveUsedSA(clientset, namespace)
	if err != nil {
		return "", err
	}

	usedServiceAccounts = RemoveDuplicatesAndSort(usedServiceAccounts)

	serviceAccountNames, err := retrieveServiceAccountNames(clientset, namespace)
	if err != nil {
		return "", err
	}

	diff := calculateCMDifference(usedServiceAccounts, serviceAccountNames)
	return FormatOutput(namespace, diff, "Service Account"), nil

}

func GetUnusedServiceAccounts(namespace string) {
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
		output, err := processNamespaceSA(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		fmt.Println(output)
		fmt.Println()
	}
}
