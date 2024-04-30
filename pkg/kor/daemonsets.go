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

var exceptionDaemonSets = []ExceptionResource{
	{
		ResourceName: "kube-proxy",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "fluentbit-gke-256pd",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "fluentbit-gke-max",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "gke-metrics-agent-scaling-10",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "gke-metrics-agent-scaling-100",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "gke-metrics-agent-scaling-20",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "gke-metrics-agent-scaling-200",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "gke-metrics-agent-scaling-50",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "gke-metrics-agent-scaling-500",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "gke-metrics-agent-windows",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "metadata-proxy-v0.1",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "nccl-fastsocket-installer",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "nvidia-gpu-device-plugin-large-cos",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "nvidia-gpu-device-plugin-large-ubuntu",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "nvidia-gpu-device-plugin-medium-cos",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "nvidia-gpu-device-plugin-medium-ubuntu",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "nvidia-gpu-device-plugin-small-cos",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "nvidia-gpu-device-plugin-small-ubuntu",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "pdcsi-node-windows",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "runsc-metric-server",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "tpu-device-plugin",
		Namespace:    "kube-system",
	},
}

func ProcessNamespaceDaemonSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	daemonSetsList, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var daemonSetsWithoutReplicas []string

	for _, daemonSet := range daemonSetsList.Items {
		if pass, _ := filter.SetObject(&daemonSet).Run(filterOpts); pass {
			continue
		}
		if isResourceException(daemonSet.Name, "", exceptionDaemonSets) {
			continue
		}

		if daemonSet.Labels["kor/used"] == "false" {
			daemonSetsWithoutReplicas = append(daemonSetsWithoutReplicas, daemonSet.Name)
			continue
		}

		if daemonSet.Status.CurrentNumberScheduled == 0 {
			daemonSetsWithoutReplicas = append(daemonSetsWithoutReplicas, daemonSet.Name)
		}
	}

	return daemonSetsWithoutReplicas, nil
}

func GetUnusedDaemonSets(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := filterOpts.Namespaces(clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := ProcessNamespaceDaemonSets(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "DaemonSet", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete DaemonSet %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "DaemonSets", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["DaemonSets"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedDaemonSets, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedDaemonSets, nil
}
