package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestNamespaces(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	// Create test namespace 1 - empty except for default resources
	emptyNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-namespace", Labels: AppLabels},
	}
	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), emptyNamespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating empty namespace: %v", err)
	}

	// Create default service account in empty namespace
	defaultSA := CreateTestServiceAccount("empty-namespace", "default", AppLabels)
	_, err = clientset.CoreV1().ServiceAccounts("empty-namespace").Create(context.TODO(), defaultSA, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating default service account: %v", err)
	}

	// Create kube-root-ca.crt configmap in empty namespace
	defaultCM := CreateTestConfigmap("empty-namespace", "kube-root-ca.crt", AppLabels)
	_, err = clientset.CoreV1().ConfigMaps("empty-namespace").Create(context.TODO(), defaultCM, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating default configmap: %v", err)
	}

	// Create test namespace 2 - has non-default resources
	nonEmptyNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "non-empty-namespace", Labels: AppLabels},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), nonEmptyNamespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating non-empty namespace: %v", err)
	}

	// Create a custom service in non-empty namespace
	customService := CreateTestService("non-empty-namespace", "custom-service")
	_, err = clientset.CoreV1().Services("non-empty-namespace").Create(context.TODO(), customService, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating custom service: %v", err)
	}

	// Create test namespace 3 - marked as used
	usedNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "used-namespace", Labels: UsedLabels},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), usedNamespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating used namespace: %v", err)
	}

	// Create test namespace 4 - marked as unused
	unusedNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "unused-namespace", Labels: UnusedLabels},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), unusedNamespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating unused namespace: %v", err)
	}

	return clientset
}

func TestRetrieveNamespaceNames(t *testing.T) {
	clientset := createTestNamespaces(t)

	namespaceNames, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(namespaceNames.Items) != 4 {
		t.Errorf("Expected 4 namespaces, got %d", len(namespaceNames.Items))
	}

	expectedNamespaces := map[string]bool{"empty-namespace": true, "non-empty-namespace": true, "used-namespace": true, "unused-namespace": true}
	for _, ns := range namespaceNames.Items {
		if !expectedNamespaces[ns.Name] {
			t.Errorf("Unexpected namespace name: %s", ns.Name)
		}
	}
}

func TestCountResourcesInNamespace(t *testing.T) {
	clientset := createTestNamespaces(t)

	// Test empty namespace - should return 0 (default resources are excluded)
	emptyCount, err := countResourcesInNamespace(clientset, "empty-namespace")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if emptyCount != 0 {
		t.Errorf("Expected 0 resources in empty namespace, got %d", emptyCount)
	}

	// Test non-empty namespace - should return 1 (the custom service)
	nonEmptyCount, err := countResourcesInNamespace(clientset, "non-empty-namespace")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if nonEmptyCount != 1 {
		t.Errorf("Expected 1 resource in non-empty namespace, got %d", nonEmptyCount)
	}
}

func TestProcessNamespaces(t *testing.T) {
	clientset := createTestNamespaces(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	filterOpts := &filters.Options{}

	output, err := processNamespaces(clientset, filterOpts, opts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should find 2 unused namespaces: empty-namespace and unused-namespace
	expectedNamespaces := []string{"empty-namespace", "unused-namespace"}
	if len(output) != 2 {
		t.Fatalf("Expected 2 unused namespaces, got %d", len(output))
	}

	for i, ns := range output {
		if ns.Name != expectedNamespaces[i] {
			t.Errorf("Expected unused namespace %s, got %s", expectedNamespaces[i], ns.Name)
		}
	}
}

func TestGetUnusedNamespaces(t *testing.T) {
	clientset := createTestNamespaces(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	filterOpts := &filters.Options{}

	output, err := GetUnusedNamespaces(filterOpts, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var result map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON output: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"": {
			"Namespace": {
				"empty-namespace",
				"unused-namespace",
			},
		},
	}

	if !reflect.DeepEqual(result, expectedOutput) {
		t.Errorf("Expected output does not match")
		t.Errorf("Expected: %+v", expectedOutput)
		t.Errorf("Got: %+v", result)
	}
}
