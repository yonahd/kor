package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

type GetUnusedResourceJSONResponse struct {
	ResourceType string              `json:"resourceType"`
	Namespaces   map[string][]string `json:"namespaces"`
}

type ResourceDiff struct {
	resourceType string
	diff         []ResourceInfo
}

func getUnusedCMs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	cmDiff, err := processNamespaceCM(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "configmaps", namespace, err)
	}
	namespaceCMDiff := ResourceDiff{
		"ConfigMap",
		cmDiff,
	}
	return namespaceCMDiff
}

func getUnusedSVCs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	svcDiff, err := processNamespaceServices(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "services", namespace, err)
	}
	namespaceSVCDiff := ResourceDiff{
		"Service",
		svcDiff,
	}
	return namespaceSVCDiff
}

func getUnusedSecrets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	secretDiff, err := processNamespaceSecret(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "secrets", namespace, err)
	}
	namespaceSecretDiff := ResourceDiff{
		"Secret",
		secretDiff,
	}
	return namespaceSecretDiff
}

func getUnusedServiceAccounts(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	saDiff, err := processNamespaceSA(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "serviceaccounts", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"ServiceAccount",
		saDiff,
	}
	return namespaceSADiff
}

func getUnusedDeployments(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	deployDiff, err := processNamespaceDeployments(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "deployments", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"Deployment",
		deployDiff,
	}
	return namespaceSADiff
}

func getUnusedStatefulSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	stsDiff, err := processNamespaceStatefulSets(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "statefulSets", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"StatefulSet",
		stsDiff,
	}
	return namespaceSADiff
}

func getUnusedRoles(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	roleDiff, err := processNamespaceRoles(clientset, namespace, filterOpts)
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

func getUnusedHpas(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	hpaDiff, err := processNamespaceHpas(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "hpas", namespace, err)
	}
	namespaceHpaDiff := ResourceDiff{
		"Hpa",
		hpaDiff,
	}
	return namespaceHpaDiff
}

func getUnusedPvcs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	pvcDiff, err := processNamespacePvcs(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pvcs", namespace, err)
	}
	namespacePvcDiff := ResourceDiff{
		"Pvc",
		pvcDiff,
	}
	return namespacePvcDiff
}

func getUnusedIngresses(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	ingressDiff, err := processNamespaceIngresses(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "ingresses", namespace, err)
	}
	namespaceIngressDiff := ResourceDiff{
		"Ingress",
		ingressDiff,
	}
	return namespaceIngressDiff
}

func getUnusedPdbs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	pdbDiff, err := processNamespacePdbs(clientset, namespace, filterOpts)
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

func getUnusedPods(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	podDiff, err := processNamespacePods(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pods", namespace, err)
	}
	namespacePodDiff := ResourceDiff{
		"Pod",
		podDiff,
	}
	return namespacePodDiff
}

func getUnusedJobs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	jobDiff, err := processNamespaceJobs(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "jobs", namespace, err)
	}
	namespaceJobDiff := ResourceDiff{
		"Job",
		jobDiff,
	}
	return namespaceJobDiff
}

func getUnusedReplicaSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	replicaSetDiff, err := processNamespaceReplicaSets(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "ReplicaSets", namespace, err)
	}
	namespaceRSDiff := ResourceDiff{
		"ReplicaSet",
		replicaSetDiff,
	}
	return namespaceRSDiff
}

func getUnusedDaemonSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	dsDiff, err := processNamespaceDaemonSets(clientset, namespace, filterOpts)
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

func getUnusedNetworkPolicies(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	netpolDiff, err := processNamespaceNetworkPolicies(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "NetworkPolicies", namespace, err)
	}
	namespaceNetpolDiff := ResourceDiff{
		"NetworkPolicy",
		netpolDiff,
	}
	return namespaceNetpolDiff
}

