package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/utils/strings/slices"

	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/clusterroles/clusterroles.json
var clusterRolesConfig []byte

func retrieveUsedClusterRoles(clientset kubernetes.Interface, filterOpts *filters.Options) ([]string, error) {

	//Get a list of all namespaces
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve namespaces: %v\n", err)
		os.Exit(1)
	}
	roleBindingsAllNameSpaces := make([]v1.RoleBinding, 0)

	for _, ns := range namespaceList.Items {
		// Get a list of all role bindings in the specified namespace
		roleBindings, err := clientset.RbacV1().RoleBindings(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list role bindings in namespace %s: %v", ns.Name, err)
		}

		roleBindingsAllNameSpaces = append(roleBindingsAllNameSpaces, roleBindings.Items...)
	}

	usedClusterRoles := make(map[string]bool)

	for _, rb := range roleBindingsAllNameSpaces {
		usedClusterRoles[rb.RoleRef.Name] = true
		if rb.RoleRef.Kind == "ClusterRole" {
			usedClusterRoles[rb.RoleRef.Name] = true
		}
	}

	// Get a list of all cluster role bindings in the specified namespace
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to list cluster role bindings %v", err)
	}

	for _, crb := range clusterRoleBindings.Items {
		usedClusterRoles[crb.RoleRef.Name] = true
	}

	// Get a list of all ClusterRoles
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster roles %v", err)
	}
	// Convert the ClusterRole list into a Map
	clusterRolesMap := make(map[string]v1.ClusterRole)
	for _, clusterRole := range clusterRoles.Items {
		clusterRolesMap[clusterRole.Name] = clusterRole
	}
	// Create a list wich holds all aggregated labels
	aggregatedLabels := make([]string, 0)

	for clusterRole := range usedClusterRoles {
		clusterRoleManifest := clusterRolesMap[clusterRole]
		if clusterRolesMap[clusterRole].AggregationRule == nil {
			continue
		}
		for _, label := range clusterRoleManifest.AggregationRule.ClusterRoleSelectors {
			for key, value := range label.MatchLabels {
				aggregatedLabels = append(aggregatedLabels, fmt.Sprintf("%s: %s", key, value))
			}
		}

		for _, clusterRole := range clusterRoles.Items {
			for label, value := range clusterRole.Labels {
				if slices.Contains(aggregatedLabels, label+": "+value) {
					usedClusterRoles[clusterRole.Name], err = strconv.ParseBool(value)
					if err != nil {
						return nil, fmt.Errorf("couldn't convert string to bool %v", err)
					}
					if clusterRole.AggregationRule == nil {
						continue
					}
					for _, label := range clusterRole.AggregationRule.ClusterRoleSelectors {
						for key, value := range label.MatchLabels {
							aggregatedLabels = append(aggregatedLabels, key+": "+value)
						}
					}
				}
			}
		}
	}
	var usedClusterRoleNames []string
	for role := range usedClusterRoles {
		usedClusterRoleNames = append(usedClusterRoleNames, role)
	}

	return usedClusterRoleNames, nil
}

func retrieveClusterRoleNames(clientset kubernetes.Interface, filterOpts *filters.Options) ([]string, []string, error) {
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	config, err := unmarshalConfig(clusterRolesConfig)
	if err != nil {
		return nil, nil, err
	}

	var unusedClusterRoles []string
	names := make([]string, 0, len(clusterRoles.Items))

	for _, clusterRole := range clusterRoles.Items {
		if pass, _ := filter.SetObject(&clusterRole).Run(filterOpts); pass {
			continue
		}

		if clusterRole.Labels["kor/used"] == "false" {
			unusedClusterRoles = append(unusedClusterRoles, clusterRole.Name)
			continue
		}

		exceptionFound, err := isResourceException(clusterRole.Name, clusterRole.Namespace, config.ExceptionClusterRoles)
		if err != nil {
			return nil, nil, err
		}

		if exceptionFound {
			continue
		}

		names = append(names, clusterRole.Name)
	}
	return names, unusedClusterRoles, nil
}

func processClusterRoles(clientset kubernetes.Interface, filterOpts *filters.Options) ([]ResourceInfo, error) {
	usedClusterRoles, err := retrieveUsedClusterRoles(clientset, filterOpts)
	if err != nil {
		return nil, err
	}

	usedClusterRoles = RemoveDuplicatesAndSort(usedClusterRoles)

	clusterRoleNames, unusedClusterRoles, err := retrieveClusterRoleNames(clientset, filterOpts)
	if err != nil {
		return nil, err
	}

	var diff []ResourceInfo

	for _, name := range CalculateResourceDifference(usedClusterRoles, clusterRoleNames) {
		reason := "ClusterRole is not used by any RoleBinding or ClusterRoleBinding"
		diff = append(diff, ResourceInfo{Name: name, Reason: reason})
	}

	for _, name := range unusedClusterRoles {
		reason := "Marked with unused label"
		diff = append(diff, ResourceInfo{Name: name, Reason: reason})
	}

	return diff, nil

}

func GetUnusedClusterRoles(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processClusterRoles(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process cluster role : %v\n", err)
	}
	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, "", "ClusterRole", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete clusterRole %s : %v\n", diff, err)
		}
	}
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["ClusterRole"] = diff
	case "resource":
		appendResources(resources, "ClusterRole", "", diff)
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

	unusedClusterRoles, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedClusterRoles, nil
}
