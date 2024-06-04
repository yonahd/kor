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

	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/services/services.json
var servicesConfig []byte

func processNamespaceServices(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	endpointsList, err := clientset.CoreV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var endpointsWithoutSubsets []ResourceInfo

	for _, endpoints := range endpointsList.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}

		config, err := unmarshalConfig(servicesConfig)
		if err != nil {
			return nil, err
		}

		if isResourceException(endpoints.Name, namespace, config.ExceptionServices) {
			continue
		}

		status := ResourceInfo{Name: endpoints.Name}

		if endpoints.Labels["kor/used"] == "false" {
			status.Reason = "Marked with unused label"
			endpointsWithoutSubsets = append(endpointsWithoutSubsets, status)
			continue
		} else if len(endpoints.Subsets) == 0 {
			status.Reason = "Service has no endpoints"
			endpointsWithoutSubsets = append(endpointsWithoutSubsets, status)
		}
	}

	return endpointsWithoutSubsets, nil
}

func GetUnusedServices(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)

	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceServices(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		switch opts.GroupBy {
		case "namespace":
			if diff != nil {
				resources[namespace] = make(map[string][]ResourceInfo)
				resources[namespace]["Service"] = diff
			}
		case "resource":
			if diff != nil {
				appendResources(resources, "Service", namespace, diff)
			}

		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource2(diff, clientset, namespace, "Service", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Service %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
	}

	var outputBuffer bytes.Buffer
	var jsonResponse []byte
	switch outputFormat {
	case "table":
		outputBuffer = FormatOutput(resources, opts)
	case "json", "yaml":
		var err error
		if jsonResponse, err = json.MarshalIndent(resources, "", "  "); err != nil {
			return "", err
		}
	}

	unusedServices, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedServices, nil
}
