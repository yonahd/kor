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

func ProcessNamespaceStatefulSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, enrich bool) (interface{}, error) {
	statefulSetsList, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	if enrich {
		var statefulSetsStatus []ResourceInfo

		for _, statefulSet := range statefulSetsList.Items {
			if pass, _ := filter.Run(filterOpts); pass {
				continue
			}

			status := ResourceInfo{Name: statefulSet.Name}
			if statefulSet.Labels["kor/used"] == "false" {
				status.Reason = "Marked with unused label"
				statefulSetsStatus = append(statefulSetsStatus, status)
				continue
			}

			if *statefulSet.Spec.Replicas == 0 {
				status.Reason = "Statefulset has no replicas"
				statefulSetsStatus = append(statefulSetsStatus, status)
			}
		}

		return statefulSetsStatus, nil
	}

	var statefulSets []string

	for _, statefulSet := range statefulSetsList.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}

		if statefulSet.Labels["kor/used"] == "false" || *statefulSet.Spec.Replicas == 0 {
			statefulSets = append(statefulSets, statefulSet.Name)
		}
	}

	return statefulSets, nil
}

func GetUnusedStatefulSets(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	var response interface{}

	namespaces := filterOpts.Namespaces(clientset)
	if opts.PrintReason {
		response = make(map[string]map[string][]ResourceInfo)
	} else {
		response = make(map[string]map[string][]string)
	}

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceStatefulSets(clientset, namespace, filterOpts, opts.PrintReason)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "StatefulSet", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete StatefulSet %s in namespace %s: %v\n", diff, namespace, err)
			}
		}

		var output string
		if opts.PrintReason {
			output = FormatEnrichedOutput(namespace, diff, "StatefulSets", opts)
		} else {
			output = FormatOutput(namespace, diff, "StatefulSets", opts)
		}

		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			switch res := response.(type) {
			case map[string]map[string][]string:
				diffSlice, _ := diff.([]string)
				res[namespace] = map[string][]string{"StatefulSets": diffSlice}
			case map[string]map[string][]ResourceInfo:
				diffSlice, _ := diff.([]ResourceInfo)
				res[namespace] = map[string][]ResourceInfo{"StatefulSets": diffSlice}
			default:
				fmt.Println("Invalid type for response")
			}
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedStatefulSets, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedStatefulSets, nil
}
