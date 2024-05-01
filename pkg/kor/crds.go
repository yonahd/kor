package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/filters"
)

var exceptionCrds = []ExceptionResource{
	{
		ResourceName: "capacityrequests.internal.autoscaling.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "clusterpodmonitorings.monitoring.googleapis.com",
		Namespace:    "",
	},
	{
		ResourceName: "clusterrules.monitoring.googleapis.com ",
		Namespace:    "",
	},
	{
		ResourceName: "frontendconfigs.networking.gke.io ",
		Namespace:    "",
	},
	{
		ResourceName: "gkenetworkparamsets.networking.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "globalrules.monitoring.googleapis.com",
		Namespace:    "",
	},
	{
		ResourceName: "managedcertificates.networking.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "memberships.hub.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "networks.networking.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "podmonitorings.monitoring.googleapis.com",
		Namespace:    "",
	},
	{
		ResourceName: "provisioningrequests.autoscaling.x-k8s.io",
		Namespace:    "",
	},
	{
		ResourceName: "rules.monitoring.googleapis.com",
		Namespace:    "",
	},
	{
		ResourceName: "serviceattachments.networking.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "servicenetworkendpointgroups.networking.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "updateinfos.nodemanagement.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "volumesnapshotclasses.snapshot.storage.k8s.io",
		Namespace:    "",
	},
	{
		ResourceName: "volumesnapshotcontents.snapshot.storage.k8s.io",
		Namespace:    "",
	},
	{
		ResourceName: "volumesnapshots.snapshot.storage.k8s.io",
		Namespace:    "",
	},
	{
		ResourceName: "allowlistedv2workloads.auto.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "allowlistedworkloads.auto.gke.io",
		Namespace:    "",
	},
	{
		ResourceName: "backendconfigs.cloud.google.com",
		Namespace:    "",
	},
	{
		ResourceName: "clusterrules.monitoring.googleapis.com",
		Namespace:    "",
	},
	{
		ResourceName: "frontendconfigs.networking.gke.io",
		Namespace:    "",
	},
}

func processCrds(apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, filterOpts *filters.Options) ([]string, error) {

	var unusedCRDs []string

	crds, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	for _, crd := range crds.Items {
		if pass := filters.KorLabelFilter(&crd, &filters.Options{}); pass {
			continue
		}
		if isResourceException(crd.Name, crd.Namespace, exceptionCrds) {
			continue
		}

		gvr := schema.GroupVersionResource{
			Group:    crd.Spec.Group,
			Version:  crd.Spec.Versions[0].Name, // We're checking the first version.
			Resource: crd.Spec.Names.Plural,
		}
		instances, err := dynamicClient.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
		if err != nil {
			return nil, err
		}
		if len(instances.Items) == 0 {
			unusedCRDs = append(unusedCRDs, crd.Name)
		}
	}
	return unusedCRDs, nil
}

func GetUnusedCrds(filterOpts *filters.Options, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts Opts) (string, error) {

	var outputBuffer bytes.Buffer
	diff, err := processCrds(apiExtClient, dynamicClient, &filters.Options{})

	response := make(map[string]map[string][]string)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process crds: %v\n", err)
	}
	if len(diff) > 0 {
		// We consider cluster scope resources in "" (empty string) namespace, as it is common in k8s
		if response[""] == nil {
			response[""] = make(map[string][]string)
		}
		response[""]["Crd"] = diff
	}
	output := FormatOutput("", diff, "Crds", opts)
	if output != "" {
		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedCRDs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedCRDs, nil
}
