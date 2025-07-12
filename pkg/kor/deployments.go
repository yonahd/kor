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

func processNamespaceDeployments(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	deploymentsList, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var deploymentsWithoutReplicas []ResourceInfo

	for _, deployment := range deploymentsList.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(deployment.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&deployment).Run(filterOpts); pass {
			continue
		}

		if deployment.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			deploymentsWithoutReplicas = append(deploymentsWithoutReplicas, ResourceInfo{Name: deployment.Name, Reason: reason})
			continue
		}

		if *deployment.Spec.Replicas == 0 {
			reason := "Deployment has no replicas"
			deploymentsWithoutReplicas = append(deploymentsWithoutReplicas, ResourceInfo{Name: deployment.Name, Reason: reason})
		}
	}
	if opts.DeleteFlag {
		if deploymentsWithoutReplicas, err = DeleteResource(deploymentsWithoutReplicas, clientset, namespace, "Deployment", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete Deployment %s in namespace %s: %v\n", deploymentsWithoutReplicas, namespace, err)
		}
	}

	return deploymentsWithoutReplicas, nil
}

func GetUnusedDeployments(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceDeployments(clientset, namespace, filterOpts, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Deployment"] = diff
		case "resource":
			appendResources(resources, "Deployment", namespace, diff)
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

	unusedDeployments, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedDeployments, nil
}
