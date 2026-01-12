package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestPriorityClass(t *testing.T) *fake.Clientset {
	clientset := fake.NewClientset()

	pc1 := CreateTestPriorityClass("test-pc1", 1000)
	_, err := clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), pc1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "PriorityClass", err)
	}

	return clientset
}

func TestRetrieveUsedPriorityClasses(t *testing.T) {
	clientset := fake.NewClientset()

	pc1 := CreateTestPriorityClass("test-pc1", 1000)
	_, err := clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), pc1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PriorityClass: %v", err)
	}

	pod := CreateTestPod(testNamespace, "test-pod", "", nil, AppLabels)
	pod.Spec.PriorityClassName = "test-pc1"
	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Pod: %v", err)
	}

	usedPriorityClasses, err := retrieveUsedPriorityClasses(clientset)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !contains(usedPriorityClasses, "test-pc1") {
		t.Errorf("Expected 'test-pc1', got %v", usedPriorityClasses)
	}
}

func TestProcessPriorityClasses(t *testing.T) {
	clientset := createTestPriorityClass(t)
	unusedPriorityClasses, err := processPriorityClasses(clientset, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedPriorityClasses) != 1 {
		t.Errorf("Expected 1 unused PriorityClass, got %d", len(unusedPriorityClasses))
	}

	if unusedPriorityClasses[0].Name != "test-pc1" {
		t.Errorf("Expected 'test-pc1', got %s", unusedPriorityClasses[0].Name)
	}
}

func TestGetUnusedPriorityClassesStructured(t *testing.T) {
	clientset := createTestPriorityClass(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedPriorityClasses(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedPriorityClasses: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"": {
			"PriorityClass": {"test-pc1"},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestFilterOwnerReferencedPriorityClasses(t *testing.T) {
	clientset := fake.NewClientset()

	// Create two priority classes - one owned by another resource, one standalone
	// PriorityClass owned by another resource
	ownedPC := CreateTestPriorityClass("owned-pc", 1000)
	// Add owner reference to another resource
	ownedPC.OwnerReferences = []v1.OwnerReference{
		{
			Kind: "Application",
			Name: "test-application",
		},
	}

	// Standalone PriorityClass
	standalonePC := CreateTestPriorityClass("standalone-pc", 2000)

	_, err := clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), ownedPC, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PriorityClass: %v", err)
	}

	_, err = clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), standalonePC, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PriorityClass: %v", err)
	}

	// Test without filter - should return both
	filterOptsNoSkip := &filters.Options{IgnoreOwnerReferences: false}
	unusedWithoutFilter, err := processPriorityClasses(clientset, filterOptsNoSkip)
	if err != nil {
		t.Fatalf("Error retrieving unused PriorityClasses: %v", err)
	}

	if len(unusedWithoutFilter) != 2 {
		t.Errorf("Expected 2 unused PriorityClass objects without filter, got %d", len(unusedWithoutFilter))
	}

	// Test with filter - should return only standalone
	filterOptsWithSkip := &filters.Options{IgnoreOwnerReferences: true}
	unusedWithFilter, err := processPriorityClasses(clientset, filterOptsWithSkip)
	if err != nil {
		t.Fatalf("Error retrieving unused PriorityClasses: %v", err)
	}

	if len(unusedWithFilter) != 1 {
		t.Errorf("Expected 1 unused PriorityClass object with filter, got %d", len(unusedWithFilter))
	}

	if unusedWithFilter[0].Name != "standalone-pc" {
		t.Errorf("Expected standalone-pc to be unused, got %s", unusedWithFilter[0].Name)
	}
}

func TestSkipGlobalDefaultPriorityClasses(t *testing.T) {
	clientset := fake.NewClientset()

	// Create a global default PriorityClass
	globalDefaultPC := CreateTestPriorityClass("global-default-pc", 0)
	globalDefaultPC.GlobalDefault = true

	// Create a regular PriorityClass
	regularPC := CreateTestPriorityClass("regular-pc", 100)

	_, err := clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), globalDefaultPC, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PriorityClass: %v", err)
	}

	_, err = clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), regularPC, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake PriorityClass: %v", err)
	}

	// Process PriorityClasses - global default should be skipped
	unusedPriorityClasses, err := processPriorityClasses(clientset, &filters.Options{})
	if err != nil {
		t.Fatalf("Error processing PriorityClasses: %v", err)
	}

	// Should only return the regular PriorityClass as unused
	if len(unusedPriorityClasses) != 1 {
		t.Errorf("Expected 1 unused PriorityClass, got %d", len(unusedPriorityClasses))
	}

	if len(unusedPriorityClasses) > 0 && unusedPriorityClasses[0].Name != "regular-pc" {
		t.Errorf("Expected regular-pc to be unused, got %s", unusedPriorityClasses[0].Name)
	}

	// Verify that global-default-pc is not in the unused list
	for _, pc := range unusedPriorityClasses {
		if pc.Name == "global-default-pc" {
			t.Errorf("Global default PriorityClass should not be marked as unused")
		}
	}
}
