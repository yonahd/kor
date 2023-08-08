package kor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func getSATokens(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	// Retrieve secrets in all namespaces with type "kubernetes.io/service-account-token"
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: "type=kubernetes.io/service-account-token",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Kubernetes Service Account tokens: %v", err)
	}

	tokenNames := make([]string, 0)

	// Extract secret names from secrets
	for _, secret := range secrets.Items {
		tokenNames = append(tokenNames, secret.ObjectMeta.Name)
	}

	return tokenNames, nil
}

func retrieveIngressTLS(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	secretNames := make([]string, 0)
	ingressList, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Ingress resources: %v", err)
	}

	// Extract secret names from Ingress TLS
	for _, ingress := range ingressList.Items {
		for _, tls := range ingress.Spec.TLS {
			secretNames = append(secretNames, tls.SecretName)
		}
	}

	return secretNames, nil

}

func retrieveUsedSecret(kubeClient *kubernetes.Clientset, namespace string) ([]string, []string, []string, []string, []string, []string, error) {
	envSecrets := []string{}
	envSecrets2 := []string{}
	volumeSecrets := []string{}
	pullSecrets := []string{}

	// Retrieve pods in the specified namespace
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
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

	tlsSecrets, err := retrieveIngressTLS(kubeClient, namespace)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	saTokens, err := getSATokens(kubeClient, namespace)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	return envSecrets, envSecrets2, volumeSecrets, pullSecrets, tlsSecrets, saTokens, nil
}

func retrieveSecretNames(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	secrets, err := kubeClient.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		if secret.Type != "helm.sh/release.v1" {
			names = append(names, secret.Name)
		}
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

func processNamespaceSecret(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	envSecrets, envSecrets2, volumeSecrets, pullSecrets, tlsSecrets, saTokens, err := retrieveUsedSecret(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	envSecrets = RemoveDuplicatesAndSort(envSecrets)
	envSecrets2 = RemoveDuplicatesAndSort(envSecrets2)
	volumeSecrets = RemoveDuplicatesAndSort(volumeSecrets)
	pullSecrets = RemoveDuplicatesAndSort(pullSecrets)
	tlsSecrets = RemoveDuplicatesAndSort(tlsSecrets)
	saTokens = RemoveDuplicatesAndSort(saTokens)

	secretNames, err := retrieveSecretNames(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	usedSecrets := append(append(append(append(append(envSecrets, envSecrets2...), volumeSecrets...), pullSecrets...), tlsSecrets...), saTokens...)
	diff := calculateSecretDifference(usedSecrets, secretNames)
	return diff, nil

}

func GetUnusedSecrets(namespace string, kubeconfig string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)

	namespaces = SetNamespaceList(namespace, kubeClient)

	for _, namespace := range namespaces {
		diff, err := processNamespaceSecret(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Secrets")
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedSecretsJSON(namespace string, kubeconfig string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient(kubeconfig)
	namespaces = SetNamespaceList(namespace, kubeClient)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceSecret(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		resourceMap := make(map[string][]string)
		resourceMap["Secrets"] = diff
		response[namespace] = resourceMap
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonResponse), nil
}
