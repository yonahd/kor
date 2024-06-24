package kor

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
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
			return clientset.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"PDB": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Role": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.RbacV1().Roles(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"ClusterRole": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.RbacV1().ClusterRoles().Delete(context.TODO(), name, metav1.DeleteOptions{})
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
		"PV": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().PersistentVolumes().Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Pod": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.CoreV1().Pods(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"Job": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.BatchV1().Jobs(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"ReplicaSet": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.AppsV1().ReplicaSets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"DaemonSet": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.AppsV1().DaemonSets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"StorageClass": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.StorageV1().StorageClasses().Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
		"NetworkPolicy": func(clientset kubernetes.Interface, namespace, name string) error {
			return clientset.NetworkingV1().NetworkPolicies(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		},
	}

	return deleteResourceApiMap
}

func FlagDynamicResource(dynamicClient dynamic.Interface, namespace string, gvr schema.GroupVersionResource, resourceName string) error {
	resource, err := dynamicClient.
		Resource(gvr).
		Namespace(namespace).
		Get(context.TODO(), resourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	labels := resource.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["kor/used"] = "true"
	resource.SetLabels(labels)
	_, err = dynamicClient.
		Resource(gvr).
		Namespace(namespace).
		Update(context.TODO(), resource, metav1.UpdateOptions{})
	return err
}

func FlagResource(clientset kubernetes.Interface, namespace, resourceType, resourceName string) error {
	resource, err := getResource(clientset, namespace, resourceType, resourceName)
	if err != nil {
		return err
	}

	labelField := reflect.ValueOf(resource).Elem().FieldByName("Labels")
	if labelField.IsValid() {
		labels := labelField.Interface().(map[string]string)
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["kor/used"] = "true"
		labelField.Set(reflect.ValueOf(labels))
	} else {
		return fmt.Errorf("unable to set labels for resource type: %s", resourceType)
	}

	_, err = updateResource(clientset, namespace, resourceType, resource)
	return err
}

func updateResource(clientset kubernetes.Interface, namespace, resourceType string, resource interface{}) (interface{}, error) {
	switch resourceType {
	case "ConfigMap":
		return clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), resource.(*corev1.ConfigMap), metav1.UpdateOptions{})
	case "Secret":
		return clientset.CoreV1().Secrets(namespace).Update(context.TODO(), resource.(*corev1.Secret), metav1.UpdateOptions{})
	case "Service":
		return clientset.CoreV1().Services(namespace).Update(context.TODO(), resource.(*corev1.Service), metav1.UpdateOptions{})
	case "Deployment":
		return clientset.AppsV1().Deployments(namespace).Update(context.TODO(), resource.(*appsv1.Deployment), metav1.UpdateOptions{})
	case "HPA":
		return clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Update(context.TODO(), resource.(*autoscalingv1.HorizontalPodAutoscaler), metav1.UpdateOptions{})
	case "Ingress":
		return clientset.NetworkingV1().Ingresses(namespace).Update(context.TODO(), resource.(*networkingv1.Ingress), metav1.UpdateOptions{})
	case "PDB":
		return clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Update(context.TODO(), resource.(*policyv1beta1.PodDisruptionBudget), metav1.UpdateOptions{})
	case "Role":
		return clientset.RbacV1().Roles(namespace).Update(context.TODO(), resource.(*rbacv1.Role), metav1.UpdateOptions{})
	case "ClusterRole":
		return clientset.RbacV1().ClusterRoles().Update(context.TODO(), resource.(*rbacv1.ClusterRole), metav1.UpdateOptions{})
	case "PVC":
		return clientset.CoreV1().PersistentVolumeClaims(namespace).Update(context.TODO(), resource.(*corev1.PersistentVolumeClaim), metav1.UpdateOptions{})
	case "StatefulSet":
		return clientset.AppsV1().StatefulSets(namespace).Update(context.TODO(), resource.(*appsv1.StatefulSet), metav1.UpdateOptions{})
	case "ServiceAccount":
		return clientset.CoreV1().ServiceAccounts(namespace).Update(context.TODO(), resource.(*corev1.ServiceAccount), metav1.UpdateOptions{})
	case "PV":
		return clientset.CoreV1().PersistentVolumes().Update(context.TODO(), resource.(*corev1.PersistentVolume), metav1.UpdateOptions{})
	case "Pod":
		return clientset.CoreV1().Pods(namespace).Update(context.TODO(), resource.(*corev1.Pod), metav1.UpdateOptions{})
	case "Job":
		return clientset.BatchV1().Jobs(namespace).Update(context.TODO(), resource.(*batchv1.Job), metav1.UpdateOptions{})
	case "ReplicaSet":
		return clientset.AppsV1().ReplicaSets(namespace).Update(context.TODO(), resource.(*appsv1.ReplicaSet), metav1.UpdateOptions{})
	case "DaemonSet":
		return clientset.AppsV1().DaemonSets(namespace).Update(context.TODO(), resource.(*appsv1.DaemonSet), metav1.UpdateOptions{})
	case "StorageClass":
		return clientset.StorageV1().StorageClasses().Update(context.TODO(), resource.(*storagev1.StorageClass), metav1.UpdateOptions{})
	case "NetworkPolicy":
		return clientset.NetworkingV1().NetworkPolicies(namespace).Update(context.TODO(), resource.(*networkingv1.NetworkPolicy), metav1.UpdateOptions{})
	}
	return nil, fmt.Errorf("resource type '%s' is not supported", resourceType)
}

