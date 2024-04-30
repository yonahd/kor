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

// These replica sets generated for default GKE
// TODO: Need to add it the ability for isResourceException to handle regex/wildcard matching
// var exceptionReplicaSets = []ExceptionResource{
// 	{
// 		ResourceName: "metrics-server-v0.6.3-7cb4458849",
// 		Namespace:    "gmp-system",
// 	},
// 	{
// 		ResourceName: "metrics-server-v0.6.3-7cb4458849",
// 		Namespace:    "kube-system",
// 	},
// }

func ProcessNamespaceReplicaSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	replicaSetList, err := clientset.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedReplicaSetNames []string

	for _, replicaSet := range replicaSetList.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}

		// if the replicaSet is specified 0 replica and current available & ready & fullyLabeled replica count is all 0, think the replicaSet is completed
		if *replicaSet.Spec.Replicas == 0 && replicaSet.Status.AvailableReplicas == 0 && replicaSet.Status.ReadyReplicas == 0 && replicaSet.Status.FullyLabeledReplicas == 0 {
			unusedReplicaSetNames = append(unusedReplicaSetNames, replicaSet.Name)
		}
	}

	return unusedReplicaSetNames, nil
}

func GetUnusedReplicaSets(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := filterOpts.Namespaces(clientset)
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
