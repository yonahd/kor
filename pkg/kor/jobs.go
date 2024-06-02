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

func processNamespaceJobs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	jobsList, err := clientset.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedJobNames []ResourceInfo

	for _, job := range jobsList.Items {
		if pass, _ := filter.SetObject(&job).Run(filterOpts); pass {
			continue
		}

		// If the job has CompletionTime and Succeeded count greater than zero, the job is completed
		if job.Status.CompletionTime != nil && job.Status.Succeeded > 0 {
			reason := "Job has completed"
			unusedJobNames = append(unusedJobNames, ResourceInfo{Name: job.Name, Reason: reason})
			continue
		}
	}

	return unusedJobNames, nil
}

func GetUnusedJobs(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceJobs(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Job"] = diff
		case "resource":
			appendResources2(resources, "Job", namespace, diff)
		}
		if opts.DeleteFlag {
			if diff, err = DeleteResource2(diff, clientset, namespace, "Job", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Job %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
	}

	var outputBuffer bytes.Buffer
	var jsonResponse []byte
	switch outputFormat {
	case "table":
		outputBuffer = FormatOutput2(resources, opts)
	case "json", "yaml":
		var err error
		if jsonResponse, err = json.MarshalIndent(resources, "", "  "); err != nil {
			return "", err
		}
	}

	unusedJobs, err := unusedResourceFormatter2(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedJobs, nil
}
