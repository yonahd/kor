package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/filters"
)

func ProcessNamespaceDeployments(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	deploymentsList, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var deploymentsWithoutReplicas []string

	for _, deployment := range deploymentsList.Items {
		if pass, _ := filter.SetObject(&deployment).Run(filterOpts); pass {
			continue
		}

		if deployment.Labels["kor/used"] == "false" {
			deploymentsWithoutReplicas = append(deploymentsWithoutReplicas, deployment.Name)
			continue
		}

		if *deployment.Spec.Replicas == 0 {
			deploymentsWithoutReplicas = append(deploymentsWithoutReplicas, deployment.Name)
		}
	}

	return deploymentsWithoutReplicas, nil
}

func GetUnusedDeployments(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	response := make(map[string]map[string][]string)

	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := ProcessNamespaceDeployments(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "Deployment", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Deployment %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "Deployments", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["Deployments"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedDeployments, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedDeployments, nil
}
