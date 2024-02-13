package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/filters"
)

type GetUnusedResourceJSONResponse struct {
	ResourceType string              `json:"resourceType"`
	Namespaces   map[string][]string `json:"namespaces"`
}

type ResourceDiff struct {
	resourceType string
	diff         []string
}

func getUnusedCMs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	cmDiff, err := processNamespaceCM(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "configmaps", namespace, err)
	}
	namespaceCMDiff := ResourceDiff{"ConfigMap", cmDiff}
	return namespaceCMDiff
}

func getUnusedSVCs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	svcDiff, err := ProcessNamespaceServices(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "services", namespace, err)
	}
	namespaceSVCDiff := ResourceDiff{"Service", svcDiff}
	return namespaceSVCDiff
}

func getUnusedSecrets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	secretDiff, err := processNamespaceSecret(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "secrets", namespace, err)
	}
	namespaceSecretDiff := ResourceDiff{"Secret", secretDiff}
	return namespaceSecretDiff
}

func getUnusedServiceAccounts(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	saDiff, err := processNamespaceSA(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "serviceaccounts", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"ServiceAccount", saDiff}
	return namespaceSADiff
}

func getUnusedDeployments(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	deployDiff, err := ProcessNamespaceDeployments(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "deployments", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"Deployment", deployDiff}
	return namespaceSADiff
}

func getUnusedStatefulSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	stsDiff, err := ProcessNamespaceStatefulSets(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "statefulSets", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"StatefulSet", stsDiff}
	return namespaceSADiff
}

func getUnusedRoles(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	roleDiff, err := processNamespaceRoles(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "roles", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"Role", roleDiff}
	return namespaceSADiff
}

func getUnusedClusterRoles(clientset kubernetes.Interface, filterOpts *filters.Options) ResourceDiff {
	clusterRoleDiff, err := processClusterRoles(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "clusterRoles", err)
	}
	aDiff := ResourceDiff{"ClusterRole", clusterRoleDiff}
	return aDiff
}

func getUnusedHpas(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	hpaDiff, err := processNamespaceHpas(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "hpas", namespace, err)
	}
	namespaceHpaDiff := ResourceDiff{"Hpa", hpaDiff}
	return namespaceHpaDiff
}

func getUnusedPvcs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	pvcDiff, err := processNamespacePvcs(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pvcs", namespace, err)
	}
	namespacePvcDiff := ResourceDiff{"Pvc", pvcDiff}
	return namespacePvcDiff
}

func getUnusedIngresses(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	ingressDiff, err := processNamespaceIngresses(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "ingresses", namespace, err)
	}
	namespaceIngressDiff := ResourceDiff{"Ingress", ingressDiff}
	return namespaceIngressDiff
}

func getUnusedPdbs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	pdbDiff, err := processNamespacePdbs(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pdbs", namespace, err)
	}
	namespacePdbDiff := ResourceDiff{"Pdb", pdbDiff}
	return namespacePdbDiff
}

func getUnusedCrds(apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, filterOpts *filters.Options) ResourceDiff {
	crdDiff, err := processCrds(apiExtClient, dynamicClient, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "Crds", err)
	}
	allCrdDiff := ResourceDiff{"Crd", crdDiff}
	return allCrdDiff
}

func getUnusedPvs(clientset kubernetes.Interface, filterOpts *filters.Options) ResourceDiff {
	pvDiff, err := processPvs(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s: %v\n", "Pvs", err)
	}
	allPvDiff := ResourceDiff{"Pv", pvDiff}
	return allPvDiff
}

func getUnusedPods(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	podDiff, err := ProcessNamespacePods(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "pods", namespace, err)
	}
	namespacePodDiff := ResourceDiff{"Pod", podDiff}
	return namespacePodDiff
}

func getUnusedJobs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	jobDiff, err := ProcessNamespaceJobs(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "jobs", namespace, err)
	}
	namespaceJobDiff := ResourceDiff{"Job", jobDiff}
	return namespaceJobDiff
}

func getUnusedReplicaSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	replicaSetDiff, err := ProcessNamespaceReplicaSets(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "jobs", namespace, err)
	}
	namespaceRSDiff := ResourceDiff{"ReplicaSet", replicaSetDiff}
	return namespaceRSDiff
}

func getUnusedDaemonSets(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ResourceDiff {
	dsDiff, err := ProcessNamespaceDaemonSets(clientset, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "DaemonSets", namespace, err)
	}
	namespaceSADiff := ResourceDiff{"DaemonSet", dsDiff}
	return namespaceSADiff
}

func GetUnusedAll(filterOpts *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts Opts) (string, error) {
	var outputBuffer bytes.Buffer

	namespaces := filterOpts.Namespaces(clientset)
	response := make(map[string]map[string][]string)

	for _, namespace := range namespaces {
		var allDiffs []ResourceDiff
		namespaceCMDiff := getUnusedCMs(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceCMDiff)
		namespaceSVCDiff := getUnusedSVCs(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceSVCDiff)
		namespaceSecretDiff := getUnusedSecrets(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceSecretDiff)
		namespaceSADiff := getUnusedServiceAccounts(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceSADiff)
		namespaceDeploymentDiff := getUnusedDeployments(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceDeploymentDiff)
		namespaceStatefulsetDiff := getUnusedStatefulSets(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceStatefulsetDiff)
		namespaceRoleDiff := getUnusedRoles(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceRoleDiff)
		namespaceHpaDiff := getUnusedHpas(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceHpaDiff)
		namespacePvcDiff := getUnusedPvcs(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespacePvcDiff)
		namespaceIngressDiff := getUnusedIngresses(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceIngressDiff)
		namespacePdbDiff := getUnusedPdbs(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespacePdbDiff)
		namespaceJobDiff := getUnusedJobs(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceJobDiff)
		namespaceRSDiff := getUnusedReplicaSets(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceRSDiff)
		namespaceDaemonsetDiff := getUnusedDaemonSets(clientset, namespace, filterOpts)
		allDiffs = append(allDiffs, namespaceDaemonsetDiff)

		output := FormatOutputAll(namespace, allDiffs, opts)

		outputBuffer.WriteString(output)
		outputBuffer.WriteString("\n")

		resourceMap := make(map[string][]string)
		for _, diff := range allDiffs {
			resourceMap[diff.resourceType] = diff.diff
		}
		response[namespace] = resourceMap
	}

	var allDiffs []ResourceDiff
	noNamespaceResourceMap := make(map[string][]string)
	crdDiff := getUnusedCrds(apiExtClient, dynamicClient, filterOpts)
	crdOutput := FormatOutputAll("", []ResourceDiff{crdDiff}, opts)
	outputBuffer.WriteString(crdOutput)
	outputBuffer.WriteString("\n")
	noNamespaceResourceMap[crdDiff.resourceType] = crdDiff.diff

	pvDiff := getUnusedPvs(clientset, filterOpts)
	pvOutput := FormatOutputAll("", []ResourceDiff{pvDiff}, opts)
	outputBuffer.WriteString(pvOutput)
	outputBuffer.WriteString("\n")
	noNamespaceResourceMap[pvDiff.resourceType] = pvDiff.diff

	clusterRoleDiff := getUnusedClusterRoles(clientset, filterOpts)
	clusterRoleOutput := FormatOutputAll("", []ResourceDiff{clusterRoleDiff}, opts)
	outputBuffer.WriteString(clusterRoleOutput)
	outputBuffer.WriteString("\n")
	noNamespaceResourceMap[clusterRoleDiff.resourceType] = clusterRoleDiff.diff

	output := FormatOutputAll("", allDiffs, opts)

	outputBuffer.WriteString(output)
	outputBuffer.WriteString("\n")
	response[""] = noNamespaceResourceMap

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	unusedAll, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedAll, nil
}
