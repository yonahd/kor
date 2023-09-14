package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/yaml"
)

var exceptionSecretTypes = []string{
	`helm.sh/release.v1`,
	`kubernetes.io/dockerconfigjson`,
	`kubernetes.io/service-account-token`,
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
	var envSecrets []string
	var envSecrets2 []string
	var volumeSecrets []string
	var pullSecrets []string
	var initContainerEnvSecrets []string

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

		for _, initContainer := range pod.Spec.InitContainers {
			for _, env := range initContainer.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					initContainerEnvSecrets = append(initContainerEnvSecrets, env.ValueFrom.SecretKeyRef.Name)
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

	return envSecrets, envSecrets2, volumeSecrets, initContainerEnvSecrets, pullSecrets, tlsSecrets, nil
}

func retrieveSecretNames(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	secrets, err := kubeClient.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		if secret.Labels["kor/used"] == "true" {
			continue
		}

		if !slices.Contains(exceptionSecretTypes, string(secret.Type)) {
			names = append(names, secret.Name)
		}
	}
	return names, nil
}

func processNamespaceSecret(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	envSecrets, envSecrets2, volumeSecrets, initContainerEnvSecrets, pullSecrets, tlsSecrets, err := retrieveUsedSecret(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	envSecrets = RemoveDuplicatesAndSort(envSecrets)
	envSecrets2 = RemoveDuplicatesAndSort(envSecrets2)
	volumeSecrets = RemoveDuplicatesAndSort(volumeSecrets)
	initContainerEnvSecrets = RemoveDuplicatesAndSort(initContainerEnvSecrets)
	pullSecrets = RemoveDuplicatesAndSort(pullSecrets)
	tlsSecrets = RemoveDuplicatesAndSort(tlsSecrets)

	secretNames, err := retrieveSecretNames(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	var usedSecrets []string
	slicesToAppend := [][]string{envSecrets, envSecrets2, volumeSecrets, pullSecrets, tlsSecrets, initContainerEnvSecrets}

	for _, slice := range slicesToAppend {
		usedSecrets = append(usedSecrets, slice...)
	}
	diff := CalculateResourceDifference(usedSecrets, secretNames)
	return diff, nil

}

func GetUnusedSecrets(includeExcludeLists IncludeExcludeLists, clientset *kubernetes.Clientset) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)

	for _, namespace := range namespaces {
		diff, err := processNamespaceSecret(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Secrets")
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedSecretsSendToSlackWebhook(includeExcludeLists IncludeExcludeLists, clientset *kubernetes.Clientset, slackWebhookURL string) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)

	var outputBuffer bytes.Buffer

	for _, namespace := range namespaces {
		diff, err := processNamespaceSecret(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Secrets")

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
	}

	if err := SendToSlackWebhook(slackWebhookURL, outputBuffer.String()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send output to Slack: %v\n", err)
	}
}

func GetUnusedSecretsSendToSlackAsFile(includeExcludeLists IncludeExcludeLists, clientset *kubernetes.Clientset, slackChannel string, slackAuthToken string) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)

	var outputBuffer bytes.Buffer

	for _, namespace := range namespaces {
		diff, err := processNamespaceSecret(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Secrets")

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
	}

	outputFilePath, _ := writeOutputToFile(outputBuffer)

	if err := SendFileToSlack(outputFilePath, "Unused Secrets", slackChannel, slackAuthToken); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send output to Slack: %v\n", err)
	}
}

func GetUnusedSecretsStructured(includeExcludeLists IncludeExcludeLists, clientset *kubernetes.Clientset, outputFormat string) (string, error) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceSecret(clientset, namespace)
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
