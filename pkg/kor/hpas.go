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
	"k8s.io/utils/strings/slices"
)

func getDeploymentNames(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(deployments.Items))
	for _, deployment := range deployments.Items {
		names = append(names, deployment.Name)
	}
	return names, nil
}

func getStatefulSetNames(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(statefulsets.Items))
	for _, statefulset := range statefulsets.Items {
		names = append(names, statefulset.Name)
	}
	return names, nil
}

func extractUnusedHpas(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	deploymentNames, err := getDeploymentNames(clientset, namespace)
	if err != nil {
		return nil, err
	}
	statefulsetNames, err := getStatefulSetNames(clientset, namespace)
	if err != nil {
		return nil, err
	}
	hpas, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var diff []string
	for _, hpa := range hpas.Items {
		switch hpa.Spec.ScaleTargetRef.Kind {
		case "Deployment":
			if !slices.Contains(deploymentNames, hpa.Spec.ScaleTargetRef.Name) {
				diff = append(diff, hpa.Name)
			}
		case "StatefulSet":
			if !slices.Contains(statefulsetNames, hpa.Spec.ScaleTargetRef.Name) {
				diff = append(diff, hpa.Name)
			}
		}
	}
	return diff, nil
}

func processNamespaceHpas(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	unusedHpas, err := extractUnusedHpas(clientset, namespace)
	if err != nil {
		return nil, err
	}
	return unusedHpas, nil
}

func GetUnusedHpas(namespace string, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)

	for _, namespace := range namespaces {
		diff, err := processNamespaceHpas(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Hpas")
		fmt.Println(output)
		fmt.Println()
	}

}

func GetUnusedHpasJson(namespace string, kubeconfig string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceHpas(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if len(diff) > 0 {
			if response[namespace] == nil {
				response[namespace] = make(map[string][]string)
			}
			response[namespace]["Hpa"] = diff
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	log.Println(string(jsonResponse))
	return string(jsonResponse), nil
}
