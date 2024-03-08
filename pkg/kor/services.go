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

func ProcessNamespaceServices(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	endpointsList, err := clientset.CoreV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var endpointsWithoutSubsets []string

	for _, endpoints := range endpointsList.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}

		if endpoints.Labels["kor/used"] == "false" {
			endpointsWithoutSubsets = append(endpointsWithoutSubsets, endpoints.Name)
			continue
		}

		if len(endpoints.Subsets) == 0 {
			endpointsWithoutSubsets = append(endpointsWithoutSubsets, endpoints.Name)
		}
	}

	return endpointsWithoutSubsets, nil
}

func GetUnusedServices(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer

	namespaces := filterOpts.Namespaces(clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceServices(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "Service", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Service %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "Services", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["Services"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedServices, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedServices, nil
}
