package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/namespaces/namespaces.json
var namespacesConfig []byte

type NamespacedResource struct {
	Identifier types.NamespacedName
	GVR        schema.GroupVersionResource
}

func processNamespaces(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface, filterOpts *filters.Options) ([]ResourceInfo, error) {
	var unusedNamespaces []ResourceInfo

	filteredNamespaceNames := filterOpts.Namespaces(clientset)

	config, err := unmarshalConfig(namespacesConfig)
	if err != nil {
		return nil, err
	}

	for _, namespaceName := range filteredNamespaceNames {
		namespace, err := clientset.CoreV1().Namespaces().Get(context.TODO(), namespaceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if pass, _ := filter.SetObject(namespace).Run(filterOpts); pass {
			continue
		}

		// ignore namespaces within exception list
		exceptionFound, err := isResourceException(namespace.Name, "", config.ExceptionNamespaces)
		if err != nil {
			return nil, err
		}
		if exceptionFound {
			continue
		}

		// skipping user labeled resources
		if namespace.Labels["kor/used"] == "false" {
			unusedNamespaces = append(
				unusedNamespaces,
				ResourceInfo{Name: namespace.Name, Reason: "Marked with unused label"},
			)
			continue
		}

		// skipping default resources here
		resourceFound, err := isErrorOrNamespaceContainsResources(ctx, clientset, dynamicClient, namespaceName, filterOpts)
		if err != nil {
			return unusedNamespaces, err
		}

		// construct list of unused namespaces here following a set of rules
		if !resourceFound {
			unusedNamespaces = append(
				unusedNamespaces,
				ResourceInfo{Name: namespace.Name, Reason: "Empty namespace"},
			)
		}
	}

	return unusedNamespaces, nil
}

func getGVR(name string, splitGV []string) (*schema.GroupVersionResource, error) {
	switch NumberOfGVPartsFound := len(splitGV); NumberOfGVPartsFound {
	case 1:
		return &schema.GroupVersionResource{
			Version:  splitGV[0],
			Resource: name,
		}, nil
	case 2:
		return &schema.GroupVersionResource{
			Group:    splitGV[0],
			Version:  splitGV[1],
			Resource: name,
		}, nil
	default:
		return nil, fmt.Errorf("gv is wrong length slice: %d", NumberOfGVPartsFound)
	}
}

func ignoreResourceType(resource string, ignoreResourceTypes []string) bool {
	for _, ignoreType := range ignoreResourceTypes {
		if resource == ignoreType {
			return true
		}
	}
	return false
}

// TODO: refactor using exception list
func ignorePredefinedResource(resource NamespacedResource) bool {
	// Specific list of resources to ignore - resources created in all namespaced by default
	if resource.GVR.Resource == "configmaps" && resource.GVR.Version == "v1" && resource.Identifier.Name == "kube-root-ca.crt" {
		return true
	}
	if resource.GVR.Resource == "serviceaccounts" && resource.GVR.Version == "v1" && resource.Identifier.Name == "default" {
		return true
	}
	if resource.GVR.Resource == "events" {
		return true
	}
	return false
}

func isNamespaceNotEmpty(gvr *schema.GroupVersionResource, unstructuredList *unstructured.UnstructuredList, filterOpts *filters.Options) bool {
	for _, unstructuredObj := range unstructuredList.Items {
		resource := NamespacedResource{
			GVR: *gvr,
			Identifier: types.NamespacedName{
				Namespace: unstructuredObj.GetNamespace(),
				Name:      unstructuredObj.GetName(),
			},
		}
		// Ignore default cluster resources
		if ignorePredefinedResource(resource) {
			continue
		}
		// User specified resource type ignore list
		if ignoreResourceType(resource.GVR.Resource, filterOpts.IgnoreResourceTypes) {
			continue
		}
		return true
	}
	return false
}

func isErrorOrNamespaceContainsResources(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface, namespace string, filterOpts *filters.Options) (bool, error) {
	apiResourceLists, err := clientset.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		return true, err
	}

	// Iterate over all API resources and list instances of each in the specified namespace
	for _, apiResourceList := range apiResourceLists {
		for _, apiResource := range apiResourceList.APIResources {
			gv := strings.Split(apiResourceList.GroupVersion, "/")
			gvr, err := getGVR(apiResource.Name, gv)
			if err != nil {
				return true, err
			}

			unstructuredList, err := dynamicClient.Resource(*gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}

			if isNamespaceNotEmpty(gvr, unstructuredList, filterOpts) {
				return true, nil
			}
		}
	}
	return false, nil
}

func GetUnusedNamespaces(ctx context.Context, filterOpts *filters.Options, clientset kubernetes.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processNamespaces(ctx, clientset, dynamicClient, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process namespaces: %v\n", err)
	}

	if len(diff) > 0 {
		// We consider cluster scope resources in "" (empty string) namespace, as it is common in k8s
		if resources[""] == nil {
			resources[""] = make(map[string][]ResourceInfo)
		}
		resources[""]["Namespaces"] = diff
	}

	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, "", "Namespace", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete namespace %s : %v\n", diff, err)
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

	unusedNamespaces, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedNamespaces, nil
}
