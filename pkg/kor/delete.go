package kor

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DeleteOpts struct {
	DeleteFlag    bool
	NoInteractive bool
}

func DeleteResourceCmd() map[string]func(clientset kubernetes.Interface, namespace, name string) error {
	var deleteResourceApiMap = map[string]func(clientset kubernetes.Interface, namespace, name string) error{
		"ConfigMap": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Secret": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Service": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Deployment": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"HPA": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Ingress": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.NetworkingV1beta1().Ingresses(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"PDB": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Roles": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.RbacV1().Roles(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"PVC": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"StatefulSet": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"ServiceAccount": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
	}

	return deleteResourceApiMap
}

func DeleteResource(diff []string, clientset kubernetes.Interface, namespace, resourceType string, noInteractive bool) ([]string, error) {
	deletedDiff := []string{}

	for _, resourceName := range diff {
		deleteFunc, exists := DeleteResourceCmd()[resourceType]
		if !exists {
			fmt.Printf("Resource type '%s' is not supported\n", resourceName)
			continue
		}

		if !noInteractive {
			fmt.Printf("Do you want to delete %s %s in namespace %s? (Y/N): ", resourceType, resourceName, namespace)
			var confirmation string
			_, err := fmt.Scanf("%s", &confirmation)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
				continue
			}

			if confirmation != "y" && confirmation != "Y" && confirmation != "yes" {
				deletedDiff = append(deletedDiff, resourceName)
				continue
			}
		}

		fmt.Printf("Deleting %s %s in namespace %s\n", resourceType, resourceName, namespace)
		if err := deleteFunc(clientset, namespace, resourceName); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete %s %s in namespace %s: %v\n", resourceType, resourceName, namespace, err)
			continue
		}
		deletedDiff = append(deletedDiff, resourceName+"-DELETED")
	}

	return deletedDiff, nil
}
