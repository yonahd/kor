package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/jobs/jobs.json
var jobsConfig []byte

func processNamespaceJobs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	jobsList, err := clientset.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(jobsConfig)
	if err != nil {
		return nil, err
	}

	var unusedJobNames []ResourceInfo

	for _, job := range jobsList.Items {
		if pass, _ := filter.SetObject(&job).Run(filterOpts); pass {
			continue
		}

		exceptionFound, err := isResourceException(job.Name, job.Namespace, config.ExceptionJobs)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		// if the job has completionTime and succeeded count greater than zero, think the job is completed
		if job.Status.CompletionTime != nil && job.Status.Succeeded > 0 {
			reason := "Job has completed"
			unusedJobNames = append(unusedJobNames, ResourceInfo{Name: job.Name, Reason: reason})
			continue
		} else {
			// Check if the job has a condition indicating it has exceeded the backoff limit
			for _, condition := range job.Status.Conditions {
				if condition.Type == batchv1.JobFailed && condition.Reason == "BackoffLimitExceeded" {
					reason := "Job has exceeded backoff limit"
					unusedJobNames = append(unusedJobNames, ResourceInfo{Name: job.Name, Reason: reason})
					break
				}
			}
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
		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "Job", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Job %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Job"] = diff
		case "resource":
			appendResources(resources, "Job", namespace, diff)
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

	unusedJobs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedJobs, nil
}
