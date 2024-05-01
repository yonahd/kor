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

	"github.com/yonahd/kor/pkg/filters"
)

var exceptionRoles = []ExceptionResource{
	{
		ResourceName: "cloud-provider",
		Namespace:    "",
	},
	{
		ResourceName: "system:controller:glbc",
		Namespace:    "",
	},
}

func retrieveUsedRoles(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	// Get a list of all role bindings in the specified namespace
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings in namespace %s: %v", namespace, err)
	}

	usedRoles := make(map[string]bool)
	for _, rb := range roleBindings.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}

		usedRoles[rb.RoleRef.Name] = true
	}

	var usedRoleNames []string
	for role := range usedRoles {
		usedRoleNames = append(usedRoleNames, role)
	}

	return usedRoleNames, nil
}

func retrieveRoleNames(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, []string, error) {
	roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, nil, err
	}
	var unusedRoleNames []string
	names := make([]string, 0, len(roles.Items))
	for _, role := range roles.Items {
		if pass := filters.KorLabelFilter(&role, &filters.Options{}); pass {
			continue
		}
		if role.Labels["kor/used"] == "false" {
			unusedRoleNames = append(unusedRoleNames, role.Name)
			continue
		}

		if isResourceException(role.Name, "", exceptionRoles) {
			continue
		}

		names = append(names, role.Name)
	}
	return names, unusedRoleNames, nil
}

func processNamespaceRoles(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	usedRoles, err := retrieveUsedRoles(clientset, namespace, filterOpts)
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

func GetUnusedRoles(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer
	namespaces := filterOpts.Namespaces(clientset)
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
