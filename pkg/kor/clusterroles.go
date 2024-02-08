package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/yonahd/kor/pkg/filters"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func retrieveUsedClusterRoles(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	// Get a list of all role bindings in the specified namespace
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings in namespace %s: %v", namespace, err)
	}

	usedClusterRoles := make(map[string]bool)

	for _, rb := range roleBindings.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}
		usedClusterRoles[rb.RoleRef.Name] = true
		if rb.RoleRef.Kind == "ClusterRole" {
			usedClusterRoles[rb.RoleRef.Name] = true
		}
	}

	// Get a list of all cluster role bindings in the specified namespace
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})

	for _, crb := range clusterRoleBindings.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}
		usedClusterRoles[crb.RoleRef.Name] = true

		usedClusterRoles[crb.RoleRef.Name] = true
	}

	var usedClusterRoleNames []string
	for role := range usedClusterRoles {
		usedClusterRoleNames = append(usedClusterRoleNames, role)
	}

	return usedClusterRoleNames, nil
}

func retrieveClusterRoleNames(clientset kubernetes.Interface) ([]string, error) {
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(clusterRoles.Items))
	for _, clusterRole := range clusterRoles.Items {
		if clusterRole.Labels["kor/used"] == "true" {
			continue
		}

		names = append(names, clusterRole.Name)
	}
	return names, nil
}

// func processNamespaceRoles(clientset kubernetes.Interface, namespace string, filterOpts *FilterOptions) ([]string, error) {
func processNamespaceClusterRoles(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	usedClusterRoles, err := retrieveUsedClusterRoles(clientset, namespace, filterOpts)
	if err != nil {
		return nil, err
	}

	usedClusterRoles = RemoveDuplicatesAndSort(usedClusterRoles)

	clusterRoleNames, err := retrieveClusterRoleNames(clientset)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedClusterRoles, clusterRoleNames)
	return diff, nil

}

func GetUnusedClusterRoles(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := filterOpts.Namespaces(clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceClusterRoles(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "ClusterRole", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Role %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "ClusterRoles", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["ClusterRoles"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedClusterRoles, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedClusterRoles, nil
}