func getResource(clientset kubernetes.Interface, namespace, resourceType, resourceName string) (interface{}, error) {
	switch resourceType {
	case "ConfigMap":
		return clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "Secret":
		return clientset.CoreV1().Secrets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "Service":
		return clientset.CoreV1().Services(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "Deployment":
		return clientset.AppsV1().Deployments(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "HPA":
		return clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "Ingress":
		return clientset.NetworkingV1().Ingresses(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "PDB":
		return clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "Role":
		return clientset.RbacV1().Roles(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "ClusterRole":
		return clientset.RbacV1().ClusterRoles().Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "PVC":
		return clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "StatefulSet":
		return clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "ServiceAccount":
		return clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "PV":
		return clientset.CoreV1().PersistentVolumes().Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "Pod":
		return clientset.CoreV1().Pods(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "Job":
		return clientset.BatchV1().Jobs(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "ReplicaSet":
		return clientset.AppsV1().ReplicaSets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "DaemonSet":
		return clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "StorageClass":
		return clientset.StorageV1().StorageClasses().Get(context.TODO(), resourceName, metav1.GetOptions{})
	case "NetworkPolicy":
		return clientset.NetworkingV1().NetworkPolicies(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	}
	return nil, fmt.Errorf("resource type '%s' is not supported", resourceType)
}

func DeleteResourceWithFinalizer(resources []ResourceInfo, dynamicClient dynamic.Interface, namespace string, gvr schema.GroupVersionResource, noInteractive bool) ([]ResourceInfo, error) {
	var remainingResources []ResourceInfo
	for _, resource := range resources {
		if !noInteractive {
			fmt.Printf("Do you want to delete %s %s in namespace %s? (Y/N): ", gvr.Resource, resource.Name, namespace)
			var confirmation string
			_, err := fmt.Scanf("%s", &confirmation)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
				continue
			}

			if strings.ToLower(confirmation) != "y" && strings.ToLower(confirmation) != "yes" {
				resource.Reason = "not deleted - user declined"
				remainingResources = append(remainingResources, resource)

				fmt.Printf("Do you want to flag the resource %s %s in namespace %s as In Use? (Y/N): ", gvr.Resource, resource.Name, namespace)
				var inUse string
				_, err = fmt.Scanf("%s", &inUse)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
					continue
				}

				if strings.ToLower(inUse) == "y" || strings.ToLower(inUse) == "yes" {
					if err := FlagDynamicResource(dynamicClient, namespace, gvr, resource.Name); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to flag resource %s %s in namespace %s as In Use: %v\n", gvr.Resource, resource.Name, namespace, err)
					} else {
						resource.Reason = "flagged as in use"
					}
				}
				continue
			}
		}

		fmt.Printf("Deleting %s %s in namespace %s\n", gvr.Resource, resource.Name, namespace)
		if _, err := dynamicClient.
			Resource(gvr).
			Namespace(namespace).
			Patch(context.TODO(), resource.Name, types.MergePatchType,
				[]byte(`{"metadata":{"finalizers":null}}`),
				metav1.PatchOptions{}); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete %s %s in namespace %s: %v\n", gvr.Resource, resource.Name, namespace, err)
			continue
		}
		resource.Name = resource.Name + "-DELETED"
		remainingResources = append(remainingResources, resource)
	}

	return remainingResources, nil
}

func DeleteResource(diff []ResourceInfo, clientset kubernetes.Interface, namespace, resourceType string, noInteractive bool) ([]ResourceInfo, error) {
	deletedDiff := []ResourceInfo{}

	for _, resource := range diff {
		deleteFunc, exists := DeleteResourceCmd()[resourceType]
		if !exists {
			fmt.Printf("Resource type '%s' is not supported\n", resource.Name)
			continue
		}

		if !noInteractive {
			fmt.Printf("Do you want to delete %s %s in namespace %s? (Y/N): ", resourceType, resource.Name, namespace)
			var confirmation string
			_, err := fmt.Scanf("%s\n", &confirmation)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
				continue
			}

			if strings.ToLower(confirmation) != "y" && strings.ToLower(confirmation) != "yes" {
				deletedDiff = append(deletedDiff, resource)

				fmt.Printf("Do you want flag the resource %s %s in namespace %s as In Use? (Y/N): ", resourceType, resource.Name, namespace)
				var inUse string
				_, err := fmt.Scanf("%s\n", &inUse)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
					continue
				}

				if strings.ToLower(inUse) == "y" || strings.ToLower(inUse) == "yes" {
					if err := FlagResource(clientset, namespace, resourceType, resource.Name); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to flag resource %s %s in namespace %s as In Use: %v\n", resourceType, resource.Name, namespace, err)
					}
					continue
				}
				continue
			}
		}

		fmt.Printf("Deleting %s %s in namespace %s\n", resourceType, resource.Name, namespace)
		if err := deleteFunc(clientset, namespace, resource.Name); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete %s %s in namespace %s: %v\n", resourceType, resource.Name, namespace, err)
			continue
		}
		deletedResource := resource
		deletedResource.Name += "-DELETED"
		deletedDiff = append(deletedDiff, deletedResource)
	}

	return deletedDiff, nil
}
