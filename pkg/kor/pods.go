package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func processNamespacePods(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	podsList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var evictedPods []ResourceInfo

	for _, pod := range podsList.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(pod.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&pod).Run(filterOpts); pass {
			continue
		}

		if pod.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			evictedPods = append(evictedPods, ResourceInfo{Name: pod.Name, Reason: reason})
			continue
		}

		if pod.Status.Phase == corev1.PodFailed && pod.Status.Reason == "Evicted" {
			reason := "Pod is evicted"
			evictedPods = append(evictedPods, ResourceInfo{Name: pod.Name, Reason: reason})
		}

		if pod.Status.Phase == corev1.PodFailed && pod.Status.Reason == "CrashLoopBackOff" {
			reason := "Pod is in CrashLoopBackOff"
			evictedPods = append(evictedPods, ResourceInfo{Name: pod.Name, Reason: reason})
		}

	}
	if opts.DeleteFlag {
		if evictedPods, err = DeleteResource(evictedPods, clientset, namespace, "Pod", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete Pod %s in namespace %s: %v\n", evictedPods, namespace, err)
		}
	}

	return evictedPods, nil
}

func GetUnusedPods(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespacePods(clientset, namespace, filterOpts, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Pod"] = diff
		case "resource":
			appendResources(resources, "Pod", namespace, diff)
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

	unusedPods, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedPods, nil
}
