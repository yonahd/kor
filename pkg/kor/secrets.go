package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/utils/strings/slices"

	"github.com/yonahd/kor/pkg/filters"
)

var exceptionSecretTypes = []string{
	`helm.sh/release.v1`,
	`kubernetes.io/dockerconfigjson`,
	`kubernetes.io/service-account-token`,
}

//go:embed exceptions/secrets/secrets.json
var secretsConfig []byte

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

func retrieveUsedSecret(clientset kubernetes.Interface, namespace string) ([]string, []string, []string, []string, []string, []string, error) {
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
			if volume.Projected != nil && volume.Projected.Sources != nil {
				for _, projectedResource := range volume.Projected.Sources {
					if projectedResource.Secret != nil {
						volumeSecrets = append(volumeSecrets, projectedResource.Secret.Name)
					}
				}
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

func retrieveSecretNames(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, []string, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, nil, err
	}

	var unusedSecretNames []string
	names := make([]string, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		if pass, _ := filter.SetObject(&secret).Run(filterOpts); pass {
			continue
		}

		if secret.Labels["kor/used"] == "false" {
			unusedSecretNames = append(unusedSecretNames, secret.Name)
			continue
		}

		config, err := unmarshalConfig(secretsConfig)
		if err != nil {
			return nil, nil, err
		}

		if !slices.Contains(exceptionSecretTypes, string(secret.Type)) && !isResourceException(secret.Name, namespace, config.ExceptionSecrets) {
			names = append(names, secret.Name)
		}
	}
	return names, unusedSecretNames, nil
}

func processNamespaceSecret(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	envSecrets, envSecrets2, volumeSecrets, initContainerEnvSecrets, pullSecrets, tlsSecrets, err := retrieveUsedSecret(clientset, namespace)
	if err != nil {
		return nil, err
	}

	envSecrets = RemoveDuplicatesAndSort(envSecrets)
	envSecrets2 = RemoveDuplicatesAndSort(envSecrets2)
	volumeSecrets = RemoveDuplicatesAndSort(volumeSecrets)
	initContainerEnvSecrets = RemoveDuplicatesAndSort(initContainerEnvSecrets)
	pullSecrets = RemoveDuplicatesAndSort(pullSecrets)
	tlsSecrets = RemoveDuplicatesAndSort(tlsSecrets)

	secretNames, unusedSecretNames, err := retrieveSecretNames(clientset, namespace, filterOpts)
	if err != nil {
		return nil, err
	}

	var usedSecrets []string
	slicesToAppend := [][]string{
		envSecrets,
		envSecrets2,
		volumeSecrets,
		pullSecrets,
		tlsSecrets,
		initContainerEnvSecrets,
	}

	for _, slice := range slicesToAppend {
		usedSecrets = append(usedSecrets, slice...)
	}
	diff := CalculateResourceDifference(usedSecrets, secretNames)
	diff = append(diff, unusedSecretNames...)
	return diff, nil

}

func GetUnusedSecrets(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := filterOpts.Namespaces(clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceSecret(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "Secret", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Secret %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "Secrets", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["Secrets"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedSecrets, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedSecrets, nil
}
