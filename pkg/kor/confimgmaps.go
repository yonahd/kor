package kor

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a todo",
	Long:  `This command will create todo`,
}

func retrieveVolumesAndEnvCM(clientset *kubernetes.Clientset, namespace string) ([]string, []string, []string, []string, []string, error) {
	volumesCM := []string{}
	volumesProjectedCM := []string{}
	envCM := []string{}
	envFromCM := []string{}
	envFromContainerCM := []string{}

	// Retrieve pods in the specified namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
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

	return volumesCM, volumesProjectedCM, envCM, envFromCM, envFromContainerCM, nil
}

func retrieveConfigMapNames(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(configmaps.Items))
	for _, configmap := range configmaps.Items {
		names = append(names, configmap.Name)
	}
	return names, nil
}

func calculateCMDifference(usedConfigMaps []string, configMapNames []string) []string {
	difference := []string{}
	for _, name := range configMapNames {
		found := false
		for _, usedName := range usedConfigMaps {
			if name == usedName {
				found = true
				break
			}
		}
		if !found {
			difference = append(difference, name)
		}
	}
	return difference
}

func processNamespaceCM(clientset *kubernetes.Clientset, namespace string) (string, error) {
	volumesCM, volumesProjectedCM, envCM, envFromCM, envFromContainerCM, err := retrieveVolumesAndEnvCM(clientset, namespace)
	if err != nil {
		return "", err
	}

	volumesCM = RemoveDuplicatesAndSort(volumesCM)
	volumesProjectedCM = RemoveDuplicatesAndSort(volumesProjectedCM)
	envCM = RemoveDuplicatesAndSort(envCM)
	envFromCM = RemoveDuplicatesAndSort(envFromCM)
	envFromContainerCM = RemoveDuplicatesAndSort(envFromContainerCM)

	configMapNames, err := retrieveConfigMapNames(clientset, namespace)
	if err != nil {
		return "", err
	}

	usedConfigMaps := append(append(append(append(volumesCM, volumesProjectedCM...), envCM...), envFromCM...), envFromContainerCM...)
	diff := calculateCMDifference(usedConfigMaps, configMapNames)
	return FormatOutput(namespace, diff, "Config Maps"), nil

}

func GetUnusedConfigmaps() {
	var kubeconfig string
	var namespaces []string

	kubeconfig = GetKubeConfigPath()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve namespaces: %v\n", err)
		os.Exit(1)
	}
	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, ns.Name)
	}

	for _, namespace := range namespaces {
		output, err := processNamespaceCM(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		fmt.Println(output)
		fmt.Println()
	}
}
