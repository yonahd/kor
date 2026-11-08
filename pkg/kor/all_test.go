package kor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestNonNamespacedResources(t *testing.T) *fake.Clientset {
	clientset := fake.NewClientset()

	pv := CreateTestPv("test-pv", "Available", AppLabels, "test-sc1")
	_, err := clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PV: %v", err)
	}

	clusterRole := CreateTestClusterRole("test-cluster-role", AppLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake ClusterRole: %v", err)
	}

	storageClass := CreateTestStorageClass("test-sc", "kor.com")
	_, err = clientset.StorageV1().StorageClasses().Create(context.TODO(), storageClass, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake StorageClass: %v", err)
	}

	priorityClass := CreateTestPriorityClass("test-pc", 1000)
	_, err = clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), priorityClass, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PriorityClass: %v", err)
	}

	return clientset
}

func createLonelyTestNamespace(t *testing.T) *fake.Clientset {
	clientset := fake.NewClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	return clientset
}

func TestGetUnusedAllNamespacedGroupByNamespace(t *testing.T) {
	clientset := createTestMultiResources(t)

	opts := common.Opts{GroupBy: "namespace"}
	output, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}

	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}

	configMaps := actual[testNamespace]["ConfigMap"]
	if !contains(configMaps, "configmap-1") {
		t.Errorf("Expected configmap-1 in output, got %v", configMaps)
	}
	deployments := actual[testNamespace]["Deployment"]
	if !contains(deployments, "test-deployment1") {
		t.Errorf("Expected test-deployment1 in output, got %v", deployments)
	}
}

func TestGetUnusedAllNamespacedGroupByResource(t *testing.T) {
	clientset := createTestMultiResources(t)

	opts := common.Opts{GroupBy: "resource"}
	output, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}

	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}

	namespaces := actual["ConfigMap"]
	if !contains(namespaces[testNamespace], "configmap-1") {
		t.Errorf("Expected configmap-1 in output, got %v", namespaces)
	}
}

func TestGetUnusedAllNamespacedTable(t *testing.T) {
	clientset := createTestMultiResources(t)

	opts := common.Opts{GroupBy: "namespace"}
	output, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "table", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}

	for _, expected := range []string{"Unused resources in namespace", "configmap-1", "test-deployment1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestGetUnusedAllNamespacedTableGroupByResource(t *testing.T) {
	clientset := createTestMultiResources(t)

	opts := common.Opts{GroupBy: "resource"}
	output, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "table", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}

	for _, expected := range []string{"Unused configmaps", "configmap-1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestGetUnusedAllNamespacedYAML(t *testing.T) {
	clientset := createTestMultiResources(t)

	opts := common.Opts{GroupBy: "namespace"}
	output, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "yaml", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}

	if !strings.Contains(output, "test-namespace:") {
		t.Errorf("Expected test-namespace in yaml output, got %s", output)
	}
}

func TestGetUnusedAllNamespacedEmptyNamespace(t *testing.T) {
	clientset := createLonelyTestNamespace(t)

	opts := common.Opts{GroupBy: "namespace", Verbose: true}
	output, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "table", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}

	if !strings.Contains(output, "No unused resources found in the namespace") {
		t.Errorf("Expected verbose empty message, got %s", output)
	}
}

func TestGetUnusedAllNonNamespaced(t *testing.T) {
	clientset := createTestNonNamespacedResources(t)
	apiExtClient, dynamicClient := createTestCRDs(t)

	opts := common.Opts{GroupBy: "namespace"}
	output, err := GetUnusedAllNonNamespaced(&filters.Options{}, clientset, apiExtClient, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNonNamespaced: %v", err)
	}

	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}

	nonNamespaced := actual[""]
	if !contains(nonNamespaced["Pv"], "test-pv") {
		t.Errorf("Expected test-pv in output, got %v", nonNamespaced["Pv"])
	}
	if !contains(nonNamespaced["ClusterRole"], "test-cluster-role") {
		t.Errorf("Expected test-cluster-role in output, got %v", nonNamespaced["ClusterRole"])
	}
	if !contains(nonNamespaced["StorageClass"], "test-sc") {
		t.Errorf("Expected test-sc in output, got %v", nonNamespaced["StorageClass"])
	}
	if !contains(nonNamespaced["PriorityClass"], "test-pc") {
		t.Errorf("Expected test-pc in output, got %v", nonNamespaced["PriorityClass"])
	}
	if len(nonNamespaced["Crd"]) != 2 {
		t.Errorf("Expected 2 unused CRDs, got %v", nonNamespaced["Crd"])
	}
}

