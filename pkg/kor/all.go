package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

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

func getUnusedNamespaces(clientset kubernetes.Interface, filterOpts *filters.Options, opts common.Opts) ResourceDiff {
	namespaceDiff, err := processNamespaces(clientset, filterOpts, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "namespaces", err)
	}
	allNamespaceDiff := ResourceDiff{
		"Namespace",
		namespaceDiff,
	}
	return allNamespaceDiff
}

func GetUnusedAllNamespaced(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["ConfigMap"] = getUnusedCMs(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Service"] = getUnusedSVCs(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Secret"] = getUnusedSecrets(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["ServiceAccount"] = getUnusedServiceAccounts(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Deployment"] = getUnusedDeployments(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["StatefulSet"] = getUnusedStatefulSets(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Role"] = getUnusedRoles(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Hpa"] = getUnusedHpas(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Pvc"] = getUnusedPvcs(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Pod"] = getUnusedPods(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Ingress"] = getUnusedIngresses(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Pdb"] = getUnusedPdbs(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["Job"] = getUnusedJobs(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["ReplicaSet"] = getUnusedReplicaSets(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["DaemonSet"] = getUnusedDaemonSets(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["NetworkPolicy"] = getUnusedNetworkPolicies(clientset, namespace, filterOpts, opts).diff
			resources[namespace]["RoleBinding"] = getUnusedRoleBindings(clientset, namespace, filterOpts, opts).diff
		case "resource":
			appendResources(resources, "ConfigMap", namespace, getUnusedCMs(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Service", namespace, getUnusedSVCs(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Secret", namespace, getUnusedSecrets(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "ServiceAccount", namespace, getUnusedServiceAccounts(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Deployment", namespace, getUnusedDeployments(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "StatefulSet", namespace, getUnusedStatefulSets(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Role", namespace, getUnusedRoles(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Hpa", namespace, getUnusedHpas(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Pvc", namespace, getUnusedPvcs(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Pod", namespace, getUnusedPods(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Ingress", namespace, getUnusedIngresses(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Pdb", namespace, getUnusedPdbs(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "Job", namespace, getUnusedJobs(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "ReplicaSet", namespace, getUnusedReplicaSets(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "DaemonSet", namespace, getUnusedDaemonSets(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "NetworkPolicy", namespace, getUnusedNetworkPolicies(clientset, namespace, filterOpts, opts).diff)
			appendResources(resources, "RoleBinding", namespace, getUnusedRoleBindings(clientset, namespace, filterOpts, opts).diff)
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
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["Crd"] = getUnusedCrds(apiExtClient, dynamicClient, filterOpts).diff
		resources[""]["Pv"] = getUnusedPvs(clientset, filterOpts).diff
		resources[""]["ClusterRole"] = getUnusedClusterRoles(clientset, filterOpts).diff
		resources[""]["ClusterRoleBinding"] = getUnusedClusterRoleBindings(clientset, filterOpts, opts).diff
		resources[""]["StorageClass"] = getUnusedStorageClasses(clientset, filterOpts).diff
		resources[""]["VolumeAttachment"] = getUnusedVolumeAttachments(clientset, filterOpts).diff
		resources[""]["Namespace"] = getUnusedNamespaces(clientset, filterOpts, opts).diff
	case "resource":
		appendResources(resources, "Crd", "", getUnusedCrds(apiExtClient, dynamicClient, filterOpts).diff)
		appendResources(resources, "Pv", "", getUnusedPvs(clientset, filterOpts).diff)
		appendResources(resources, "ClusterRole", "", getUnusedClusterRoles(clientset, filterOpts).diff)
		appendResources(resources, "ClusterRoleBinding", "", getUnusedClusterRoleBindings(clientset, filterOpts, opts).diff)
		appendResources(resources, "StorageClass", "", getUnusedStorageClasses(clientset, filterOpts).diff)
		appendResources(resources, "VolumeAttachment", "", getUnusedVolumeAttachments(clientset, filterOpts).diff)
		appendResources(resources, "Namespace", "", getUnusedNamespaces(clientset, filterOpts, opts).diff)

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
