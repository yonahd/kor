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

func ProcessNamespaceJobs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	jobsList, err := clientset.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedJobNames []string

	for _, job := range jobsList.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}

		// if the job has completionTime and succeeded count greater than zero, think the job is completed
		if job.Status.CompletionTime != nil && job.Status.Succeeded > 0 {
			unusedJobNames = append(unusedJobNames, job.Name)
		}
	}

	return unusedJobNames, nil
}

func GetUnusedJobs(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := filterOpts.Namespaces(clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceJobs(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "Job", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Job %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "Job", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["Jobs"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedJobs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedJobs, nil
}
