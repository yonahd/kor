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
	"sigs.k8s.io/yaml"
)

func processCrds(apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface) ([]string, error) {

	var unusedCRDs []string

	crds, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, crd := range crds.Items {
		if crd.Labels["kor/used"] == "true" {
			continue
		}

		gvr := schema.GroupVersionResource{
			Group:    crd.Spec.Group,
			Version:  crd.Spec.Versions[0].Name, // We're checking the first version.
			Resource: crd.Spec.Names.Plural,
		}
		instances, err := dynamicClient.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		if len(instances.Items) == 0 {
			unusedCRDs = append(unusedCRDs, crd.Name)
		}
	}
	return unusedCRDs, nil
}

func GetUnusedCrds(apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, slackOpts SlackOpts) {

	var outputBuffer bytes.Buffer
	diff, err := processCrds(apiExtClient, dynamicClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process crds %v\n", err)
	}
	output := FormatOutput("", diff, "Crds")

	outputBuffer.WriteString(output)
	outputBuffer.WriteString("\n")

	if slackOpts != (SlackOpts{}) {
		if err := SendToSlack(SlackMessage{}, slackOpts, outputBuffer.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send message to slack: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(outputBuffer.String())
	}
}

func GetUnusedCrdsStructured(apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string) (string, error) {
	response := make(map[string]map[string][]string)

	diff, err := processCrds(apiExtClient, dynamicClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process crds: %v\n", err)
	}
	if len(diff) > 0 {
		// We consider cluster scope resources in "" (empty string) namesapce, as it is common in k8s
		if response[""] == nil {
			response[""] = make(map[string][]string)
		}
		response[""]["Crd"] = diff
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	if outputFormat == "yaml" {
		yamlResponse, err := yaml.JSONToYAML(jsonResponse)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		return string(yamlResponse), nil
	} else {
		return string(jsonResponse), nil
	}
}
