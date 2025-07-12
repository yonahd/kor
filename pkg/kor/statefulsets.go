package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func processNamespaceStatefulSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	statefulSetsList, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var statefulSetsWithoutReplicas []ResourceInfo

	for _, statefulSet := range statefulSetsList.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(statefulSet.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&statefulSet).Run(filterOpts); pass {
			continue
		}

		status := ResourceInfo{Name: statefulSet.Name}

		if statefulSet.Labels["kor/used"] == "false" {
			status.Reason = "Marked with unused label"
			statefulSetsWithoutReplicas = append(statefulSetsWithoutReplicas, status)
			continue
		}

		if *statefulSet.Spec.Replicas == 0 {
			status.Reason = "StatefulSet has no replicas"
			statefulSetsWithoutReplicas = append(statefulSetsWithoutReplicas, status)
		}
	}
	if opts.DeleteFlag {
		if statefulSetsWithoutReplicas, err = DeleteResource(statefulSetsWithoutReplicas, clientset, namespace, "StatefulSet", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete Statefulset %s in namespace %s: %v\n", statefulSetsWithoutReplicas, namespace, err)
		}
	}

	return statefulSetsWithoutReplicas, nil
}

func GetUnusedStatefulSets(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceStatefulSets(clientset, namespace, filterOpts, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			if diff != nil {
				resources[namespace] = make(map[string][]ResourceInfo)
				resources[namespace]["StatefulSet"] = diff
			}
		case "resource":
			if diff != nil {
				appendResources(resources, "StatefulSet", namespace, diff)
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

	unusedStatefulsets, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedStatefulsets, nil
}
