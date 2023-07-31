package kor

import (
	"context"
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"os"
)

func retrieveUsedRoles(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	// Get a list of all role bindings in the specified namespace
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings in namespace %s: %v", namespace, err)
	}

	// Create a map to store role binding names
	usedRoles := make(map[string]bool)

	// Populate the map with role binding names
	for _, rb := range roleBindings.Items {
		usedRoles[rb.RoleRef.Name] = true
	}

	// Create a slice to store used role names
	var usedRoleNames []string

	// Extract used role names from the map
	for role := range usedRoles {
		usedRoleNames = append(usedRoleNames, role)
	}

	return usedRoleNames, nil
}

func retrieveRoleNames(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	roles, err := kubeClient.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(roles.Items))
	for _, role := range roles.Items {
		names = append(names, role.Name)
	}
	return names, nil
}

func calculateRoleDifference(usedRoles []string, roleNames []string) []string {
	difference := []string{}
	for _, name := range roleNames {
		found := false
		for _, usedName := range usedRoles {
			if name == usedName {
				found = true
				break
			}
		}
		if !found {
			difference = append(difference, name)
		}
	}
	return difference
}

func processNamespaceRoles(kubeClient *kubernetes.Clientset, namespace string) ([]string, error) {
	usedRoles, err := retrieveUsedRoles(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	usedRoles = RemoveDuplicatesAndSort(usedRoles)

	roleNames, err := retrieveRoleNames(kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	diff := calculateRoleDifference(usedRoles, roleNames)
	return diff, nil

}

func GetUnusedRoles(namespace string) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient()
	namespaces = SetNamespaceList(namespace, kubeClient)

	for _, namespace := range namespaces {
		diff, err := processNamespaceRoles(kubeClient, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Roles")
		fmt.Println(output)
		fmt.Println()
	}
}

func GetUnusedRolesJSON(namespace string) (string, error) {
	var kubeClient *kubernetes.Clientset
	var namespaces []string

	kubeClient = GetKubeClient()
	namespaces = SetNamespaceList(namespace, kubeClient)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		var allDiffs []ResourceDiff

		namespaceRoleDiff := getUnusedRoles(kubeClient, namespace)
		allDiffs = append(allDiffs, namespaceRoleDiff)

		resourceMap := make(map[string][]string)
		for _, diff := range allDiffs {
			resourceMap[diff.resourceType] = diff.diff
		}
		response[namespace] = resourceMap
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonResponse), nil
}
