package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func validateServiceBackend(clientset kubernetes.Interface, namespace string, backend *v1.IngressBackend) bool {
	if backend.Service != nil {
		serviceName := backend.Service.Name

		_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			return false
		}
	}
	return true
}

func retrieveUsedIngress(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ([]string, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	usedIngresses := []string{}

	for _, ingress := range ingresses.Items {
		if ingress.Labels["kor/used"] == "true" {
			continue
		}

		// checks if the resource has any labels that match the excluded selector specified in opts.ExcludeLabels.
		// If it does, the resource is skipped.
		if excluded, _ := HasExcludedLabel(ingress.Labels, opts.ExcludeLabels); excluded {
			continue
		}
		// checks if the resource’s age (measured from its creation time) falls within the range specified by opts.MinAge
		// and opts.MaxAge. If it doesn’t, the resource is skipped.
		if !HasIncludedAge(ingress.CreationTimestamp, opts) {
			continue
		}
		// checks if the resource’s size falls within the range specified by opts.MinSize and opts.MaxSize.
		// If it doesn’t, the resource is skipped.
		if included, _ := HasIncludedSize(ingress, opts); !included {
			continue
		}

		used := true

		if ingress.Spec.DefaultBackend != nil {
			used = validateServiceBackend(clientset, namespace, ingress.Spec.DefaultBackend)
		}
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				used = true
				break
			}
			for _, path := range rule.HTTP.Paths {
				used = validateServiceBackend(clientset, namespace, &path.Backend)
				if used {
					break
				}
			}
			if used {
				break
			}
		}
		if used {
			usedIngresses = append(usedIngresses, ingress.Name)
		}
	}
	return usedIngresses, nil
}

func retrieveIngressNames(clientset kubernetes.Interface, namespace string) ([]string, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(ingresses.Items))
	for _, ingress := range ingresses.Items {
		names = append(names, ingress.Name)
	}
	return names, nil
}

func processNamespaceIngresses(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ([]string, error) {
	usedIngresses, err := retrieveUsedIngress(clientset, namespace, opts)
	if err != nil {
		return nil, err
	}
	ingressNames, err := retrieveIngressNames(clientset, namespace)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedIngresses, ingressNames)
	return diff, nil

}

func GetUnusedIngresses(includeExcludeLists IncludeExcludeLists, opts *FilterOptions, clientset kubernetes.Interface, outputFormat string, slackOpts SlackOpts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceIngresses(clientset, namespace, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Ingresses")

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")

		resourceMap := make(map[string][]string)
		resourceMap["Ingresses"] = diff
		response[namespace] = resourceMap
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedIngresses, err := unusedResourceFormatter(outputFormat, outputBuffer, slackOpts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedIngresses, nil
}
