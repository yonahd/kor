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
	"sigs.k8s.io/yaml"
)

func processNamespacePdbs(clientset kubernetes.Interface, namespace string) ([]string, error) {
	var unusedPdbs []string
	pdbs, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pdb := range pdbs.Items {
		if pdb.Labels["kor/used"] == "true" {
			continue
		}

		selector := pdb.Spec.Selector
		if len(selector.MatchLabels) == 0 {
			unusedPdbs = append(unusedPdbs, pdb.Name)
			continue
		}
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(selector),
		})
		if err != nil {
			return nil, err
		}
		statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(selector),
		})
		if err != nil {
			return nil, err
		}
		if len(deployments.Items) == 0 && len(statefulSets.Items) == 0 {
			unusedPdbs = append(unusedPdbs, pdb.Name)
		}
	}
	return unusedPdbs, nil
}

func GetUnusedPdbs(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, slackOpts SlackOpts) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)

	var outputBuffer bytes.Buffer

	for _, namespace := range namespaces {
		diff, err := processNamespacePdbs(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Pdbs")

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

func GetUnusedPdbsStructured(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string) (string, error) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespacePdbs(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if len(diff) > 0 {
			if response[namespace] == nil {
				response[namespace] = make(map[string][]string)
			}
			response[namespace]["Pdb"] = diff
		}
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
