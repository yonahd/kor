package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

var NamespacedFlagUsed bool

type GetUnusedResourceJSONResponse struct {
	ResourceType string              `json:"resourceType"`
	Namespaces   map[string][]string `json:"namespaces"`
}

type ResourceDiff struct {
	resourceType string
	diff         []ResourceInfo
}

func getMaxParallelResourceWorkers() int {
	maxWorkers := runtime.GOMAXPROCS(0)
	if maxWorkers < 1 {
		return 1
	}
	return maxWorkers
}

func runResourceDiffJobs(jobs []func() ResourceDiff, maxWorkers int) []ResourceDiff {
	if len(jobs) == 0 {
		return nil
	}

	if maxWorkers < 1 {
		maxWorkers = 1
	}

	results := make([]ResourceDiff, len(jobs))
	workers := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, job := range jobs {
		wg.Add(1)
		go func(index int, diffFn func() ResourceDiff) {
			defer wg.Done()

			workers <- struct{}{}
			defer func() {
				<-workers
			}()

			results[index] = diffFn()
		}(i, job)
	}

	wg.Wait()
	return results
}

func getUnusedCMs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	cmDiff, err := processNamespaceCM(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "configmaps", namespace, err)
	}
	namespaceCMDiff := ResourceDiff{
		"ConfigMap",
		cmDiff,
	}
	return namespaceCMDiff
}

func getUnusedSVCs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	svcDiff, err := processNamespaceServices(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "services", namespace, err)
	}
	namespaceSVCDiff := ResourceDiff{
		"Service",
		svcDiff,
	}
	return namespaceSVCDiff
}

func getUnusedSecrets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	secretDiff, err := processNamespaceSecret(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "secrets", namespace, err)
	}
	namespaceSecretDiff := ResourceDiff{
		"Secret",
		secretDiff,
	}
	return namespaceSecretDiff
}

func getUnusedServiceAccounts(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	saDiff, err := processNamespaceSA(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "serviceaccounts", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"ServiceAccount",
		saDiff,
	}
	return namespaceSADiff
}

func getUnusedDeployments(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	deployDiff, err := processNamespaceDeployments(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "deployments", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"Deployment",
		deployDiff,
	}
	return namespaceSADiff
}

func getUnusedStatefulSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	stsDiff, err := processNamespaceStatefulSets(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "statefulSets", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"StatefulSet",
		stsDiff,
	}
	return namespaceSADiff
}

func getUnusedRoles(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	roleDiff, err := processNamespaceRoles(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "roles", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"Role",
		roleDiff,
	}
	return namespaceSADiff
}

func getUnusedClusterRoles(clientset kubernetes.Interface, filterOpts *filters.Options) ResourceDiff {
	clusterRoleDiff, err := processClusterRoles(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "clusterRoles", err)
	}
	aDiff := ResourceDiff{
		"ClusterRole",
		clusterRoleDiff,
	}
	return aDiff
}

func getUnusedClusterRoleBindings(clientset kubernetes.Interface, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	clusterRoleBindingDiff, err := processClusterRoleBindings(clientset, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "clusterRoleBindings", err)
	}
	aDiff := ResourceDiff{
		"ClusterRoleBinding",
		clusterRoleBindingDiff,
	}
	return aDiff
}

func getUnusedHpas(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	hpaDiff, err := processNamespaceHpas(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "hpas", namespace, err)
	}
	namespaceHpaDiff := ResourceDiff{
		"Hpa",
		hpaDiff,
	}
	return namespaceHpaDiff
}

func getUnusedPvcs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	pvcDiff, err := processNamespacePvcs(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pvcs", namespace, err)
	}
	namespacePvcDiff := ResourceDiff{
		"Pvc",
		pvcDiff,
	}
	return namespacePvcDiff
}

func getUnusedIngresses(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	ingressDiff, err := processNamespaceIngresses(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "ingresses", namespace, err)
	}
	namespaceIngressDiff := ResourceDiff{
		"Ingress",
		ingressDiff,
	}
	return namespaceIngressDiff
}

func getUnusedPdbs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	pdbDiff, err := processNamespacePdbs(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pdbs", namespace, err)
	}
	namespacePdbDiff := ResourceDiff{
		"Pdb",
		pdbDiff,
	}
	return namespacePdbDiff
}

func getUnusedCrds(apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, filterOpts *filters.Options) ResourceDiff {
	crdDiff, err := processCrds(apiExtClient, dynamicClient, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "Crds", err)
	}
	allCrdDiff := ResourceDiff{
		"Crd",
		crdDiff,
	}
	return allCrdDiff
}

func getUnusedPvs(clientset kubernetes.Interface, filterOpts *filters.Options) ResourceDiff {
	pvDiff, err := processPvs(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "Pvs", err)
	}
	allPvDiff := ResourceDiff{
		"Pv",
		pvDiff,
	}
	return allPvDiff
}

func getUnusedPods(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	podDiff, err := processNamespacePods(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pods", namespace, err)
	}
	namespacePodDiff := ResourceDiff{
		"Pod",
		podDiff,
	}
	return namespacePodDiff
}

func getUnusedJobs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	jobDiff, err := processNamespaceJobs(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "jobs", namespace, err)
	}
	namespaceJobDiff := ResourceDiff{
		"Job",
		jobDiff,
	}
	return namespaceJobDiff
}

func getUnusedReplicaSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	replicaSetDiff, err := processNamespaceReplicaSets(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "ReplicaSets", namespace, err)
	}
	namespaceRSDiff := ResourceDiff{
		"ReplicaSet",
		replicaSetDiff,
	}
	return namespaceRSDiff
}

func getUnusedDaemonSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	dsDiff, err := processNamespaceDaemonSets(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "DaemonSets", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"DaemonSet",
		dsDiff,
	}
	return namespaceSADiff
}

