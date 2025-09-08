package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/namespaces/namespaces.json
var namespacesConfig []byte

// isDefaultResource checks if a resource is one of the default resources
// that are automatically created in every namespace
func isDefaultResource(resourceType, name string) bool {
	// Default service account
	if resourceType == "serviceaccount" && name == "default" {
		return true
	}

	// Default CA configmap
	if resourceType == "configmap" && name == "kube-root-ca.crt" {
		return true
	}

	// Default service account token secret (for older Kubernetes versions)
	if resourceType == "secret" && name == "default-token-" {
		return true
	}

	return false
}

// countResourcesInNamespace counts non-default resources in a namespace
func countResourcesInNamespace(clientset kubernetes.Interface, namespace string) (int, error) {
	totalCount := 0

	// Count ConfigMaps (excluding default ones)
	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	for _, cm := range configmaps.Items {
		if !isDefaultResource("configmap", cm.Name) {
			totalCount++
		}
	}

	// Count Secrets (excluding default ones)
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	for _, secret := range secrets.Items {
		if !isDefaultResource("secret", secret.Name) {
			totalCount++
		}
	}

	// Count ServiceAccounts (excluding default one)
	serviceAccounts, err := clientset.CoreV1().ServiceAccounts(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	for _, sa := range serviceAccounts.Items {
		if !isDefaultResource("serviceaccount", sa.Name) {
			totalCount++
		}
	}

	// Count Services
	services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(services.Items)

	// Count Pods
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(pods.Items)

	// Count Deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(deployments.Items)

	// Count StatefulSets
	statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(statefulSets.Items)

	// Count DaemonSets
	daemonSets, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(daemonSets.Items)

	// Count ReplicaSets
	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(replicaSets.Items)

	// Count Jobs
	jobs, err := clientset.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(jobs.Items)

	// Count CronJobs
	cronJobs, err := clientset.BatchV1().CronJobs(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(cronJobs.Items)

	// Count PersistentVolumeClaims
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(pvcs.Items)

	// Count Ingresses
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(ingresses.Items)

	// Count NetworkPolicies
	networkPolicies, err := clientset.NetworkingV1().NetworkPolicies(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(networkPolicies.Items)

	// Count Roles
	roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(roles.Items)

	// Count RoleBindings
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(roleBindings.Items)

	// Count PodDisruptionBudgets
	pdbs, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(pdbs.Items)

	// Count HorizontalPodAutoscalers
	hpas, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(hpas.Items)

	// Count ResourceQuotas
	resourceQuotas, err := clientset.CoreV1().ResourceQuotas(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(resourceQuotas.Items)

	// Count LimitRanges
	limitRanges, err := clientset.CoreV1().LimitRanges(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(limitRanges.Items)

	// Count Endpoints
	endpoints, err := clientset.CoreV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(endpoints.Items)

	// Count EndpointSlices
	endpointSlices, err := clientset.DiscoveryV1().EndpointSlices(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	totalCount += len(endpointSlices.Items)

	// Count Events (recent activity indicator)
	events, err := clientset.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	// Count events from the last hour as "recent activity"
	recentEvents := 0
	for _, event := range events.Items {
		if event.LastTimestamp.Time.After(time.Now().Add(-time.Hour)) {
			recentEvents++
		}
	}
	totalCount += recentEvents

	return totalCount, nil
}

func processNamespaces(clientset kubernetes.Interface, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	// Get all namespaces
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(namespacesConfig)
	if err != nil {
		return nil, err
	}

	var unusedNamespaces []ResourceInfo

	for _, ns := range namespaceList.Items {
		// Skip system namespaces by default unless explicitly included
		if ns.Name == "kube-system" || ns.Name == "kube-public" || ns.Name == "kube-node-lease" {
			continue
		}

		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(ns.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&ns).Run(filterOpts); pass {
			continue
		}

		if ns.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedNamespaces = append(unusedNamespaces, ResourceInfo{Name: ns.Name, Reason: reason})
			continue
		}

		// Check for exception
		exceptionFound, err := isResourceException(ns.Name, "", config.ExceptionNamespaces)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		// Count resources in the namespace
		resourceCount, err := countResourcesInNamespace(clientset, ns.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to count resources in namespace %s: %v\n", ns.Name, err)
			continue
		}

		// If namespace has no resources (or only default resources), it's unused
		if resourceCount == 0 {
			reason := "Namespace contains no resources (excluding default resources)"
			unusedNamespaces = append(unusedNamespaces, ResourceInfo{Name: ns.Name, Reason: reason})
		}
	}

	if opts.DeleteFlag {
		if unusedNamespaces, err = DeleteResource(unusedNamespaces, clientset, "", "Namespace", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete Namespace %s: %v\n", unusedNamespaces, err)
		}
	}

	return unusedNamespaces, nil
}

func GetUnusedNamespaces(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)

	diff, err := processNamespaces(clientset, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process namespaces: %v\n", err)
		return "", err
	}

	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["Namespace"] = diff
	case "resource":
		appendResources(resources, "Namespace", "", diff)
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

	unusedNamespaces, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedNamespaces, nil
}
