package kor

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

func getDeploymentsWithoutReplicas(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	deploymentsList, err := kubeClient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var deploymentsWithoutReplicas []string

	for _, deployment := range deploymentsList.Items {
		if *deployment.Spec.Replicas == 0 {
			deploymentsWithoutReplicas = append(deploymentsWithoutReplicas, deployment.Name)
		}
	}

	return deploymentsWithoutReplicas, nil
}

func ProcessNamespaceDeployments(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	usedServices, err := getDeploymentsWithoutReplicas(clientset, namespace)
	if err != nil {
		return nil, err
	}

	return usedServices, nil

}

func GetUnusedDeployments(namespace string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient()

	namespaces = SetNamespaceList(namespace, kubeClient)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceDeployments(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Deployments")
		fmt.Println(output)
		fmt.Println()
	}
}
