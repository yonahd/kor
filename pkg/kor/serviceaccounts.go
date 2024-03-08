package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/filters"
)

var exceptionServiceAccounts = []ExceptionResource{
	{ResourceName: "default", Namespace: "*"},
}

func getServiceAccountsFromClusterRoleBindings(clientset kubernetes.Interface, namespace string) ([]string, error) {
	// Get a list of all role bindings in the specified namespace
	roleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings in namespace %s: %v", namespace, err)
	}

	// Create a slice to store service account names
	var serviceAccounts []string

	// Extract service account names from the role bindings
	for _, rb := range roleBindings.Items {
		if pass := filters.KorLabelFilter(&rb, &filters.Options{}); pass {
			continue
		}

		for _, subject := range rb.Subjects {

			if subject.Kind == "ServiceAccount" {
				serviceAccounts = append(serviceAccounts, subject.Name)
			}
		}
	}

	return serviceAccounts, nil
}

func getServiceAccountsFromRoleBindings(clientset kubernetes.Interface, namespace string) ([]string, error) {
	// Get a list of all role bindings in the specified namespace
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings in namespace %s: %v", namespace, err)
	}

	// Create a slice to store service account names
	var serviceAccounts []string

	// Extract service account names from the role bindings
	for _, rb := range roleBindings.Items {
		if rb.Labels["kor/used"] == "true" {
			continue
		}

		for _, subject := range rb.Subjects {
			if subject.Kind == "ServiceAccount" {
				serviceAccounts = append(serviceAccounts, subject.Name)
			}
		}
	}

	return serviceAccounts, nil
}

func retrieveUsedSA(clientset kubernetes.Interface, namespace string) ([]string, []string, []string, error) {

	var podServiceAccounts []string

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, err
	}

	// Extract service account names from pods
	for _, pod := range pods.Items {
		if pod.Spec.ServiceAccountName != "" {
			podServiceAccounts = append(podServiceAccounts, pod.Spec.ServiceAccountName)
		}
	}

	for _, resource := range exceptionServiceAccounts {
		if resource.Namespace == namespace || resource.Namespace == "*" {
			podServiceAccounts = append(podServiceAccounts, resource.ResourceName)
		}
	}

	roleServiceAccounts, err := getServiceAccountsFromRoleBindings(clientset, namespace)
	if err != nil {
		return nil, nil, nil, err
	}
	clusterRoleServiceAccounts, err := getServiceAccountsFromClusterRoleBindings(clientset, namespace)
	if err != nil {
		return nil, nil, nil, err
	}
	return podServiceAccounts, roleServiceAccounts, clusterRoleServiceAccounts, nil
}

func retrieveServiceAccountNames(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, []string, error) {
	serviceaccounts, err := clientset.CoreV1().ServiceAccounts(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(serviceaccounts.Items))
	var unusedServiceAccountNames []string

	for _, serviceaccount := range serviceaccounts.Items {
		if pass, _ := filter.SetObject(&serviceaccount).Run(filterOpts); pass {
			continue
		}

		if serviceaccount.Labels["kor/used"] == "false" {
			unusedServiceAccountNames = append(unusedServiceAccountNames, serviceaccount.Name)
			continue
		}

		names = append(names, serviceaccount.Name)
	}
	return names, unusedServiceAccountNames, nil
}

func processNamespaceSA(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	usedServiceAccounts, roleServiceAccounts, clusterRoleServiceAccounts, err := retrieveUsedSA(clientset, namespace)
	if err != nil {
		return nil, err
	}

	usedServiceAccounts = RemoveDuplicatesAndSort(usedServiceAccounts)
	roleServiceAccounts = RemoveDuplicatesAndSort(roleServiceAccounts)
	clusterRoleServiceAccounts = RemoveDuplicatesAndSort(clusterRoleServiceAccounts)

	usedServiceAccounts = append(append(usedServiceAccounts, roleServiceAccounts...), clusterRoleServiceAccounts...)

	serviceAccountNames, unusedServiceAccountNames, err := retrieveServiceAccountNames(clientset, namespace, filterOpts)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedServiceAccounts, serviceAccountNames)
	diff = append(diff, unusedServiceAccountNames...)
	return diff, nil

}

func GetUnusedServiceAccounts(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer

	namespaces := filterOpts.Namespaces(clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceSA(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "ServiceAccount", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Serviceaccount %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "Serviceaccounts", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["ServiceAccounts"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedServiceAccounts, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedServiceAccounts, nil
}
