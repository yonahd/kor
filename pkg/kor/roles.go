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

func retrieveUsedRoles(clientset kubernetes.Interface, namespace string) ([]string, error) {
	// Get a list of all role bindings in the specified namespace
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings in namespace %s: %v", namespace, err)
	}

	usedRoles := make(map[string]bool)
	for _, rb := range roleBindings.Items {
		usedRoles[rb.RoleRef.Name] = true
	}

	var usedRoleNames []string
	for role := range usedRoles {
		usedRoleNames = append(usedRoleNames, role)
	}

	return usedRoleNames, nil
}

func retrieveRoleNames(clientset kubernetes.Interface, namespace string, filterOpts *FilterOptions) ([]string, []string, error) {
	roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(roles.Items))
	var unusedRoleNames []string
	for _, role := range roles.Items {
		if value, exists := role.Labels["kor/used"]; exists {
			if value == "true" {
				continue
			} else if value == "false" {
				unusedRoleNames = append(unusedRoleNames, role.Name)
				continue
			}
		}

		// checks if the resource has any labels that match the excluded selector specified in opts.ExcludeLabels.
		// If it does, the resource is skipped.
		if excluded, _ := HasExcludedLabel(role.Labels, filterOpts.ExcludeLabels); excluded {
			continue
		}
		// checks if the resource's age (measured from its last modified time) matches the included criteria
		// specified by the filter options.
		if included, _ := HasIncludedAge(role.CreationTimestamp, filterOpts); !included {
			continue
		}

		names = append(names, role.Name)
	}
	return names, unusedRoleNames, nil
}

func processNamespaceRoles(clientset kubernetes.Interface, namespace string, filterOpts *FilterOptions) ([]string, error) {
	usedRoles, err := retrieveUsedRoles(clientset, namespace)
	if err != nil {
		return nil, err
	}

	usedRoles = RemoveDuplicatesAndSort(usedRoles)

	roleNames, rolesUnusedFromLabel, err := retrieveRoleNames(clientset, namespace, filterOpts)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedRoles, roleNames)
	diff = append(diff, rolesUnusedFromLabel...)
	return diff, nil

}

func GetUnusedRoles(includeExcludeLists IncludeExcludeLists, filterOpts *FilterOptions, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceRoles(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "Role", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Role %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		output := FormatOutput(namespace, diff, "Roles", opts)
		if output != "" {
			outputBuffer.WriteString(output)
			outputBuffer.WriteString("\n")

			resourceMap := make(map[string][]string)
			resourceMap["Roles"] = diff
			response[namespace] = resourceMap
		}
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedRoles, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedRoles, nil
}
