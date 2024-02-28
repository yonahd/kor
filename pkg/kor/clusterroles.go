package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/yonahd/kor/pkg/filters"
	v1 "k8s.io/api/rbac/v1"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

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

	if err != nil {
		return nil, fmt.Errorf("failed to list cluster role bindings %v", err)
	}

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

func retrieveClusterRoleNames(clientset kubernetes.Interface, filterOpts *filters.Options) ([]string, []string, error) {
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
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

		names = append(names, clusterRole.Name)
	}
	return names, unusedClusterRoles, nil
}

func processClusterRoles(clientset kubernetes.Interface, filterOpts *filters.Options) ([]string, error) {
	usedClusterRoles, err := retrieveUsedClusterRoles(clientset, filterOpts)
	if err != nil {
		return nil, err
	}

	usedClusterRoles = RemoveDuplicatesAndSort(usedClusterRoles)

	clusterRoleNames, unusedClusterRoles, err := retrieveClusterRoleNames(clientset, filterOpts)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedClusterRoles, clusterRoleNames)
	diff = append(diff, unusedClusterRoles...)

	return diff, nil

}

func GetUnusedClusterRoles(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer

	response := make(map[string]map[string][]string)

	diff, err := processClusterRoles(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process cluster role : %v\n", err)
	}

	if len(diff) > 0 {
		// We consider cluster scope resources in "" (empty string) namespace, as it is common in k8s
		if response[""] == nil {
			response[""] = make(map[string][]string)
		}
		response[""]["ClusterRoles"] = diff
	}

	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, "", "ClusterRole", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete clusterRole %s : %v\n", diff, err)
		}
	}
	output := FormatOutput("", diff, "ClusterRoles", opts)
	if output != "" {
		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
		response[""]["ClusterRoles"] = diff
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
