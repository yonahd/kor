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
)

var exceptionSecretTypes = []string{
	`helm.sh/release.v1`,
	`kubernetes.io/dockerconfigjson`,
	`kubernetes.io/service-account-token`,
}

func retrieveIngressTLS(clientset kubernetes.Interface, namespace string) ([]string, error) {
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

func retrieveUsedSecret(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ([]string, []string, []string, []string, []string, []string, error) {
	var envSecrets []string
	var envSecrets2 []string
	var volumeSecrets []string
	var pullSecrets []string
	var initContainerEnvSecrets []string

	// Retrieve pods in the specified namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
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

	tlsSecrets, err := retrieveIngressTLS(clientset, namespace)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	return envSecrets, envSecrets2, volumeSecrets, initContainerEnvSecrets, pullSecrets, tlsSecrets, nil
}

func retrieveSecretNames(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ([]string, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		if secret.Labels["kor/used"] == "true" {
			continue
		}

		// checks if the resource has any labels that match the excluded selector specified in opts.ExcludeLabels.
		// If it does, the resource is skipped.
		if excluded, _ := HasExcludedLabel(secret.Labels, opts.ExcludeLabels); excluded {
			continue
		}
		// checks if the resource's age (measured from its last modified time) matches the included criteria
		// specified by the filter options.
		if included, _ := HasIncludedAge(secret.CreationTimestamp, opts); !included {
			continue
		}
		// checks if the resource’s size falls within the range specified by opts.MinSize and opts.MaxSize.
		// If it doesn’t, the resource is skipped.
		if included, _ := HasIncludedSize(secret, opts); !included {
			continue
		}

		if !slices.Contains(exceptionSecretTypes, string(secret.Type)) {
			names = append(names, secret.Name)
		}
	}
	return names, nil
}

func processNamespaceSecret(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ([]string, error) {
	envSecrets, envSecrets2, volumeSecrets, initContainerEnvSecrets, pullSecrets, tlsSecrets, err := retrieveUsedSecret(clientset, namespace, opts)
	if err != nil {
		return nil, err
	}

	envSecrets = RemoveDuplicatesAndSort(envSecrets)
	envSecrets2 = RemoveDuplicatesAndSort(envSecrets2)
	volumeSecrets = RemoveDuplicatesAndSort(volumeSecrets)
	initContainerEnvSecrets = RemoveDuplicatesAndSort(initContainerEnvSecrets)
	pullSecrets = RemoveDuplicatesAndSort(pullSecrets)
	tlsSecrets = RemoveDuplicatesAndSort(tlsSecrets)

	secretNames, err := retrieveSecretNames(clientset, namespace, opts)
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

func GetUnusedSecrets(includeExcludeLists IncludeExcludeLists, opts *FilterOptions, clientset kubernetes.Interface, outputFormat string, slackOpts SlackOpts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceSecret(clientset, namespace, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Secrets")

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")

		resourceMap := make(map[string][]string)
		resourceMap["Secrets"] = diff
		response[namespace] = resourceMap
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedSecrets, err := unusedResourceFormatter(outputFormat, outputBuffer, slackOpts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedSecrets, nil
}
