package kor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"sigs.k8s.io/yaml"
)

var exceptionconfigmaps = []ExceptionResource{
	{ResourceName: "aws-auth", Namespace: "kube-system"},
	{ResourceName: "kube-root-ca.crt", Namespace: "*"},
}

func retrieveUsedCM(kubeClient *kubernetes.Clientset, namespace string) ([]string, []string, []string, []string, []string, error) {
	volumesCM := []string{}
	volumesProjectedCM := []string{}
	envCM := []string{}
	envFromCM := []string{}
	envFromContainerCM := []string{}

	// Retrieve pods in the specified namespace
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Extract volume and environment information from pods
	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.ConfigMap != nil {
				volumesCM = append(volumesCM, volume.ConfigMap.Name)
			}
			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.ConfigMap != nil {
						volumesProjectedCM = append(volumesProjectedCM, source.ConfigMap.Name)
					}
				}
			}
		}
		for _, container := range pod.Spec.Containers {
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					envCM = append(envCM, env.ValueFrom.ConfigMapKeyRef.Name)
				}
			}
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					envFromCM = append(envFromCM, envFrom.ConfigMapRef.Name)
				}
			}
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					envFromContainerCM = append(envFromContainerCM, envFrom.ConfigMapRef.Name)
				}
			}
		}
	}

	for _, resource := range exceptionconfigmaps {
		if resource.Namespace == namespace || resource.Namespace == "*" {
			volumesCM = append(volumesCM, resource.ResourceName)
		}
	}

	return volumesCM, volumesProjectedCM, envCM, envFromCM, envFromContainerCM, nil
}

func retrieveConfigMapNames(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	configmaps, err := kubeClient.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(configmaps.Items))
	for _, configmap := range configmaps.Items {
		if configmap.Labels["kor/used"] == "true" {
			continue
		}

		names = append(names, configmap.Name)
	}
	return names, nil
}

func processNamespaceCM(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	volumesCM, volumesProjectedCM, envCM, envFromCM, envFromContainerCM, err := retrieveUsedCM(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	volumesCM = RemoveDuplicatesAndSort(volumesCM)
	volumesProjectedCM = RemoveDuplicatesAndSort(volumesProjectedCM)
	envCM = RemoveDuplicatesAndSort(envCM)
	envFromCM = RemoveDuplicatesAndSort(envFromCM)
	envFromContainerCM = RemoveDuplicatesAndSort(envFromContainerCM)

	configMapNames, err := retrieveConfigMapNames(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	usedConfigMaps := append(append(append(append(volumesCM, volumesProjectedCM...), envCM...), envFromCM...), envFromContainerCM...)
	diff := CalculateResourceDifference(usedConfigMaps, configMapNames)
	return diff, nil

}

func GetUnusedConfigmaps(includeExcludeLists IncludeExcludeLists, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(includeExcludeLists, kubeClient)

	for _, namespace := range namespaces {
		diff, err := processNamespaceCM(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Config Maps")
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedConfigmapsStructured(includeExcludeLists IncludeExcludeLists, kubeconfig string, outputFormat string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)
	namespaces = SetNamespaceList(includeExcludeLists, kubeClient)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceCM(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		resourceMap := make(map[string][]string)
		resourceMap["ConfigMap"] = diff
		response[namespace] = resourceMap
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	if outputFormat == "yaml" {
		yamlResponse, err := yaml.JSONToYAML(jsonResponse)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		return string(yamlResponse), nil
	} else {
		return string(jsonResponse), nil
	}

}
