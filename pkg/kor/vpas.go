package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

var VpaGVR = schema.GroupVersionResource{
	Group:    "autoscaling.k8s.io",
	Version:  "v1",
	Resource: "verticalpodautoscalers",
}

func isVpaSupported(clientset kubernetes.Interface) bool {
	resources, err := clientset.Discovery().ServerResourcesForGroupVersion("autoscaling.k8s.io/v1")
	if err != nil {
		return false
	}
	return resources != nil
}

func getVpaScaleTargetRef(vpa *unstructured.Unstructured) (kind, name string, err error) {
	targetRef, found, err := unstructured.NestedMap(vpa.Object, "spec", "targetRef")
	if err != nil {
		return "", "", err
	}
	if !found {
		return "", "", fmt.Errorf("vpa %s has no spec.targetRef", vpa.GetName())
	}
	kind, _ = targetRef["kind"].(string)
	name, _ = targetRef["name"].(string)
	if kind == "" || name == "" {
		return "", "", fmt.Errorf("vpa %s has incomplete spec.targetRef", vpa.GetName())
	}
	return kind, name, nil
}

func processNamespaceVpas(clientset kubernetes.Interface, dynamicClient dynamic.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	deploymentNames, err := getDeploymentNames(clientset, namespace)
	if err != nil {
		return nil, err
	}

	statefulsetNames, err := getStatefulSetNames(clientset, namespace)
	if err != nil {
		return nil, err
	}

	vpas, err := dynamicClient.Resource(VpaGVR).Namespace(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedVpas []ResourceInfo
	for _, vpa := range vpas.Items {
		vpaObject := vpa
		if pass, _ := filter.SetObject(&vpaObject).Run(filterOpts); pass {
			continue
		}

		// Skip if resource has owner references and ignore flag is set
		if filterOpts.IgnoreOwnerReferences && len(vpaObject.GetOwnerReferences()) > 0 {
			continue
		}

		if vpaObject.GetLabels()["kor/used"] == "false" {
			unusedVpas = append(unusedVpas, ResourceInfo{Name: vpaObject.GetName(), Reason: "Marked with unused label"})
			continue
		}

		kind, name, err := getVpaScaleTargetRef(&vpaObject)
		if err != nil {
			continue
		}

		switch kind {
		case "Deployment":
			if !slices.Contains(deploymentNames, name) {
				unusedVpas = append(unusedVpas, ResourceInfo{Name: vpaObject.GetName(), Reason: "Scale target Deployment does not exist"})
			}
		case "StatefulSet":
			if !slices.Contains(statefulsetNames, name) {
				unusedVpas = append(unusedVpas, ResourceInfo{Name: vpaObject.GetName(), Reason: "Scale target StatefulSet does not exist"})
			}
		}
	}
	if opts.DeleteFlag {
		if unusedVpas, err = DeleteDynamicResource(unusedVpas, dynamicClient, namespace, VpaGVR, opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete VPA %s in namespace %s: %v\n", unusedVpas, namespace, err)
		}
	}
	return unusedVpas, nil
}

func GetUnusedVpas(filterOpts *filters.Options, clientset kubernetes.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)

	if isVpaSupported(clientset) {
		for _, namespace := range filterOpts.Namespaces(clientset) {
			diff, err := processNamespaceVpas(clientset, dynamicClient, namespace, filterOpts, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
				continue
			}
			switch opts.GroupBy {
			case "namespace":
				resources[namespace] = make(map[string][]ResourceInfo)
				resources[namespace]["Vpa"] = diff
			case "resource":
				appendResources(resources, "Vpa", namespace, diff)
			}
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

	unusedVpas, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedVpas, nil
}
