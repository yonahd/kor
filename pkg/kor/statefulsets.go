package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func ProcessNamespaceStatefulSets(clientset kubernetes.Interface, namespace string) ([]string, error) {
	statefulSetsList, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var statefulSetsWithoutReplicas []string

	for _, statefulSet := range statefulSetsList.Items {
		if *statefulSet.Spec.Replicas == 0 {
			statefulSetsWithoutReplicas = append(statefulSetsWithoutReplicas, statefulSet.Name)
		}
	}

	return statefulSetsWithoutReplicas, nil
}

func GetUnusedStatefulSets(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, slackOpts SlackOpts) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)

	var outputBuffer bytes.Buffer

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceStatefulSets(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Statefulsets")

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
	}

	if slackOpts != (SlackOpts{}) {
		if err := SendToSlack(SlackMessage{}, slackOpts, outputBuffer.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send message to slack: %v\n", err)
		}
	} else {
		fmt.Println(outputBuffer.String())
	}
}

func GetUnusedStatefulSetsStructured(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string) (string, error) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceStatefulSets(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		resourceMap := make(map[string][]string)
		resourceMap["Statefulsets"] = diff
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
