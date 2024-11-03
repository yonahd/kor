package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func retrieveNoNamespaceDiff(clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, resourceList []string, filterOpts *filters.Options) ([]ResourceDiff, []string) {
	var noNamespaceDiff []ResourceDiff
	markedForRemoval := make([]bool, len(resourceList))
	updatedResourceList := resourceList

	for counter, resource := range resourceList {
		switch resource {
		case "crd", "crds", "customresourcedefinition", "customresourcedefinitions":
			crdDiff := getUnusedCrds(apiExtClient, dynamicClient, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, crdDiff)
			markedForRemoval[counter] = true
		case "pv", "persistentvolume", "persistentvolumes":
			pvDiff := getUnusedPvs(clientset, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, pvDiff)
			markedForRemoval[counter] = true
		case "clusterrole", "clusterroles":
			clusterRoleDiff := getUnusedClusterRoles(clientset, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, clusterRoleDiff)
			markedForRemoval[counter] = true
		case "sc", "storageclass", "storageclasses":
			storageClassDiff := getUnusedStorageClasses(clientset, filterOpts)
			noNamespaceDiff = append(noNamespaceDiff, storageClassDiff)
			markedForRemoval[counter] = true
		}
	}

	// Remove elements marked for removal
	var clearedResourceList []string
	for i, marked := range markedForRemoval {
		if !marked {
			clearedResourceList = append(clearedResourceList, updatedResourceList[i])
		}
	}

	return noNamespaceDiff, clearedResourceList
}

func retrieveNamespaceDiffs(clientset kubernetes.Interface, namespace string, resourceList []string, filterOpts *filters.Options) []ResourceDiff {
	var allDiffs []ResourceDiff
	for _, resource := range resourceList {
		var diffResult ResourceDiff
		switch resource {
		case "cm", "configmap", "configmaps":
			diffResult = getUnusedCMs(clientset, namespace, filterOpts)
		case "svc", "service", "services":
			diffResult = getUnusedSVCs(clientset, namespace, filterOpts)
		case "secret", "secrets":
			diffResult = getUnusedSecrets(clientset, namespace, filterOpts)
		case "sa", "serviceaccount", "serviceaccounts":
			diffResult = getUnusedServiceAccounts(clientset, namespace, filterOpts)
		case "deploy", "deployment", "deployments":
			diffResult = getUnusedDeployments(clientset, namespace, filterOpts)
		case "sts", "statefulset", "statefulsets":
			diffResult = getUnusedStatefulSets(clientset, namespace, filterOpts)
		case "role", "roles":
			diffResult = getUnusedRoles(clientset, namespace, filterOpts)
		case "hpa", "horizontalpodautoscaler", "horizontalpodautoscalers":
			diffResult = getUnusedHpas(clientset, namespace, filterOpts)
		case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
			diffResult = getUnusedPvcs(clientset, namespace, filterOpts)
		case "ing", "ingress", "ingresses":
			diffResult = getUnusedIngresses(clientset, namespace, filterOpts)
		case "pdb", "poddisruptionbudget", "poddisruptionbudgets":
			diffResult = getUnusedPdbs(clientset, namespace, filterOpts)
		case "po", "pod", "pods":
			diffResult = getUnusedPods(clientset, namespace, filterOpts)
		case "job", "jobs":
			diffResult = getUnusedJobs(clientset, namespace, filterOpts)
		case "rs", "replicaset", "replicasets":
			diffResult = getUnusedReplicaSets(clientset, namespace, filterOpts)
		case "ds", "daemonset", "daemonsets":
			diffResult = getUnusedDaemonSets(clientset, namespace, filterOpts)
		case "netpol", "networkpolicy", "networkpolicies":
			diffResult = getUnusedNetworkPolicies(clientset, namespace, filterOpts)
		case "rolebinding", "rolebindings":
			diffResult = getUnusedNetworkPolicies(clientset, namespace, filterOpts)
		default:
			fmt.Printf("resource type %q is not supported\n", resource)
		}
		allDiffs = append(allDiffs, diffResult)
	}
	return allDiffs
}

func GetUnusedMulti(resourceNames string, filterOpts *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, clientsetinterface clusterconfig.ClientInterface, outputFormat string, opts common.Opts) (string, error) {
	resourceList := strings.Split(resourceNames, ",")
	namespaces := filterOpts.Namespaces(clientset)
	resources := make(map[string]map[string][]ResourceInfo)
	var err error

	if opts.GroupBy == "namespace" {
		resources[""] = make(map[string][]ResourceInfo)
	}

	noNamespaceDiff, resourceList := retrieveNoNamespaceDiff(clientset, apiExtClient, dynamicClient, resourceList, filterOpts)
	if len(noNamespaceDiff) != 0 {
		for _, diff := range noNamespaceDiff {
			if len(diff.diff) != 0 {
				if opts.DeleteFlag {
					if diff.diff, err = DeleteResource(diff.diff, clientset, "", diff.resourceType, opts.NoInteractive); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to delete %s %s: %v\n", diff.resourceType, diff.diff, err)
					}
				}
				switch opts.GroupBy {
				case "namespace":
					resources[""][diff.resourceType] = diff.diff
				case "resource":
					appendResources(resources, diff.resourceType, "", diff.diff)
				}
			}
		}
	}

	for _, namespace := range namespaces {
		allDiffs := retrieveNamespaceDiffs(clientset, namespace, resourceList, filterOpts)
		if opts.GroupBy == "namespace" {
			resources[namespace] = make(map[string][]ResourceInfo)
		}

		for _, diff := range allDiffs {
			if opts.DeleteFlag {
				if diff.diff, err = DeleteResource(diff.diff, clientset, namespace, diff.resourceType, opts.NoInteractive); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete %s %s in namespace %s: %v\n", diff.resourceType, diff.diff, namespace, err)
				}
			}
			switch opts.GroupBy {
			case "namespace":
				resources[namespace][diff.resourceType] = diff.diff
			case "resource":
				appendResources(resources, diff.resourceType, namespace, diff.diff)
			}
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

	unusedMulti, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedMulti, nil
}
