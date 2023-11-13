package kor

import (
	"context"
	"fmt"
	"os"
	"strings"

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

func FlagResource(clientset kubernetes.Interface, namespace, resourceType, resourceName string) error {
	var flagResourceApiMap = map[string]func(clientset kubernetes.Interface, namespace, resourceName string) error{
		"ConfigMap": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if configMap.Labels == nil {
				configMap.Labels = make(map[string]string)
			}

			configMap.Labels["kor/used"] = "true"
			_, err = clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
			return err
		},
		"Secret": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if secret.Labels == nil {
				secret.Labels = make(map[string]string)
			}

			secret.Labels["kor/used"] = "true"
			_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
			return err
		},
		"Service": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if service.Labels == nil {
				service.Labels = make(map[string]string)
			}

			service.Labels["kor/used"] = "true"
			_, err = clientset.CoreV1().Services(namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
			return err
		},
		"Deployment": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if deployment.Labels == nil {
				deployment.Labels = make(map[string]string)
			}

			deployment.Labels["kor/used"] = "true"
			_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
			return err
		},
		"HPA": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			hpa, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if hpa.Labels == nil {
				hpa.Labels = make(map[string]string)
			}

			hpa.Labels["kor/used"] = "true"
			_, err = clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Update(context.TODO(), hpa, metav1.UpdateOptions{})
			return err
		},
		"Ingress": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			ingress, err := clientset.NetworkingV1beta1().Ingresses(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if ingress.Labels == nil {
				ingress.Labels = make(map[string]string)
			}

			ingress.Labels["kor/used"] = "true"
			_, err = clientset.NetworkingV1beta1().Ingresses(namespace).Update(context.TODO(), ingress, metav1.UpdateOptions{})
			return err
		},
		"PDB": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			pdb, err := clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if pdb.Labels == nil {
				pdb.Labels = make(map[string]string)
			}

			pdb.Labels["kor/used"] = "true"
			_, err = clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Update(context.TODO(), pdb, metav1.UpdateOptions{})
			return err
		},
		"Roles": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			role, err := clientset.RbacV1().Roles(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if role.Labels == nil {
				role.Labels = make(map[string]string)
			}

			role.Labels["kor/used"] = "true"
			_, err = clientset.RbacV1().Roles(namespace).Update(context.TODO(), role, metav1.UpdateOptions{})
			return err
		},
		"PVC": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if pvc.Labels == nil {
				pvc.Labels = make(map[string]string)
			}

			pvc.Labels["kor/used"] = "true"
			_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Update(context.TODO(), pvc, metav1.UpdateOptions{})
			return err
		},
		"StatefulSet": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if statefulSet.Labels == nil {
				statefulSet.Labels = make(map[string]string)
			}

			statefulSet.Labels["kor/used"] = "true"
			_, err = clientset.AppsV1().StatefulSets(namespace).Update(context.TODO(), statefulSet, metav1.UpdateOptions{})
			return err
		},
		"ServiceAccount": func(clientset kubernetes.Interface, namespace, resourceName string) error {
			serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if serviceAccount.Labels == nil {
				serviceAccount.Labels = make(map[string]string)
			}

			serviceAccount.Labels["kor/used"] = "true"
			_, err = clientset.CoreV1().ServiceAccounts(namespace).Update(context.TODO(), serviceAccount, metav1.UpdateOptions{})
			return err
		},
	}

	return flagResourceApiMap[resourceType](clientset, namespace, resourceName)
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

			if strings.ToLower(confirmation) != "y" && strings.ToLower(confirmation) != "yes" {
				deletedDiff = append(deletedDiff, resourceName)

				fmt.Printf("Do you want flag the resource %s %s in namespace %s as In Use? (Y/N): ", resourceType, resourceName, namespace)
				var inUse string
				_, err := fmt.Scanf("%s", &inUse)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
					continue
				}

				if strings.ToLower(inUse) == "y" || strings.ToLower(inUse) == "yes" {
					if err := FlagResource(clientset, namespace, resourceType, resourceName); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to flag resource %s %s in namespace %s as In Use: %v\n", resourceType, resourceName, namespace, err)
					}
					continue
				}
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
