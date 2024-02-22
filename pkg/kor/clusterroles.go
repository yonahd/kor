package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/yonahd/kor/pkg/filters"
	v1 "k8s.io/api/rbac/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func retrieveUsedClusterRoles(clientset kubernetes.Interface, filterOpts *filters.Options) ([]string, error) {
	usedClusterRoles := make(map[string]bool)

	//Get a list of all ClusterRoles
	clusterRoleList, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve ClusterRoles: %v\n", err)
	}
	//Define a List where all labels for aggregation are stored
	aggregationLabelList := make(map[string]string)

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

	for _, rb := range roleBindingsAllNameSpaces {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}
		usedClusterRoles[rb.RoleRef.Name] = true
		if rb.RoleRef.Kind == "ClusterRole" {
			usedClusterRoles[rb.RoleRef.Name] = true
			clusterRole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), rb.RoleRef.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get ClusterRole")
			}
			if clusterRole.AggregationRule != nil {
				fmt.Println(rb.RoleRef.Name + clusterRole.Name)
				for _, matchLabel := range clusterRole.AggregationRule.ClusterRoleSelectors {
					for key, value := range matchLabel.MatchLabels {
						aggregationLabelList[key] = value
					}
				}
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get ClusterRole")
			}
		}

	}

	// Get a list of all cluster role bindings in the specified namespace
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to list cluster role bindings %v", err)
	}

	for _, crb := range clusterRoleBindings.Items {
		fmt.Println(crb.RoleRef.Name)
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}
		clusterRole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), crb.RoleRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get ClusterRole")
		}
		if clusterRole.AggregationRule != nil {
			// fmt.Println(crb.RoleRef.Name + clusterRole.Name)
			for _, matchLabel := range clusterRole.AggregationRule.ClusterRoleSelectors {
				for key, value := range matchLabel.MatchLabels {
					aggregationLabelList[key] = value
				}
			}
		}
		usedClusterRoles[crb.RoleRef.Name] = true

		usedClusterRoles[crb.RoleRef.Name] = true
	}
	//If the Role is aggregated add it to the List of used ClusterRoles
	for _, cr := range clusterRoleList.Items {
		for label := range cr.Labels {
			_, aggregated := aggregationLabelList[label]
			if aggregated {
				usedClusterRoles[cr.Name] = true
			}
		}
	}

	var usedClusterRoleNames []string
	for role := range usedClusterRoles {
		usedClusterRoleNames = append(usedClusterRoleNames, role)
	}
	return usedClusterRoleNames, nil
}

func retrieveClusterRoleNames(clientset kubernetes.Interface, filterOpts *filters.Options) ([]string, error) {
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(clusterRoles.Items))
	for _, clusterRole := range clusterRoles.Items {
		if pass, _ := filter.Run(filterOpts); pass {
			continue
		}

		names = append(names, clusterRole.Name)
	}
	return names, nil
}

func processClusterRoles(clientset kubernetes.Interface, filterOpts *filters.Options) ([]string, error) {
	usedClusterRoles, err := retrieveUsedClusterRoles(clientset, filterOpts)
	if err != nil {
		return nil, err
	}

	usedClusterRoles = RemoveDuplicatesAndSort(usedClusterRoles)

	clusterRoleNames, err := retrieveClusterRoleNames(clientset, filterOpts)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedClusterRoles, clusterRoleNames)
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
