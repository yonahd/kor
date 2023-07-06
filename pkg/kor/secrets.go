package kor

import (
	"bytes"
	"context"
	"fmt"
	"github.com/olekukonko/tablewriter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

func retrieveVolumesAndEnvSecret(clientset *kubernetes.Clientset, namespace string) ([]string, []string, []string, []string, []string, error) {
	envSecrets := []string{}
	envSecrets2 := []string{}
	volumeSecrets := []string{}
	pullSecrets := []string{}
	tlsSecrets := []string{}

	// Retrieve pods in the specified namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Extract volume and environment information from pods
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					envSecrets = append(envSecrets, env.ValueFrom.SecretKeyRef.Name)
				}
			}
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil {
					envSecrets2 = append(envSecrets2, envFrom.SecretRef.Name)
				}
			}
		}
		for _, volume := range pod.Spec.Volumes {
			if volume.Secret != nil {
				volumeSecrets = append(volumeSecrets, volume.Secret.SecretName)
			}
		}
		if pod.Spec.ImagePullSecrets != nil {
			for _, secret := range pod.Spec.ImagePullSecrets {
				pullSecrets = append(pullSecrets, secret.Name)
			}
		}
	}

	// TODO get tls secrets

	return envSecrets, envSecrets2, volumeSecrets, pullSecrets, tlsSecrets, nil
}

func retrieveSecretNames(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	secrets, err := clientset.CoreV1().Secrets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		names = append(names, secret.Name)
	}
	return names, nil
}

func calculateSecretDifference(usedSecrets []string, secretNames []string) []string {
	difference := []string{}
	for _, name := range secretNames {
		found := false
		for _, usedName := range usedSecrets {
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

func formatOutput(namespace string, secretNames []string) string {
	if len(secretNames) == 0 {
		return fmt.Sprintf("No unused config maps found in the namespace: %s", namespace)
	}

	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"#", "Config Map Name"})

	for i, name := range secretNames {
		table.Append([]string{fmt.Sprintf("%d", i+1), name})
	}

	table.Render()

	return fmt.Sprintf("Unused Config Maps in Namespace: %s\n%s", namespace, buf.String())
}

func processNamespaceSecret(clientset *kubernetes.Clientset, namespace string) (string, error) {
	envSecrets, envSecrets2, volumeSecrets, pullSecrets, tlsSecrets, err := retrieveVolumesAndEnvSecret(clientset, namespace)
	if err != nil {
		return "", err
	}

	envSecrets = RemoveDuplicatesAndSort(envSecrets)
	envSecrets2 = RemoveDuplicatesAndSort(envSecrets2)
	volumeSecrets = RemoveDuplicatesAndSort(volumeSecrets)
	pullSecrets = RemoveDuplicatesAndSort(pullSecrets)
	tlsSecrets = RemoveDuplicatesAndSort(tlsSecrets)

	secretNames, err := retrieveSecretNames(clientset, namespace)
	if err != nil {
		return "", err
	}

	usedSecrets := append(append(append(append(envSecrets, envSecrets2...), volumeSecrets...), pullSecrets...), tlsSecrets...)
	diff := calculateCMDifference(usedSecrets, secretNames)
	return FormatOutput(namespace, diff, "Secrets"), nil

}

func GetUnusedSecrets() {
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
		output, err := processNamespaceSecret(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		fmt.Println(output)
		fmt.Println()
	}
}
