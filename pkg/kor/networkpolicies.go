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

func processNamespaceNetworkPolicies(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	netpolList, err := clientset.NetworkingV1().NetworkPolicies(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedNetpols []ResourceInfo

	for _, netpol := range netpolList.Items {
		if pass, _ := filter.SetObject(&netpol).Run(filterOpts); pass {
			continue
		}

		if netpol.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedNetpols = append(unusedNetpols, ResourceInfo{Name: netpol.Name, Reason: reason})
			continue
		}

		// retrieve pods selected by the NetworkPolicy
		labelSelector, err := metav1.LabelSelectorAsSelector(&netpol.Spec.PodSelector)
		if err != nil {
			return nil, err
		}
		podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labelSelector.String(),
		})
		if err != nil {
			return nil, err
		}

		if len(podList.Items) == 0 {
			reason := "NetworkPolicy selects no pods"
			unusedNetpols = append(unusedNetpols, ResourceInfo{Name: netpol.Name, Reason: reason})
		}
	}

	return unusedNetpols, nil
}

func GetUnusedNetworkPolicies(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)

	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceNetworkPolicies(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if opts.DeleteFlag {
			if diff, err := DeleteResource(diff, clientset, namespace, "NetworkPolicy", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete NetworkPolicy %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["NetworkPolicy"] = diff
		case "resource":
			appendResources(resources, "NetworkPolicy", namespace, diff)
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

	unusedNetworkPolicies, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedNetworkPolicies, nil
}