func TestGetUnusedAllNonNamespacedGroupByResource(t *testing.T) {
	clientset := createTestNonNamespacedResources(t)
	apiExtClient, dynamicClient := createTestCRDs(t)

	opts := common.Opts{GroupBy: "resource"}
	output, err := GetUnusedAllNonNamespaced(&filters.Options{}, clientset, apiExtClient, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNonNamespaced: %v", err)
	}

	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}

	if !contains(actual["Pv"][""], "test-pv") {
		t.Errorf("Expected test-pv in output, got %v", actual["Pv"])
	}
	if !contains(actual["ClusterRole"][""], "test-cluster-role") {
		t.Errorf("Expected test-cluster-role in output, got %v", actual["ClusterRole"])
	}
}

func createTestAllClientset(t *testing.T) *fake.Clientset {
	clientset := createTestMultiResources(t)

	_, err := clientset.CoreV1().PersistentVolumes().Create(context.TODO(), CreateTestPv("test-pv", "Available", AppLabels, "test-sc1"), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PV: %v", err)
	}

	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), CreateTestClusterRole("test-cluster-role", AppLabels), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake ClusterRole: %v", err)
	}

	_, err = clientset.StorageV1().StorageClasses().Create(context.TODO(), CreateTestStorageClass("test-sc", "kor.com"), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake StorageClass: %v", err)
	}

	_, err = clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), CreateTestPriorityClass("test-pc", 1000), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PriorityClass: %v", err)
	}

	return clientset
}

func TestGetUnusedAll(t *testing.T) {
	clientset := createTestAllClientset(t)
	apiExtClient, dynamicClient := createTestCRDs(t)

	opts := common.Opts{GroupBy: "namespace"}
	output, err := GetUnusedAll(&filters.Options{}, clientset, apiExtClient, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAll: %v", err)
	}

	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}

	if !contains(actual[testNamespace]["ConfigMap"], "configmap-1") {
		t.Errorf("Expected configmap-1 in output, got %v", actual)
	}
	if !contains(actual[""]["ClusterRole"], "test-cluster-role") {
		t.Errorf("Expected test-cluster-role in output, got %v", actual)
	}
}

func TestGetUnusedAllTable(t *testing.T) {
	clientset := createTestAllClientset(t)
	apiExtClient, dynamicClient := createTestCRDs(t)

	opts := common.Opts{GroupBy: "namespace"}
	output, err := GetUnusedAll(&filters.Options{}, clientset, apiExtClient, dynamicClient, "table", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAll: %v", err)
	}

	for _, expected := range []string{"configmap-1", "test-pv", "test-cluster-role"} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestGetUnusedAllWithIncludeNamespaces(t *testing.T) {
	clientset := createTestAllClientset(t)
	apiExtClient, dynamicClient := createTestCRDs(t)

	filterOpts := &filters.Options{
		IncludeNamespaces: []string{testNamespace},
	}
	opts := common.Opts{GroupBy: "namespace"}
	output, err := GetUnusedAll(filterOpts, clientset, apiExtClient, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAll: %v", err)
	}

	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}

	if !contains(actual[testNamespace]["ConfigMap"], "configmap-1") {
		t.Errorf("Expected configmap-1 in output, got %v", actual)
	}
	if _, exists := actual[""]; exists {
		t.Errorf("Expected no non-namespaced resources, got %v", actual)
	}
}

func TestGetUnusedAllFlags(t *testing.T) {
	SetNamespacedFlagState(true)
	t.Cleanup(func() { SetNamespacedFlagState(false) })

	clientset := createTestAllClientset(t)
	apiExtClient, dynamicClient := createTestCRDs(t)

	opts := common.Opts{GroupBy: "namespace", Namespaced: true}
	output, err := GetUnusedAll(&filters.Options{}, clientset, apiExtClient, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAll: %v", err)
	}

	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}
	if _, exists := actual[""]; exists {
		t.Errorf("Expected namespaced-only output when Namespaced flag set, got %v", actual)
	}

	opts.Namespaced = false
	output, err = GetUnusedAll(&filters.Options{}, clientset, apiExtClient, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAll: %v", err)
	}
	actual = nil
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}
	if _, exists := actual[testNamespace]; exists {
		t.Errorf("Expected non-namespaced-only output when Namespaced flag unset, got %v", actual)
	}
}

func TestSetNamespacedFlagState(t *testing.T) {
	SetNamespacedFlagState(true)
	if !NamespacedFlagUsed {
		t.Errorf("Expected NamespacedFlagUsed to be true")
	}
	SetNamespacedFlagState(false)
	if NamespacedFlagUsed {
		t.Errorf("Expected NamespacedFlagUsed to be false")
	}
}
