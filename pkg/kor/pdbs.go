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

func processNamespacePdbs(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	var unusedPdbs []string
	pdbs, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pdb := range pdbs.Items {
		selector := pdb.Spec.Selector
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(selector),
		})
		if err != nil {
			return nil, err
		}
		statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(selector),
		})
		if err != nil {
			return nil, err
		}
		if len(deployments.Items) == 0 && len(statefulsets.Items) == 0 {
			unusedPdbs = append(unusedPdbs, pdb.Name)
		}
	}
	return unusedPdbs, nil
}

func GetUnusedPdbs(namespace string, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)

	for _, namespace := range namespaces {
		diff, err := processNamespacePdbs(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Pdbs")
		fmt.Println(output)
		fmt.Println()
	}

}

func GetUnusedPdbsJson(namespace string, kubeconfig string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespacePdbs(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if len(diff) > 0 {
			if response[namespace] == nil {
				response[namespace] = make(map[string][]string)
			}
			response[namespace]["Pdb"] = diff
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	log.Println(string(jsonResponse))
	return string(jsonResponse), nil
}
