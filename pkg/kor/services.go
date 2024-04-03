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

var exceptionServices = []ExceptionResource{
	{ResourceName: "docker.io-hostpath", Namespace: "kube-system"},
}

type ResourceInfo struct {
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

func ProcessNamespaceServices(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, enrich bool) (interface{}, error) {
	endpointsList, err := clientset.CoreV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	if enrich {
		var servicesStatus []ResourceInfo

		for _, endpoints := range endpointsList.Items {
			if pass, _ := filter.Run(filterOpts); pass {
				continue
			}

			status := ResourceInfo{Name: endpoints.Name}
			if endpoints.Labels["kor/used"] == "false" {
				status.Reason = "Marked with unused label"
				servicesStatus = append(servicesStatus, status)
				continue
			} else if len(endpoints.Subsets) == 0 {
				status.Reason = "No endpoints"
				servicesStatus = append(servicesStatus, status)
			}

		}

		return servicesStatus, nil
	}

	var services []string

	for _, endpoints := range endpointsList.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}

		if endpoints.Labels["kor/used"] == "false" || len(endpoints.Subsets) == 0 {
			services = append(services, endpoints.Name)
		}
	}

	return services, nil
}

func GetUnusedServices(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	var output string
	var response interface{}

	namespaces := filterOpts.Namespaces(clientset)
	if opts.PrintReason {
		response = make(map[string]map[string][]ResourceInfo)
	} else {
		response = make(map[string]map[string][]string)
	}

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceServices(clientset, namespace, filterOpts, opts.PrintReason)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "Service", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Service %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		if opts.PrintReason {
			output = FormatEnrichedOutput(namespace, diff, "Services", opts)
		} else {
			output = FormatOutput(namespace, diff, "Services", opts)
		}

		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			switch res := response.(type) {
			case map[string]map[string][]string:
				diffSlice, _ := diff.([]string)
				res[namespace] = map[string][]string{"Services": diffSlice}
			case map[string]map[string][]ResourceInfo:
				diffSlice, _ := diff.([]ResourceInfo)
				res[namespace] = map[string][]ResourceInfo{"Services": diffSlice}
			default:
				fmt.Println("Invalid type for response")
			}

		}
	}

	var jsonResponse []byte
	var err error

	if opts.PrintReason {
		jsonResponse, err = json.MarshalIndent(response, "", "  ")
	} else {
		// Marshal the response map instead of the string
		jsonResponse, err = json.MarshalIndent(response, "", "  ")
	}

	if err != nil {
		return "", err
	}

	unusedServices, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedServices, nil
}
