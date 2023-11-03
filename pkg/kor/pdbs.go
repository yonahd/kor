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
)

func processNamespacePdbs(clientset kubernetes.Interface, namespace string) ([]string, error) {
	var unusedPdbs []string
	pdbs, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pdb := range pdbs.Items {
		if pdb.Labels["kor/used"] == "true" {
			continue
		}

		selector := pdb.Spec.Selector
		if len(selector.MatchLabels) == 0 {
			unusedPdbs = append(unusedPdbs, pdb.Name)
			continue
		}
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(selector),
		})
		if err != nil {
			return nil, err
		}
		statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(selector),
		})
		if err != nil {
			return nil, err
		}
		if len(deployments.Items) == 0 && len(statefulSets.Items) == 0 {
			unusedPdbs = append(unusedPdbs, pdb.Name)
		}
	}
	return unusedPdbs, nil
}

func GetUnusedPdbs(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer

	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespacePdbs(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "PDB", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete PDB %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "PDBs")
		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")

		resourceMap := make(map[string][]string)
		resourceMap["Pdb"] = diff
		response[namespace] = resourceMap
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedPdbs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedPdbs, nil
}
