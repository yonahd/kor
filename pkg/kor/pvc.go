package kor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func retreiveUsedPvcs(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list Pods: %v\n", err)
		os.Exit(1)
	}
	var usedPvcs []string
	// Iterate through each Pod and check for PVC usage
	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				usedPvcs = append(usedPvcs, volume.PersistentVolumeClaim.ClaimName)
			}
		}
	}
	return usedPvcs, err
}

func processNamespacePvcs(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pvcNames := make([]string, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		pvcNames = append(pvcNames, pvc.Name)
	}

	usedPvcs, err := retreiveUsedPvcs(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedPvcs, pvcNames)
	return diff, nil
}

func GetUnusedPvcs(namespace string, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)

	for _, namespace := range namespaces {
		diff, err := processNamespacePvcs(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Pvcs")
		fmt.Println(output)
		fmt.Println()
	}

}

func GetUnusedPvcsSlack(namespace string, kubeconfig string, slackWebhookURL string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)

	payload := ""

	for _, namespace := range namespaces {
		diff, err := processNamespacePvcs(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Pvcs")

		payload += output + "\n"
	}

	if err := sendToSlack(slackWebhookURL, payload); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send payload to Slack: %v\n", err)
	}
}

func GetUnusedPvcsJson(namespace string, kubeconfig string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespacePvcs(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if len(diff) > 0 {
			if response[namespace] == nil {
				response[namespace] = make(map[string][]string)
			}
			response[namespace]["Pvc"] = diff
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	log.Println(string(jsonResponse))
	return string(jsonResponse), nil
}