func getUnusedStorageClasses(clientset kubernetes.Interface, filterOpts *filters.Options) ResourceDiff {
	scDiff, err := processStorageClasses(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "StorageClasses", err)
	}
	allScDiff := ResourceDiff{
		"StorageClass",
		scDiff,
	}
	return allScDiff
}
func getUnusedVolumeAttachments(clientset kubernetes.Interface, filterOpts *filters.Options) ResourceDiff {
	vattsDiff, err := processVolumeAttachments(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "VolumeAttachments", err)
	}
	allVattsDiff := ResourceDiff{
		"VolumeAttachment",
		vattsDiff,
	}
	return allVattsDiff
}

func getUnusedPriorityClasses(clientset kubernetes.Interface, filterOpts *filters.Options) ResourceDiff {
	pcDiff, err := processPriorityClasses(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "PriorityClasses", err)
	}
	allPcDiff := ResourceDiff{
		"PriorityClass",
		pcDiff,
	}
	return allPcDiff
}

func getUnusedNetworkPolicies(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	netpolDiff, err := processNamespaceNetworkPolicies(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "NetworkPolicies", namespace, err)
	}
	namespaceNetpolDiff := ResourceDiff{
		"NetworkPolicy",
		netpolDiff,
	}
	return namespaceNetpolDiff
}

func getUnusedRoleBindings(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	roleBindingDiff, err := processNamespaceRoleBindings(clientset, namespace, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "RoleBindings", namespace, err)
	}

	namespaceRoleBindingDiff := ResourceDiff{
		"RoleBinding",
		roleBindingDiff,
	}
	return namespaceRoleBindingDiff
}

func GetUnusedAllNamespaced(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	maxWorkers := getMaxParallelResourceWorkers()

	for _, namespace := range filterOpts.Namespaces(clientset) {
		resourceDiffs := runResourceDiffJobs([]func() ResourceDiff{
			func() ResourceDiff { return getUnusedCMs(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedSVCs(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedSecrets(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedServiceAccounts(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedDeployments(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedStatefulSets(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedRoles(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedHpas(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedPvcs(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedPods(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedIngresses(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedPdbs(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedJobs(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedReplicaSets(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedDaemonSets(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedNetworkPolicies(clientset, namespace, filterOpts, opts) },
			func() ResourceDiff { return getUnusedRoleBindings(clientset, namespace, filterOpts, opts) },
		}, maxWorkers)

		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			for _, diff := range resourceDiffs {
				resources[namespace][diff.resourceType] = diff.diff
			}
		case "resource":
			for _, diff := range resourceDiffs {
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

	unusedAllNamespaced, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedAllNamespaced, nil
}

func GetUnusedAllNonNamespaced(filterOpts *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	resourceDiffs := runResourceDiffJobs([]func() ResourceDiff{
		func() ResourceDiff { return getUnusedCrds(apiExtClient, dynamicClient, filterOpts) },
		func() ResourceDiff { return getUnusedPvs(clientset, filterOpts) },
		func() ResourceDiff { return getUnusedClusterRoles(clientset, filterOpts) },
		func() ResourceDiff { return getUnusedClusterRoleBindings(clientset, filterOpts, opts) },
		func() ResourceDiff { return getUnusedStorageClasses(clientset, filterOpts) },
		func() ResourceDiff { return getUnusedVolumeAttachments(clientset, filterOpts) },
		func() ResourceDiff { return getUnusedPriorityClasses(clientset, filterOpts) },
	}, getMaxParallelResourceWorkers())
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		for _, diff := range resourceDiffs {
			resources[""][diff.resourceType] = diff.diff
		}
	case "resource":
		for _, diff := range resourceDiffs {
			appendResources(resources, diff.resourceType, "", diff.diff)
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

	unusedAllNonNamespaced, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedAllNonNamespaced, nil
}

func GetUnusedAll(filterOpts *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	if NamespacedFlagUsed {
		if opts.Namespaced {
			return GetUnusedAllNamespaced(filterOpts, clientset, outputFormat, opts)
		}
		return GetUnusedAllNonNamespaced(filterOpts, clientset, apiExtClient, dynamicClient, outputFormat, opts)
	}

	unusedAllNamespaced, err := GetUnusedAllNamespaced(filterOpts, clientset, outputFormat, opts)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	// Skip getting non-namespaced resources if --include-namespaces flag is used
	if len(filterOpts.IncludeNamespaces) > 0 {
		return unusedAllNamespaced, nil
	}

	unusedAllNonNamespaced, err := GetUnusedAllNonNamespaced(filterOpts, clientset, apiExtClient, dynamicClient, outputFormat, opts)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	unusedAll := make(map[string]interface{})

	if outputFormat != "json" {
		unusedAll := unusedAllNamespaced + unusedAllNonNamespaced

		return unusedAll, nil
	} else {
		var namespacedResourceMap, nonNamespacedResourceMap map[string]interface{}

		if err := json.Unmarshal([]byte(unusedAllNamespaced), &namespacedResourceMap); err != nil {
			return "", err
		}
		if err := json.Unmarshal([]byte(unusedAllNonNamespaced), &nonNamespacedResourceMap); err != nil {
			return "", err
		}

		for k, v := range namespacedResourceMap {
			unusedAll[k] = v
		}
		for k, v := range nonNamespacedResourceMap {
			unusedAll[k] = v
		}

		jsonResponse, err := json.MarshalIndent(unusedAll, "", "  ")
		if err != nil {
			return "", err
		}

		return string(jsonResponse), nil
	}
}

func SetNamespacedFlagState(isFlagUsed bool) {
	NamespacedFlagUsed = isFlagUsed
}
