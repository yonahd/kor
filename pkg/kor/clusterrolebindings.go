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

//go:embed exceptions/clusterrolebindings/clusterrolebindings.json
var clusterRoleBindingsConfig []byte

// Check if any valid service accounts exist in the ClusterRoleBinding
func isUsingValidServiceAccountClusterScoped(serviceAccounts []v1.Subject, allServiceAccountNames map[string]map[string]bool) bool {
	for _, sa := range serviceAccounts {
		// Check if the service account exists in the specified namespace
		if namespaceMap, namespaceExists := allServiceAccountNames[sa.Namespace]; namespaceExists {
			if namespaceMap[sa.Name] {
				return true
			}
		}
	}
	return false
}

func validateClusterRoleReference(crb v1.ClusterRoleBinding, clusterRoleNames map[string]bool) *ResourceInfo {
	if crb.RoleRef.Kind == "ClusterRole" && !clusterRoleNames[crb.RoleRef.Name] {
		return &ResourceInfo{Name: crb.Name, Reason: "ClusterRoleBinding references a non-existing ClusterRole"}
	}

	return nil
}

func processClusterRoleBindings(clientset kubernetes.Interface, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	clusterRoleBindingsList, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	clusterRoleNames, err := convertNamesToPresenseMap(retrieveClusterRoleNames(clientset, filterOpts))
	if err != nil {
		return nil, err
	}

	// Get all service accounts from all namespaces for cluster-scoped checking
	allServiceAccountNames := make(map[string]map[string]bool)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		serviceAccountNames, err := convertNamesToPresenseMap(retrieveServiceAccountNames(clientset, namespace, filterOpts))
		if err != nil {
			return nil, err
		}
		allServiceAccountNames[namespace] = serviceAccountNames
	}

	config, err := unmarshalConfig(clusterRoleBindingsConfig)
	if err != nil {
		return nil, err
	}

	var unusedClusterRoleBindingNames []ResourceInfo

	for _, crb := range clusterRoleBindingsList.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(crb.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&crb).Run(filterOpts); pass {
			continue
		}

		if crb.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedClusterRoleBindingNames = append(unusedClusterRoleBindingNames, ResourceInfo{Name: crb.Name, Reason: reason})
			continue
		}

		if exceptionFound, err := isResourceException(crb.Name, "", config.ExceptionClusterRoleBindings); err != nil {
			return nil, err
		} else if exceptionFound {
			continue
		}

		clusterRoleReferenceIssue := validateClusterRoleReference(crb, clusterRoleNames)
		if clusterRoleReferenceIssue != nil {
			unusedClusterRoleBindingNames = append(unusedClusterRoleBindingNames, *clusterRoleReferenceIssue)
			continue
		}

		serviceAccountSubjects := filterSubjects(crb.Subjects, "ServiceAccount")

		// If other kinds (Users/Groups) are used, we assume they exist for now
		if len(serviceAccountSubjects) != len(crb.Subjects) {
			continue
		}

		// Check if ClusterRoleBinding uses a valid service account
		if !isUsingValidServiceAccountClusterScoped(serviceAccountSubjects, allServiceAccountNames) {
			unusedClusterRoleBindingNames = append(unusedClusterRoleBindingNames, ResourceInfo{Name: crb.Name, Reason: "ClusterRoleBinding references a non-existing ServiceAccount"})
		}
	}
	if opts.DeleteFlag {
		if unusedClusterRoleBindingNames, err = DeleteResource(unusedClusterRoleBindingNames, clientset, "", "ClusterRoleBinding", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete ClusterRoleBinding %s: %v\n", unusedClusterRoleBindingNames, err)
		}
	}
	return unusedClusterRoleBindingNames, nil
}

func GetUnusedClusterRoleBindings(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processClusterRoleBindings(clientset, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process clusterrolebindings: %v\n", err)
	}
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["ClusterRoleBinding"] = diff
	case "resource":
		appendResources(resources, "ClusterRoleBinding", "", diff)
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

	unusedClusterRoleBindings, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedClusterRoleBindings, nil
}