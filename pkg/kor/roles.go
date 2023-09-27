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
	"sigs.k8s.io/yaml"
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

func retrieveRoleNames(clientset kubernetes.Interface, namespace string) ([]string, error) {
	roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(roles.Items))
	for _, role := range roles.Items {
		if role.Labels["kor/used"] == "true" {
			continue
		}

		names = append(names, role.Name)
	}
	return names, nil
}

func processNamespaceRoles(clientset kubernetes.Interface, namespace string) ([]string, error) {
	usedRoles, err := retrieveUsedRoles(clientset, namespace)
	if err != nil {
		return nil, err
	}

	usedRoles = RemoveDuplicatesAndSort(usedRoles)

	roleNames, err := retrieveRoleNames(clientset, namespace)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedRoles, roleNames)
	return diff, nil

}

func GetUnusedRoles(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, slackOpts SlackOpts) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)

	var outputBuffer bytes.Buffer

	for _, namespace := range namespaces {
		diff, err := processNamespaceRoles(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		output := FormatOutput(namespace, diff, "Roles")

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")
	}

	if slackOpts != (SlackOpts{}) {
		if err := SendToSlack(SlackMessage{}, slackOpts, outputBuffer.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send message to slack: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(outputBuffer.String())
	}
}

func GetUnusedRolesStructured(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string) (string, error) {
	namespaces := SetNamespaceList(includeExcludeLists, clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		diff, err := processNamespaceRoles(clientset, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		resourceMap := make(map[string][]string)
		resourceMap["Roles"] = diff
		response[namespace] = resourceMap
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	if outputFormat == "yaml" {
		yamlResponse, err := yaml.JSONToYAML(jsonResponse)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		return string(yamlResponse), nil
	} else {
		return string(jsonResponse), nil
	}
}
