package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

func ProcessNamespaceReplicaSets(clientset kubernetes.Interface, namespace string, filterOpts *FilterOptions) ([]string, error) {
	replicaSetList, err := clientset.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var unusedReplicaSetNames []string

	for _, replicaSet := range replicaSetList.Items {
		if replicaSet.Labels["kor/used"] == "true" {
			continue
		}

		// checks if the resource has any labels that match the excluded selector specified in opts.ExcludeLabels.
		// If it does, the resource is skipped.
		if excluded, _ := HasExcludedLabel(replicaSet.Labels, filterOpts.ExcludeLabels); excluded {
			continue
		}
		// checks if the resource's age (measured from its last modified time) matches the included criteria
		// specified by the filter options.
		if included, _ := HasIncludedAge(replicaSet.CreationTimestamp, filterOpts); !included {
			continue
		}

		// if the replicaSet is specified 0 replica and current available & ready & fullyLabeled replica count is all 0, think the replicaSet is completed
		if *replicaSet.Spec.Replicas == 0 && replicaSet.Status.AvailableReplicas == 0 && replicaSet.Status.ReadyReplicas == 0 && replicaSet.Status.FullyLabeledReplicas == 0 {
			unusedReplicaSetNames = append(unusedReplicaSetNames, replicaSet.Name)
		}
	}

	return unusedReplicaSetNames, nil
}

func GetUnusedReplicaSets(includeExcludeLists IncludeExcludeLists, filterOpts *FilterOptions, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceReplicaSets(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "ReplicaSet", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete ReplicaSet %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "ReplicaSets", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["ReplicaSets"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedReplicaSets, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedReplicaSets, nil
}