func getUnusedRoleBindings(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	roleBindingDiff, err := processNamespaceRoleBindings(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "RoleBindings", namespace, err)
	}

	namespaceRoleBindingDiff := ResourceDiff{
		"RoleBinding",
		roleBindingDiff,
	}
	return namespaceRoleBindingDiff
}

func GetUnusedAllNamespaced(filterOpts *filters.Options, clientset kubernetes.Interface, clientsetinterface clusterconfig.ClientInterface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["ConfigMap"] = getUnusedCMs(clientset, namespace, filterOpts).diff
			resources[namespace]["Service"] = getUnusedSVCs(clientset, namespace, filterOpts).diff
			resources[namespace]["Secret"] = getUnusedSecrets(clientset, namespace, filterOpts).diff
			resources[namespace]["ServiceAccount"] = getUnusedServiceAccounts(clientset, namespace, filterOpts).diff
			resources[namespace]["Deployment"] = getUnusedDeployments(clientset, namespace, filterOpts).diff
			resources[namespace]["StatefulSet"] = getUnusedStatefulSets(clientset, namespace, filterOpts).diff
			resources[namespace]["Role"] = getUnusedRoles(clientset, namespace, filterOpts).diff
			resources[namespace]["Hpa"] = getUnusedHpas(clientset, namespace, filterOpts).diff
			resources[namespace]["Pvc"] = getUnusedPvcs(clientset, namespace, filterOpts).diff
			resources[namespace]["Pod"] = getUnusedPods(clientset, namespace, filterOpts).diff
			resources[namespace]["Ingress"] = getUnusedIngresses(clientset, namespace, filterOpts).diff
			resources[namespace]["Pdb"] = getUnusedPdbs(clientset, namespace, filterOpts).diff
			resources[namespace]["Job"] = getUnusedJobs(clientset, namespace, filterOpts).diff
			resources[namespace]["ReplicaSet"] = getUnusedReplicaSets(clientset, namespace, filterOpts).diff
			resources[namespace]["DaemonSet"] = getUnusedDaemonSets(clientset, namespace, filterOpts).diff
			resources[namespace]["NetworkPolicy"] = getUnusedNetworkPolicies(clientset, namespace, filterOpts).diff
			resources[namespace]["RoleBinding"] = getUnusedRoleBindings(clientset, namespace, filterOpts).diff
			GetUnusedCrdsThirdParty(opts.GroupBy, clientsetinterface, namespace, filterOpts, resources, true)

		case "resource":
			appendResources(resources, "ConfigMap", namespace, getUnusedCMs(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Service", namespace, getUnusedSVCs(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Secret", namespace, getUnusedSecrets(clientset, namespace, filterOpts).diff)
			appendResources(resources, "ServiceAccount", namespace, getUnusedServiceAccounts(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Deployment", namespace, getUnusedDeployments(clientset, namespace, filterOpts).diff)
			appendResources(resources, "StatefulSet", namespace, getUnusedStatefulSets(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Role", namespace, getUnusedRoles(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Hpa", namespace, getUnusedHpas(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Pvc", namespace, getUnusedPvcs(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Pod", namespace, getUnusedPods(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Ingress", namespace, getUnusedIngresses(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Pdb", namespace, getUnusedPdbs(clientset, namespace, filterOpts).diff)
			appendResources(resources, "Job", namespace, getUnusedJobs(clientset, namespace, filterOpts).diff)
			appendResources(resources, "ReplicaSet", namespace, getUnusedReplicaSets(clientset, namespace, filterOpts).diff)
			appendResources(resources, "DaemonSet", namespace, getUnusedDaemonSets(clientset, namespace, filterOpts).diff)
			appendResources(resources, "NetworkPolicy", namespace, getUnusedNetworkPolicies(clientset, namespace, filterOpts).diff)
			appendResources(resources, "RoleBinding", namespace, getUnusedRoleBindings(clientset, namespace, filterOpts).diff)
			GetUnusedCrdsThirdParty(opts.GroupBy, clientsetinterface, namespace, filterOpts, resources, true)
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

func GetUnusedAllNonNamespaced(filterOpts *filters.Options, clientset kubernetes.Interface, clientsetinterface clusterconfig.ClientInterface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["Crd"] = getUnusedCrds(apiExtClient, dynamicClient, filterOpts).diff
		resources[""]["Pv"] = getUnusedPvs(clientset, filterOpts).diff
		resources[""]["ClusterRole"] = getUnusedClusterRoles(clientset, filterOpts).diff
		resources[""]["StorageClass"] = getUnusedStorageClasses(clientset, filterOpts).diff
		GetUnusedCrdsThirdParty(opts.GroupBy, clientsetinterface, "", filterOpts, resources, false)
	case "resource":
		appendResources(resources, "Crd", "", getUnusedCrds(apiExtClient, dynamicClient, filterOpts).diff)
		appendResources(resources, "Pv", "", getUnusedPvs(clientset, filterOpts).diff)
		appendResources(resources, "ClusterRole", "", getUnusedClusterRoles(clientset, filterOpts).diff)
		appendResources(resources, "StorageClass", "", getUnusedStorageClasses(clientset, filterOpts).diff)
		GetUnusedCrdsThirdParty(opts.GroupBy, clientsetinterface, "", filterOpts, resources, false)
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

func GetUnusedAll(filterOpts *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, clientsetinterface clusterconfig.ClientInterface, outputFormat string, opts common.Opts) (string, error) {
	unusedAllNamespaced, err := GetUnusedAllNamespaced(filterOpts, clientset, clientsetinterface, outputFormat, opts)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	// Skip getting non-namespaced resources if --include-namespaces flag is used
	if len(filterOpts.IncludeNamespaces) > 0 {
		return unusedAllNamespaced, nil
	}

	unusedAllNonNamespaced, err := GetUnusedAllNonNamespaced(filterOpts, clientset, clientsetinterface, apiExtClient, dynamicClient, outputFormat, opts)
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

func GetUnusedCrdsThirdParty(groupBy string, clientsetinterface clusterconfig.ClientInterface, namespace string, filterOpts *filters.Options, resources map[string]map[string][]ResourceInfo, namespaced bool) map[string]map[string][]ResourceInfo {
	for _, crd := range filterOpts.CleanRepeatedCrds() {
		if namespaced {
			switch crd {
			case "argo-rollouts":
				unusedArgoRollouts := GetUnusedArgoRollouts(clientsetinterface, namespace, filterOpts)
				separateItemsThirdParty(resources, namespace, groupBy, "ArgoRollout", unusedArgoRollouts.diff)
			case "argo-rollouts-analysis-templates":
				unusedArgoRolloutsAnalysisTemplates := GetUnusedArgoRolloutsAnalysisTemplates(clientsetinterface, namespace, filterOpts)
				separateItemsThirdParty(resources, namespace, groupBy, "ArgoRollouts-AnalysisTemplate", unusedArgoRolloutsAnalysisTemplates.diff)
			}
		}
		if !namespaced {
			switch crd {
			case "argo-rollouts-cluster-analysis-templates":
				unusedArgoRolloutsClusterAnalysisTemplates := GetUnusedArgoRolloutsClusterAnalysisTemplates(clientsetinterface, "", filterOpts)
				separateItemsThirdParty(resources, "", groupBy, "ArgoRollouts-ClusterAnalysisTemplate", unusedArgoRolloutsClusterAnalysisTemplates.diff)
			}
		}
	}
	return resources
}

func separateItemsThirdParty(resources map[string]map[string][]ResourceInfo, namespace string, groupBy string, name string, diff []ResourceInfo) map[string]map[string][]ResourceInfo {
	switch groupBy {
	case "namespace":
		resources[namespace][name] = diff
	case "resource":
		appendResources(resources, name, namespace, diff)
	}

	return resources
}
