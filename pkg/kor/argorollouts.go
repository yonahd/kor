package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func processNamespaceArgoRollouts(clientset kubernetes.Interface, clientsetrollout versioned.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	argoRolloutList, err := clientsetrollout.ArgoprojV1alpha1().Rollouts(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})

	if err != nil {
		return nil, err
	}

	var argoRolloutWithoutReplicas []ResourceInfo

	for _, argoRollout := range argoRolloutList.Items {
		if pass, _ := filter.SetObject(&argoRollout).Run(filterOpts); pass {
			continue
		}
		if argoRollout.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			argoRolloutWithoutReplicas = append(argoRolloutWithoutReplicas, ResourceInfo{Name: argoRollout.Name, Reason: reason})
			continue
		}
		deploymentWorkLoadRef := argoRollout.Spec.WorkloadRef

		if deploymentWorkLoadRef.Kind == "Deployment" {
			deploymentItem, _ := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentWorkLoadRef.Name, metav1.GetOptions{})

			if deploymentItem.GetName() == "" {
				reason := "Rollout has no deployments"
				argoRolloutWithoutReplicas = append(argoRolloutWithoutReplicas, ResourceInfo{Name: argoRollout.Name, Reason: reason})
			}
		}
	}

	return argoRolloutWithoutReplicas, nil
}

func GetUnusedArgoRollouts(filterOpts *filters.Options, clientset kubernetes.Interface, clientsetargorollouts versioned.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceArgoRollouts(clientset, clientsetargorollouts, namespace, filterOpts)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if opts.DeleteFlag {
			if diff, err = DeleteArgoRolloutsResource(diff, clientsetargorollouts, namespace, "ArgoRollout", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete ArgoRollout %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["ArgoRollout"] = diff
		case "resource":
			appendResources(resources, "ArgoRollout", namespace, diff)
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

	unusedDeployments, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedDeployments, nil
}
