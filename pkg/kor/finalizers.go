package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/utils/strings/slices"

	"github.com/yonahd/kor/pkg/filters"
)

func CheckFinalizers(finalizers []string, deletionTimestamp *metav1.Time) bool {
	if len(finalizers) > 0 && deletionTimestamp != nil {
		return true
	}
	return false
}

func retrievePendingDeletionResources(resourceTypes []*metav1.APIResourceList, dynamicClient dynamic.Interface, filterOpts *filters.Options) (map[string]map[schema.GroupVersionResource][]ResourceInfo, error) {
	pendingDeletionResources := make(map[string]map[schema.GroupVersionResource][]ResourceInfo) //map[namespace]map[gvr][]resourceNames

	for _, apiResourceList := range resourceTypes {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			return pendingDeletionResources, err
		}

		for _, resourceType := range apiResourceList.APIResources {

			if slices.Contains(resourceType.Verbs, "list") {

				gvr := gv.WithResource(resourceType.Name)
				resourceList, err := dynamicClient.
					Resource(gvr).
					Namespace(metav1.NamespaceAll).
					List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
				if err != nil {
					fmt.Printf("Error listing resources for GVR %s: %v\n", apiResourceList.GroupVersion, err)
					continue
				}
				for _, item := range resourceList.Items {
					if pass, _ := filter.SetObject(&item).Run(filterOpts); pass {
						continue
					}
					if CheckFinalizers(item.GetFinalizers(), item.GetDeletionTimestamp()) {
						if pendingDeletionResources[item.GetNamespace()] == nil {
							pendingDeletionResources[item.GetNamespace()] = make(map[schema.GroupVersionResource][]ResourceInfo)
						}
						finalizerInfo := ResourceInfo{
							Name:   item.GetName(),
							Reason: "Pending deletion waiting for finalizers",
						}
						pendingDeletionResources[item.GetNamespace()][gvr] = append(pendingDeletionResources[item.GetNamespace()][gvr], finalizerInfo)
					}
				}
			}
		}
	}
	return pendingDeletionResources, nil
}

func getResourcesWithFinalizersPendingDeletion(clientset kubernetes.Interface, dynamicClient dynamic.Interface, filterOpts *filters.Options) (map[string]map[schema.GroupVersionResource][]ResourceInfo, error) {
	// Use the discovery client to fetch API resources
	resourceTypes, err := clientset.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		fmt.Printf("Error fetching server resources: %v\n", err)
		os.Exit(1)
	}

	return retrievePendingDeletionResources(resourceTypes, dynamicClient, filterOpts)
}

func GetUnusedfinalizers(filterOpts *filters.Options, clientset kubernetes.Interface, dynamicClient *dynamic.DynamicClient, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := filterOpts.Namespaces(clientset)
	response := make(map[string]map[string][]ResourceInfo)
	pendingDeletionDiffs, err := getResourcesWithFinalizersPendingDeletion(clientset, dynamicClient, filterOpts)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process resources waiting for finalizers: %v\n", err)
	}

	allDiffs := make(map[string][]ResourceInfo)

	for namespace, resourceType := range pendingDeletionDiffs {
		if slices.Contains(namespaces, namespace) {
			for gvr, resourceDiff := range resourceType {
				if opts.DeleteFlag {
					if resourceDiff, err = DeleteResourceWithFinalizer(resourceDiff, dynamicClient, namespace, gvr, opts.NoInteractive); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to delete objects waiting for Finalizers %s in namespace %s: %v\n", resourceDiff, namespace, err)
					}
				}
				allDiffs[gvr.Resource] = resourceDiff
			}

			output := formatOutputForNamespace(namespace, allDiffs, opts)
			outputBuffer.WriteString(output)

			response[namespace] = allDiffs
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedFinalizers, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedFinalizers, nil
}
