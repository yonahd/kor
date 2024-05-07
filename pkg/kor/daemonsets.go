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

//go:embed exceptions/daemonsets/daemonsets.json
var daemonsetsConfig []byte

func processNamespaceDaemonSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	daemonSetsList, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var daemonSetsWithoutReplicas []string

	for _, daemonSet := range daemonSetsList.Items {
		if pass, _ := filter.SetObject(&daemonSet).Run(filterOpts); pass {
			continue
		}

		config, err := unmarshalConfig(daemonsetsConfig)
		if err != nil {
			return nil, err
		}

		if isResourceException(daemonSet.Name, daemonSet.Namespace, config.ExceptionDaemonSets) {
			continue
		}

		if daemonSet.Labels["kor/used"] == "false" {
			daemonSetsWithoutReplicas = append(daemonSetsWithoutReplicas, daemonSet.Name)
			continue
		}

		if daemonSet.Status.CurrentNumberScheduled == 0 {
			daemonSetsWithoutReplicas = append(daemonSetsWithoutReplicas, daemonSet.Name)
		}
	}

	return daemonSetsWithoutReplicas, nil
}

func GetUnusedDaemonSets(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]string)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceDaemonSets(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]string)
			resources[namespace]["DaemonSet"] = diff
		case "resource":
			appendResources(resources, "DaemonSet", namespace, diff)
		}
		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "DaemonSet", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete DaemonSet %s in namespace %s: %v\n", diff, namespace, err)
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

	unusedDaemonSets, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedDaemonSets, nil
}
