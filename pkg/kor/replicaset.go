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

func processNamespaceReplicaSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	replicaSetList, err := clientset.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedReplicaSetNames []ResourceInfo

	for _, replicaSet := range replicaSetList.Items {
		if pass, _ := filter.SetObject(&replicaSet).Run(filterOpts); pass {
			continue
		}

		if replicaSet.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedReplicaSetNames = append(unusedReplicaSetNames, ResourceInfo{Name: replicaSet.Name, Reason: reason})
			continue
		}

		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(replicaSet.OwnerReferences) > 0 {
			continue
		}

		// if the replicaSet is specified 0 replica and current available & ready & fullyLabeled replica count is all 0, think the replicaSet is completed
		if *replicaSet.Spec.Replicas == 0 && replicaSet.Status.AvailableReplicas == 0 && replicaSet.Status.ReadyReplicas == 0 && replicaSet.Status.FullyLabeledReplicas == 0 {
			reason := "ReplicaSet is not in use"
			unusedReplicaSetNames = append(unusedReplicaSetNames, ResourceInfo{Name: replicaSet.Name, Reason: reason})
		}
	}
	if opts.DeleteFlag {
		if unusedReplicaSetNames, err = DeleteResource(unusedReplicaSetNames, clientset, namespace, "ReplicaSet", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete ReplicaSet %s in namespace %s: %v\n", unusedReplicaSetNames, namespace, err)
		}
	}
	return unusedReplicaSetNames, nil
}

func GetUnusedReplicaSets(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceReplicaSets(clientset, namespace, filterOpts, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["ReplicaSet"] = diff
		case "resource":
			appendResources(resources, "ReplicaSet", namespace, diff)
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

	unusedReplicaSets, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedReplicaSets, nil
}
