package kor

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

var exceptionServiceAccounts = []ExceptionResource{
	{ResourceName: "default", Namespace: "*"},
}

func retrieveUsedSA(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {

	podServiceAccounts := []string{}

	// Retrieve pods in the specified namespace
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
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

func retrieveServiceAccountNames(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	serviceaccounts, err := kubeClient.CoreV1().ServiceAccounts(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(serviceaccounts.Items))
	for _, serviceaccount := range serviceaccounts.Items {
		names = append(names, serviceaccount.Name)
	}
	return names, nil
}

func processNamespaceSA(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	usedServiceAccounts, err := retrieveUsedSA(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	usedServiceAccounts = RemoveDuplicatesAndSort(usedServiceAccounts)

	serviceAccountNames, err := retrieveServiceAccountNames(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	diff := calculateCMDifference(usedServiceAccounts, serviceAccountNames)
	return diff, nil

}

func GetUnusedServiceAccounts(namespace string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient()

	namespaces = SetNamespaceList(namespace, kubeClient)

	for _, namespace := range namespaces {
		diff, err := processNamespaceSA(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "ServiceAccount")
		fmt.Println(output)
		fmt.Println()
	}
}
