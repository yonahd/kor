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
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/priorityclasses/priorityclasses.json
var priorityClassesConfig []byte

func retrieveUsedPriorityClasses(clientset kubernetes.Interface) ([]string, error) {
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Pods: %v", err)
	}

	var usedPriorityClasses []string

	// Iterate through each Pod and check for PriorityClass usage
	for _, pod := range pods.Items {
		if pod.Spec.PriorityClassName != "" {
			usedPriorityClasses = append(usedPriorityClasses, pod.Spec.PriorityClassName)
		}
	}

	return usedPriorityClasses, nil
}

func processPriorityClasses(clientset kubernetes.Interface, filterOpts *filters.Options) ([]ResourceInfo, error) {
	pcs, err := clientset.SchedulingV1().PriorityClasses().List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(priorityClassesConfig)
	if err != nil {
		return nil, err
	}

	var unusedPriorityClasses []ResourceInfo
	priorityClassNames := make([]string, 0, len(pcs.Items))

	for _, pc := range pcs.Items {
		// Skip global default PriorityClasses as they are used by pods without explicit priority class
		if pc.GlobalDefault {
			continue
		}

		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(pc.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&pc).Run(filterOpts); pass {
			continue
		}

		if pc.Labels["kor/used"] == "false" {
			unusedPriorityClasses = append(unusedPriorityClasses, ResourceInfo{Name: pc.Name, Reason: "Marked with unused label"})
			continue
		}

		exceptionFound, err := isResourceException(pc.Name, "", config.ExceptionPriorityClasses)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		priorityClassNames = append(priorityClassNames, pc.Name)
	}

	usedPriorityClasses, err := retrieveUsedPriorityClasses(clientset)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedPriorityClasses, priorityClassNames)
	for _, name := range diff {
		unusedPriorityClasses = append(unusedPriorityClasses, ResourceInfo{Name: name, Reason: "Not in Use"})
	}
	return unusedPriorityClasses, nil
}

func GetUnusedPriorityClasses(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processPriorityClasses(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process priorityClasses: %v\n", err)
	}
	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, "", "PriorityClass", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete PriorityClass %s: %v\n", diff, err)
		}
	}
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["PriorityClass"] = diff
	case "resource":
		appendResources(resources, "PriorityClass", "", diff)
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

	unusedPriorityClasses, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedPriorityClasses, nil
}
