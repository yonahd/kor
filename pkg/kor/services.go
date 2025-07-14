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

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/services/services.json
var servicesConfig []byte

func processNamespaceServices(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	endpointSlices, err := clientset.DiscoveryV1().EndpointSlices(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(servicesConfig)
	if err != nil {
		return nil, err
	}

	var endpointsWithoutSubsets []ResourceInfo

	for _, endpoints := range endpointSlices.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(endpoints.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&endpoints).Run(filterOpts); pass {
			continue
		}

		exceptionFound, err := isResourceException(endpoints.Name, endpoints.Namespace, config.ExceptionServices)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		status := ResourceInfo{Name: endpoints.ObjectMeta.Labels["kubernetes.io/service-name"]}

		if endpoints.Labels["kor/used"] == "false" {
			status.Reason = "Marked with unused label"
			endpointsWithoutSubsets = append(endpointsWithoutSubsets, status)
			continue
		} else if len(endpoints.Endpoints) == 0 {
			status.Reason = "Service has no endpointslices"
			endpointsWithoutSubsets = append(endpointsWithoutSubsets, status)
		}
	}

	if opts.DeleteFlag {
		if endpointsWithoutSubsets, err = DeleteResource(endpointsWithoutSubsets, clientset, namespace, "Service", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete Service %s in namespace %s: %v\n", endpointsWithoutSubsets, namespace, err)
		}
	}

	return endpointsWithoutSubsets, nil
}

func GetUnusedServices(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)

	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceServices(clientset, namespace, filterOpts, opts)
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
