package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/rolebindings/rolebindings.json
var roleBindingsConfig []byte

// Filter out subjects base on Kind, can be later used for User and Group
func filterSubjects(subjects []v1.Subject, kind string) []v1.Subject {
	var serviceAccountSubjects []v1.Subject
	for _, subject := range subjects {
		if subject.Kind == kind {
			serviceAccountSubjects = append(serviceAccountSubjects, subject)
		}
	}
	return serviceAccountSubjects
}

// Check if any valid service accounts exist in the RoleBinding
func isUsingValidServiceAccount(serviceAccounts []v1.Subject, serviceAccountNames map[string]bool) bool {
	for _, sa := range serviceAccounts {
		if serviceAccountNames[sa.Name] {
			return true
		}
	}
	return false
}

func validateRoleReference(rb v1.RoleBinding, roleNames, clusterRoleNames map[string]bool) *ResourceInfo {
	if rb.RoleRef.Kind == "Role" && !roleNames[rb.RoleRef.Name] {
		return &ResourceInfo{Name: rb.Name, Reason: "RoleBinding references a non-existing Role"}
	}

	if rb.RoleRef.Kind == "ClusterRole" && !clusterRoleNames[rb.RoleRef.Name] {
		return &ResourceInfo{Name: rb.Name, Reason: "RoleBinding references a non-existing ClusterRole"}
	}

	return nil
}

func processNamespaceRoleBindings(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	roleBindingsList, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	roleNames, err := convertNamesToPresenseMap(retrieveRoleNames(clientset, namespace, filterOpts))
	if err != nil {
		return nil, err
	}

	clusterRoleNames, err := convertNamesToPresenseMap(retrieveClusterRoleNames(clientset, filterOpts))
	if err != nil {
		return nil, err
	}

	serviceAccountNames, err := convertNamesToPresenseMap(retrieveServiceAccountNames(clientset, namespace, filterOpts))
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(roleBindingsConfig)
	if err != nil {
		return nil, err
	}

	var unusedRoleBindingNames []ResourceInfo

	for _, rb := range roleBindingsList.Items {
		if pass, _ := filter.SetObject(&rb).Run(filterOpts); pass {
			continue
		}

		if rb.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedRoleBindingNames = append(unusedRoleBindingNames, ResourceInfo{Name: rb.Name, Reason: reason})
			continue
		}

		if exceptionFound, err := isResourceException(rb.Name, rb.Namespace, config.ExceptionRoleBindings); err != nil {
			return nil, err
		} else if exceptionFound {
			continue
		}

		roleReferenceIssue := validateRoleReference(rb, roleNames, clusterRoleNames)
		if roleReferenceIssue != nil {
			unusedRoleBindingNames = append(unusedRoleBindingNames, *roleReferenceIssue)
			continue
		}

		serviceAccountSubjects := filterSubjects(rb.Subjects, "ServiceAccount")

		// If other kinds (Users/Groups) are used, we assume they exists for now
		if len(serviceAccountSubjects) != len(rb.Subjects) {
			continue
		}

		// Check if RoleBinding uses a valid service account
		if !isUsingValidServiceAccount(serviceAccountSubjects, serviceAccountNames) {
			unusedRoleBindingNames = append(unusedRoleBindingNames, ResourceInfo{Name: rb.Name, Reason: "RoleBinding references a non-existing ServiceAccount"})
		}
	}

	return unusedRoleBindingNames, nil
}

func GetUnusedRoleBindings(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceRoleBindings(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}

		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "RoleBinding", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete RoleBinding %s in namespace %s: %v\n", diff, namespace, err)
			}
		}

		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["RoleBinding"] = diff
		case "resource":
			appendResources(resources, "RoleBinding", namespace, diff)
		}
	}

	var outputBuffer bytes.Buffer
	var jsonResponse []byte
	switch outputFormat {
	case "table":
		outputBuffer = FormatOutput(resources, opts)
	case "json", "yaml":
		var err error
		if jsonResponse, err = json.MarshalIndent(resources, "", " "); err != nil {
			return "", err
		}
	}

	unusedRoleBindings, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedRoleBindings, nil
}
