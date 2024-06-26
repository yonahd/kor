package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/roles/roles.json
var rolesConfig []byte

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

func retrieveRoleNames(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, []string, error) {
	roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, nil, err
	}

	config, err := unmarshalConfig(rolesConfig)
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

		exceptionFound, err := isResourceException(role.Name, role.Namespace, config.ExceptionRoles)
		if err != nil {
			return nil, nil, err
		}

		if exceptionFound {
			continue
		}
		names = append(names, role.Name)
	}
	return names, unusedRoleNames, nil
}

func processNamespaceRoles(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	usedRoles, err := retrieveUsedRoles(clientset, namespace)
	if err != nil {
		return nil, err
	}

	usedRoles = RemoveDuplicatesAndSort(usedRoles)

	roleInfos, rolesUnusedFromLabel, err := retrieveRoleNames(clientset, namespace, filterOpts)
	if err != nil {
		return nil, err
	}

	var diff []ResourceInfo

	for _, name := range CalculateResourceDifference(usedRoles, roleInfos) {
		reason := "ServiceAccount is not in use"
		diff = append(diff, ResourceInfo{Name: name, Reason: reason})
	}

	for _, name := range rolesUnusedFromLabel {
		reason := "Marked with unused label"
		diff = append(diff, ResourceInfo{Name: name, Reason: reason})
	}

	return diff, nil
}

func GetUnusedRoles(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
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
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Role"] = diff
		case "resource":
			appendResources(resources, "Role", namespace, diff)
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

	unusedRoles, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedRoles, nil
}
