package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/crds/crds.json
var crdsConfig []byte
var servedVersions []string

func processCrds(apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, filterOpts *filters.Options) ([]ResourceInfo, error) {

	var unusedCRDs []ResourceInfo

	crds, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(crdsConfig)
	if err != nil {
		return nil, err
	}

	for _, crd := range crds.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(crd.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&crd).Run(filterOpts); pass {
			continue
		}

		if crd.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedCRDs = append(unusedCRDs, ResourceInfo{Name: crd.Name, Reason: reason})
			continue
		}

		exceptionFound, err := isResourceException(crd.Name, crd.Namespace, config.ExceptionCrds)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}
		
		for _, v := range crd.Spec.Versions {
			if v.Served {
				servedVersions = append(servedVersions, v.Name)
			}
		}

		// Skip CRD if no served versions are found
		if len(servedVersions) == 0 {
			continue
		}

		hasInstances := false
		for _, version := range servedVersions {
			gvr := schema.GroupVersionResource{
				Group:    crd.Spec.Group,
				Version:  version,
				Resource: crd.Spec.Names.Plural,
			}

			instances, err := dynamicClient.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
			if err != nil {
				// Skip versions that fail to list
				continue
			}
			if len(instances.Items) > 0 {
				hasInstances = true
				break
			}
		}

		if !hasInstances {
			reason := "CRD has no instances in any served version"
			unusedCRDs = append(unusedCRDs, ResourceInfo{Name: crd.Name, Reason: reason})
		}

	}
	return unusedCRDs, nil
}

func GetUnusedCrds(_ *filters.Options, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processCrds(apiExtClient, dynamicClient, &filters.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process crds: %v\n", err)
	}
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["Crd"] = diff
	case "resource":
		appendResources(resources, "Crd", "", diff)
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

	unusedCRDs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedCRDs, nil
}
