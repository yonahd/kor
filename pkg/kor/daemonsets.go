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

//go:embed exceptions/daemonsets/daemonsets.json
var daemonsetsConfig []byte

func processNamespaceDaemonSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	daemonSetsList, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(daemonsetsConfig)
	if err != nil {
		return nil, err
	}

	var daemonSetsWithoutReplicas []ResourceInfo

	for _, daemonSet := range daemonSetsList.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(daemonSet.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&daemonSet).Run(filterOpts); pass {
			continue
		}

		exceptionFound, err := isResourceException(daemonSet.Name, daemonSet.Namespace, config.ExceptionDaemonSets)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		if daemonSet.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			daemonSetsWithoutReplicas = append(daemonSetsWithoutReplicas, ResourceInfo{Name: daemonSet.Name, Reason: reason})
			continue
		}

		if daemonSet.Status.CurrentNumberScheduled == 0 {
			reason := "DaemonSet has no replicas"
			daemonSetsWithoutReplicas = append(daemonSetsWithoutReplicas, ResourceInfo{Name: daemonSet.Name, Reason: reason})
		}
	}
	if opts.DeleteFlag {
		if daemonSetsWithoutReplicas, err = DeleteResource(daemonSetsWithoutReplicas, clientset, namespace, "DaemonSet", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete DaemonSet %s in namespace %s: %v\n", daemonSetsWithoutReplicas, namespace, err)
		}
	}
	return daemonSetsWithoutReplicas, nil
}

func GetUnusedDaemonSets(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceDaemonSets(clientset, namespace, filterOpts, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["DaemonSet"] = diff
		case "resource":
			appendResources(resources, "DaemonSet", namespace, diff)
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

	unusedDaemonSets, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedDaemonSets, nil
}
