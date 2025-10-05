package kor

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/strings/slices"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

type NamespacedResource struct {
	Identifier types.NamespacedName
	GVR        schema.GroupVersionResource
}

var (
	//go:embed all:exceptions
	exceptionsFS embed.FS

	//go:embed exceptions/namespaces/namespaces.json
	namespacesConfig []byte

	Exclusions                    []ExceptionNamespacedResource
	AllNamespacedAPIResourceTypes []*metav1.APIResourceList
	NamespacedExceptionsConfig    *Config
)

func isNamespacedResourceType(resourceType string) bool {
	for _, apiList := range AllNamespacedAPIResourceTypes {
		if apiList == nil {
			continue
		}
		for _, r := range apiList.APIResources {
			if r.Name == resourceType || r.SingularName == resourceType || slices.Contains(r.ShortNames, resourceType) || strings.EqualFold(r.Kind, resourceType) {
				return r.Namespaced
			}
		}
	}

	return false
}

func processNamespaces(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface, filterOpts *filters.Options) ([]ResourceInfo, error) {
	var unusedNamespaces []ResourceInfo

	filteredNamespaceNames := filterOpts.Namespaces(clientset)

	config, err := unmarshalConfig(namespacesConfig)
	if err != nil {
		return nil, err
	}

	for _, namespaceName := range filteredNamespaceNames {
		namespace, err := clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if pass, _ := filter.SetObject(namespace).Run(filterOpts); pass {
			continue
		}

		// ignore namespaces within exception list
		exceptionFound, err := isResourceException("", namespace.Name, config.ExceptionNamespaces)
		if err != nil {
			return nil, err
		}
		if exceptionFound {
			continue
		}

		// skipping user labeled resources
		if namespace.Labels["kor/used"] == "false" && !exceptionFound {
			unusedNamespaces = append(
				unusedNamespaces,
				ResourceInfo{Name: namespace.Name, Reason: "Marked with unused label"},
			)
			continue
		}

		// skipping default resources here
		resourceFound, err := isNamespaceUsed(ctx, clientset, dynamicClient, namespaceName, filterOpts)
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

func getGVR(groupVersion string, name string) (*schema.GroupVersionResource, error) {
	splitGV := strings.Split(groupVersion, "/")
	if groupVersion == "" {
		splitGV = []string{}
	}
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
		return nil, fmt.Errorf("GroupVersion can only be sliced to 1 or 2 parts, got: %d", NumberOfGVPartsFound)
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

func exceptionsFSWalkFn(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return nil
	}

	if d.IsDir() {
		return nil
	}

	// match .json suffix (case-insensitive)
	if strings.EqualFold(filepath.Ext(path), ".json") {
		b, err := exceptionsFS.ReadFile(path)
		if err != nil {
			return nil
		}

		// Decode top-level object
		var top map[string]json.RawMessage
		if err := json.Unmarshal(b, &top); err != nil {
			return nil
		}

		// skip non-namespaced resource types
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if !isNamespacedResourceType(base) {
			return nil
		}

		// Require exactly one key
		if len(top) != 1 {
			return nil
		}

		// The value under that key must be an array of ExceptionNamespacedResource
		var key string
		for k := range top {
			key = k
			break
		}
		var items []ExceptionNamespacedResource
		if err := json.Unmarshal(top[key], &items); err != nil {
			return nil
		}
		if len(items) == 0 {
			return nil
		}

		// Keep only items where MatchRegex == true
		var kept []ExceptionNamespacedResource
		for _, it := range items {
			if it.MatchRegex {
				kept = append(kept, it)
			}
		}
		if len(kept) == 0 {
			return nil
		}
		// Set ResourceType for each kept item from the file base name
		for i := range kept {
			kept[i].ResourceType = base
		}
		Exclusions = append(Exclusions, kept...)
	}
	return nil
}

func loadNamespacedEmbeddedExceptions(
	ctx context.Context,
	fileSystem embed.FS,
) (*Config, error) {
	if err := fs.WalkDir(exceptionsFS, ".", exceptionsFSWalkFn); err != nil {
		return nil, err
	}
	// always exclude events
	Exclusions = append(Exclusions, ExceptionNamespacedResource{
		Namespace:    ".*",
		ResourceName: ".*",
		ResourceType: "events",
		MatchRegex:   true,
	},
	)
	config := &Config{
		ExceptionNamespacedResources: Exclusions,
	}
	return config, nil
}

func isNamespaceUsed(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface, namespace string, filterOpts *filters.Options) (bool, error) {
	config := NamespacedExceptionsConfig
	apiResourceLists, err := clientset.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		return true, err
	}

	// Iterate over all API resources and list instances of each in the specified namespace
	for _, apiResourceList := range apiResourceLists {
		for _, apiResource := range apiResourceList.APIResources {
			gvr, err := getGVR(apiResourceList.GroupVersion, apiResource.Name)
			if err != nil {
				return true, err
			}

			resourcesInNamespace, err := dynamicClient.Resource(*gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}

			// check if Namespace is Not Empty
			for _, resourceInNamespace := range resourcesInNamespace.Items {
				resource := NamespacedResource{
					GVR: *gvr,
					Identifier: types.NamespacedName{
						Namespace: resourceInNamespace.GetNamespace(),
						Name:      resourceInNamespace.GetName(),
					},
				}

				// User specified resource type ignore list
				if ignoreResourceType(resource.GVR.Resource, filterOpts.IgnoreResourceTypes) {
					continue
				}

				// ignore namespaced resources within exception list
				exceptionFound, err := isNamespacedResourceException(resource.Identifier.Name, resource.Identifier.Namespace, resource.GVR.Resource, config.ExceptionNamespacedResources)
				if err != nil {
					return true, err
				}
				if exceptionFound {
					continue
				}

				return true, nil
			}
		}
	}
	return false, nil
}

func GetUnusedNamespaces(ctx context.Context, filterOpts *filters.Options, clientset kubernetes.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	// TODO: EZ: func GetUnusedNamespaces(ctx context.Context, filterOpts *filters.Options, clientset kubernetes.Interface, dynamicClient dynamic.Interface, discoveryClient discovery.DiscoveryInterface, outputFormat string, opts common.Opts) (string, error) {
	allNamespacedAPIs, err := clientset.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error fetching preferred resources (partial results may be available): %v", err)
	}
	AllNamespacedAPIResourceTypes = allNamespacedAPIs

	NamespacedExceptionsConfig, err = loadNamespacedEmbeddedExceptions(ctx, exceptionsFS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process namespaced exceptions: %v\n", err)
	}

	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processNamespaces(ctx, clientset, dynamicClient, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process namespaces: %v\n", err)
	}

	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["Namespace"] = diff
	case "resource":
		appendResources(resources, "Namespace", "", diff)
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
