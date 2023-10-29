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

func getDeploymentNames(clientset kubernetes.Interface, namespace string) ([]string, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(deployments.Items))
	for _, deployment := range deployments.Items {
		names = append(names, deployment.Name)
	}
	return names, nil
}

func getStatefulSetNames(clientset kubernetes.Interface, namespace string) ([]string, error) {
	statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(statefulSets.Items))
	for _, statefulSet := range statefulSets.Items {
		names = append(names, statefulSet.Name)
	}
	return names, nil
}

func extractUnusedHpas(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ([]string, error) {
	deploymentNames, err := getDeploymentNames(clientset, namespace)
	if err != nil {
		return nil, err
	}
	statefulsetNames, err := getStatefulSetNames(clientset, namespace)
	if err != nil {
		return nil, err
	}
	hpas, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var diff []string
	for _, hpa := range hpas.Items {
		if hpa.Labels["kor/used"] == "true" {
			continue
		}

		// checks if the resource has any labels that match the excluded selector specified in opts.ExcludeLabels.
		// If it does, the resource is skipped.
		if excluded, _ := HasExcludedLabel(hpa.Labels, opts.ExcludeLabels); excluded {
			continue
		}
		// checks if the resource's age (measured from its last modified time) matches the included criteria
		// specified by the filter options.
		if included, _ := HasIncludedAge(hpa.CreationTimestamp, opts); !included {
			continue
		}
		// checks if the resource’s size falls within the range specified by opts.MinSize and opts.MaxSize.
		// If it doesn’t, the resource is skipped.
		if included, _ := HasIncludedSize(hpa, opts); !included {
			continue
		}

		switch hpa.Spec.ScaleTargetRef.Kind {
		case "Deployment":
			if !slices.Contains(deploymentNames, hpa.Spec.ScaleTargetRef.Name) {
				diff = append(diff, hpa.Name)
			}
		case "StatefulSet":
			if !slices.Contains(statefulsetNames, hpa.Spec.ScaleTargetRef.Name) {
				diff = append(diff, hpa.Name)
			}
		}
	}
	return diff, nil
}

func processNamespaceHpas(clientset kubernetes.Interface, namespace string, opts *FilterOptions) ([]string, error) {
	unusedHpas, err := extractUnusedHpas(clientset, namespace, opts)
	if err != nil {
		return nil, err
	}
	return unusedHpas, nil
}

func GetUnusedHpas(includeExcludeLists IncludeExcludeLists, opts *FilterOptions, clientset kubernetes.Interface, outputFormat string, slackOpts SlackOpts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceHpas(clientset, namespace, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Hpas")

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")

		resourceMap := make(map[string][]string)
		resourceMap["Hpa"] = diff
		response[namespace] = resourceMap
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedHpas, err := unusedResourceFormatter(outputFormat, outputBuffer, slackOpts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedHpas, nil
}
