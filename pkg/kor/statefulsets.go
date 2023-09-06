package kor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

func GetUnusedStatefulsets(namespace string, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)

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

func GetUnusedStatefulsetsSlack(namespace string, kubeconfig string, slackWebhookURL string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)

	payload := ""

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceStatefulsets(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Statefulsets")

		payload += output + "\n"
	}

	if err := sendToSlack(slackWebhookURL, payload); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send payload to Slack: %v\n", err)
	}
}

func GetUnusedStatefulsetsJSON(namespace string, kubeconfig string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)
	namespaces = SetNamespaceList(namespace, kubeClient)
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

	return string(jsonResponse), nil
}
