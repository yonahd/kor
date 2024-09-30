package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//go:embed exceptions/rolebindings/rolebindings.json
var roleBindingsConfig []byte

func processNamespaceRoleBindings(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	roleBindingsList, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	roleList, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	roleNames := make(map[string]bool)
	for _, role := range roleList.Items {
		roleNames[role.Name] = true
	}

	// TODO is this list too big ?
	clusterRoleList, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterRoleNames := make(map[string]bool)
	for _, cr := range clusterRoleList.Items {
		clusterRoleNames[cr.Name] = true
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

		exceptionFound, err := isResourceException(rb.Name, rb.Namespace, config.ExceptionRoleBindings)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		if rb.RoleRef.Kind == "Role" && !roleNames[rb.RoleRef.Name] {
			unusedRoleBindingNames = append(unusedRoleBindingNames, ResourceInfo{Name: rb.Name, Reason: "Referenced Role does not exists"})
			continue
		}

		if rb.RoleRef.Kind == "ClusterRole" && !clusterRoleNames[rb.RoleRef.Name] {
			unusedRoleBindingNames = append(unusedRoleBindingNames, ResourceInfo{Name: rb.Name, Reason: "Referenced Cluster role does not exists"})
			continue
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
