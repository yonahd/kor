package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/filters"
)

var exceptionConfigMaps = []ExceptionResource{
	{
		ResourceName: "aws-auth",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "kube-root-ca.crt",
		Namespace:    "*",
	},
	{
		ResourceName: "extension-apiserver-authentication",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "kube-apiserver-legacy-service-account-token-tracking",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "kubeadm-config",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "kubelet-config",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "kubernetes-dashboard-settings",
		Namespace:    "kubernetes-dashboard",
	},
	{
		ResourceName: "cluster-info",
		Namespace:    "kube-public",
	},
	{
		ResourceName: "config-images",
		Namespace:    "gmp-system",
	},
	{
		ResourceName: "webhook-ca",
		Namespace:    "gmp-system",
	},
	{
		ResourceName: "cluster-autoscaler-status",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "cluster-kubestore",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "clustermetrics",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "gke-common-webhook-heartbeat",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "ingress-uid",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "konnectivity-agent-autoscaler-config",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "kube-dns-autoscaler",
		Namespace:    "kube-system",
	},
	{
		ResourceName: "kubedns-config-images",
		Namespace:    "kube-system",
	},
}

func retrieveUsedCM(clientset kubernetes.Interface, namespace string) ([]string, []string, []string, []string, []string, error) {
	var volumesCM []string
	var envCM []string
	var envFromCM []string
	var envFromContainerCM []string
	var envFromInitContainerCM []string

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.ConfigMap != nil {
				volumesCM = append(volumesCM, volume.ConfigMap.Name)
			}
			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.ConfigMap != nil {
						volumesCM = append(volumesCM, source.ConfigMap.Name)
					}
				}
			}
		}
		for _, container := range pod.Spec.Containers {
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					envCM = append(envCM, env.ValueFrom.ConfigMapKeyRef.Name)
				}
			}
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					envFromCM = append(envFromCM, envFrom.ConfigMapRef.Name)
				}
			}
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					envFromContainerCM = append(envFromContainerCM, envFrom.ConfigMapRef.Name)
				}
			}
		}
		for _, initContainer := range pod.Spec.InitContainers {
			for _, volume := range initContainer.VolumeMounts {
				if volume.Name != "" && volume.MountPath != "" {
					volumesCM = append(volumesCM, volume.Name)
				}
			}
			for _, env := range initContainer.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					envFromInitContainerCM = append(envFromInitContainerCM, env.ValueFrom.ConfigMapKeyRef.Name)
				}
			}
		}
	}

	for _, resource := range exceptionConfigMaps {
		if resource.Namespace == namespace || resource.Namespace == "*" {
			volumesCM = append(volumesCM, resource.ResourceName)
		}
	}

	return volumesCM, envCM, envFromCM, envFromContainerCM, envFromInitContainerCM, nil
}

func retrieveConfigMapNames(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, []string, error) {
	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, nil, err
	}

	var unusedConfigmapNames []string
	names := make([]string, 0, len(configmaps.Items))

	for _, configmap := range configmaps.Items {
		if pass, _ := filter.SetObject(&configmap).Run(filterOpts); pass {
			continue
		}

		if configmap.Labels["kor/used"] == "false" {
			unusedConfigmapNames = append(unusedConfigmapNames, configmap.Name)
			continue
		}

		names = append(names, configmap.Name)
	}
	return names, unusedConfigmapNames, nil
}

func processNamespaceCM(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	volumesCM, envCM, envFromCM, envFromContainerCM, envFromInitContainerCM, err := retrieveUsedCM(clientset, namespace)
	if err != nil {
		return nil, err
	}

	volumesCM = RemoveDuplicatesAndSort(volumesCM)
	envCM = RemoveDuplicatesAndSort(envCM)
	envFromCM = RemoveDuplicatesAndSort(envFromCM)
	envFromContainerCM = RemoveDuplicatesAndSort(envFromContainerCM)
	envFromInitContainerCM = RemoveDuplicatesAndSort(envFromInitContainerCM)

	configMapNames, unusedConfigmapNames, err := retrieveConfigMapNames(clientset, namespace, filterOpts)
	if err != nil {
		return nil, err
	}

	var usedConfigMaps []string
	slicesToAppend := [][]string{
		volumesCM,
		envCM,
		envFromCM,
		envFromContainerCM,
		envFromInitContainerCM,
	}

	for _, slice := range slicesToAppend {
		usedConfigMaps = append(usedConfigMaps, slice...)
	}

	diff := CalculateResourceDifference(usedConfigMaps, configMapNames)
	diff = append(diff, unusedConfigmapNames...)

	return diff, nil

}

func GetUnusedConfigmaps(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	response := make(map[string]map[string][]string)

	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceCM(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "ConfigMap", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete ConfigMap %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "Configmaps", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["ConfigMap"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedCMs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedCMs, nil
}
