package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/pdbs/pdbs.json
var pdbsConfig []byte

func processNamespacePdbs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	var unusedPdbs []ResourceInfo
	pdbs, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(pdbsConfig)
	if err != nil {
		return nil, err
	}

	for _, pdb := range pdbs.Items {
		if pass, _ := filter.SetObject(&pdb).Run(filterOpts); pass {
			continue
		}

		exceptionFound, err := isResourceException(pdb.Name, pdb.Namespace, config.ExceptionPdbs)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		if pdb.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedPdbs = append(unusedPdbs, ResourceInfo{Name: pdb.Name, Reason: reason})
			continue
		}

		selector := pdb.Spec.Selector
		var hasMatchingTemplates, hasMatchingWorkloads bool

		// Validate empty selector
		if selector == nil || len(selector.MatchLabels) == 0 {
			hasRunningPods, err := validateRunningPods(clientset, namespace)
			if err != nil {
				return nil, err
			}

			if !hasRunningPods {
				reason := "Pdb matches every pod (empty selector) but 0 pods run"
				unusedPdbs = append(unusedPdbs, ResourceInfo{Name: pdb.Name, Reason: reason})
			}

			continue
		} else {
			hasMatchingTemplates, err = validateMatchingTemplates(clientset, namespace, selector)
			if err != nil {
				return nil, err
			}

			hasMatchingWorkloads, err = validateMatchingWorkloads(clientset, namespace, selector)
			if err != nil {
				return nil, err
			}
		}

		if !hasMatchingTemplates && !hasMatchingWorkloads {
			reason := "Pdb is not referencing any deployments, statefulsets or pods"
			unusedPdbs = append(unusedPdbs, ResourceInfo{Name: pdb.Name, Reason: reason})
		}
	}

	return unusedPdbs, nil
}

func validateRunningPods(clientset kubernetes.Interface, namespace string) (bool, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return false, err
	}

	// Field status.phase=Running can still reference Terminating pods
	// Return true if at least one pod is running
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			return true, nil
		}
	}

	return false, nil
}

func validateMatchingTemplates(clientset kubernetes.Interface, namespace string, selector *metav1.LabelSelector) (bool, error) {
	labelSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, err
	}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, deployment := range deployments.Items {
		deploymentLabels := labels.Set(deployment.Spec.Template.Labels)
		if labelSelector.Matches(deploymentLabels) {
			return true, nil
		}
	}

	statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, statefulSet := range statefulSets.Items {
		statefulSetLabels := labels.Set(statefulSet.Spec.Template.Labels)
		if labelSelector.Matches(statefulSetLabels) {
			return true, nil
		}
	}

	return false, nil
}

func validateMatchingWorkloads(clientset kubernetes.Interface, namespace string, selector *metav1.LabelSelector) (bool, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(selector),
	})
	if err != nil {
		return false, err
	}
	if len(pods.Items) > 0 {
		return true, nil
	}

	return false, nil
}

func GetUnusedPdbs(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespacePdbs(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "PDB", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete PDB %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Pdb"] = diff
		case "resource":
			appendResources(resources, "Pdb", namespace, diff)
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

	unusedPdbs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedPdbs, nil
}
