package kor

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func DeleteResourceCmd() map[string]func(clientset kubernetes.Interface, namespace, name string) error {
	var deleteResourceApiMap = map[string]func(clientset kubernetes.Interface, namespace, name string) error{
		"ConfigMap": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Secret": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
	}

	return deleteResourceApiMap
}

func DeleteResource(diff []string, clientset kubernetes.Interface, namespace, resourceType string) error {
	for _, cm := range diff {
		deleteFunc, exists := DeleteResourceCmd()[resourceType]
		if !exists {
			fmt.Printf("Resource type '%s' is not supported\n", cm)
			continue
		}

		fmt.Printf("Do you want to delete %s %s in namespace %s? (Y/N): ", resourceType, cm, namespace)
		var confirmation string
		_, err := fmt.Scanf("%s", &confirmation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
			continue
		}

		if confirmation == "y" || confirmation == "Y" || confirmation == "yes" {
			fmt.Printf("Deleting %s %s in namespace %s\n", resourceType, cm, namespace)
			if err := deleteFunc(clientset, namespace, cm); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete %s %s in namespace %s: %v\n", resourceType, cm, namespace, err)
				continue
			}
		}
	}

	return nil
}
