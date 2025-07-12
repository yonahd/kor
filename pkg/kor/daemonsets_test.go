package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestDaemonSets(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	ds1 := CreateTestDaemonSet(testNamespace, "test-ds1", AppLabels, &appsv1.DaemonSetStatus{
		CurrentNumberScheduled: 0,
	})
	_, err = clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), ds1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "DaemonSet", err)
	}

	ds2 := CreateTestDaemonSet(testNamespace, "test-ds2", AppLabels, &appsv1.DaemonSetStatus{
		CurrentNumberScheduled: 1,
	})
	_, err = clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), ds2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "DaemonSet", err)
	}

	ds3 := CreateTestDaemonSet(testNamespace, "test-ds3", UsedLabels, &appsv1.DaemonSetStatus{
		CurrentNumberScheduled: 0,
	})
	_, err = clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), ds3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "DaemonSet", err)
	}

	ds4 := CreateTestDaemonSet(testNamespace, "test-ds4", UnusedLabels, &appsv1.DaemonSetStatus{
		CurrentNumberScheduled: 1,
	})
	_, err = clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), ds4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "DaemonSet", err)
	}

	return clientset
}

func createTestDaemonSetsWithOwnerReferences(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// DaemonSet with ownerReferences (should be ignored when --ignore-owner-references is true)
	dsWithOwner := CreateTestDaemonSet(testNamespace, "test-ds-with-owner", AppLabels, &appsv1.DaemonSetStatus{
		CurrentNumberScheduled: 0,
	})
	dsWithOwner.OwnerReferences = []v1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-deployment",
			UID:        "test-uid",
		},
	}
	_, err = clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), dsWithOwner, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake DaemonSet with ownerReferences: %v", err)
	}

	// DaemonSet without ownerReferences (should be included)
	dsWithoutOwner := CreateTestDaemonSet(testNamespace, "test-ds-without-owner", AppLabels, &appsv1.DaemonSetStatus{
		CurrentNumberScheduled: 0,
	})
	_, err = clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), dsWithoutOwner, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake DaemonSet without ownerReferences: %v", err)
	}

	return clientset
}

func TestProcessNamespaceDaemonSets(t *testing.T) {
	clientset := createTestDaemonSets(t)

	daemonSetsWithoutReplicas, err := processNamespaceDaemonSets(clientset, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(daemonSetsWithoutReplicas) != 2 {
		t.Errorf("Expected 1 DaemonSet without replicas, got %d", len(daemonSetsWithoutReplicas))
	}

	if daemonSetsWithoutReplicas[0].Name != "test-ds1" && daemonSetsWithoutReplicas[1].Name != "test-ds4" {
		t.Errorf("Expected 'test-ds1', 'test-ds4', got %s, %s", daemonSetsWithoutReplicas[0], daemonSetsWithoutReplicas[1])
	}
}

func TestProcessNamespaceDaemonSetsWithOwnerReferences(t *testing.T) {
	clientset := createTestDaemonSetsWithOwnerReferences(t)

	// Test with --ignore-owner-references=false (default behavior)
	daemonSetsWithoutReplicas, err := processNamespaceDaemonSets(clientset, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should include both DaemonSets (with and without ownerReferences)
	if len(daemonSetsWithoutReplicas) != 2 {
		t.Errorf("Expected 2 DaemonSets without replicas, got %d", len(daemonSetsWithoutReplicas))
	}

	// Test with --ignore-owner-references=true
	daemonSetsWithoutReplicas, err = processNamespaceDaemonSets(clientset, testNamespace, &filters.Options{IgnoreOwnerReferences: true}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should only include DaemonSet without ownerReferences
	if len(daemonSetsWithoutReplicas) != 1 {
		t.Errorf("Expected 1 DaemonSet without replicas when ignoring ownerReferences, got %d", len(daemonSetsWithoutReplicas))
	}

	if daemonSetsWithoutReplicas[0].Name != "test-ds-without-owner" {
		t.Errorf("Expected 'test-ds-without-owner', got %s", daemonSetsWithoutReplicas[0].Name)
	}
}

func TestGetUnusedDaemonSetsStructured(t *testing.T) {
	clientset := createTestDaemonSets(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedDaemonSets(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedDaemonSetsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"DaemonSet": {
				"test-ds1",
				"test-ds4",
			},
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

func TestGetUnusedDaemonSetsStructuredWithOwnerReferences(t *testing.T) {
	clientset := createTestDaemonSetsWithOwnerReferences(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	// Test with --ignore-owner-references=true
	output, err := GetUnusedDaemonSets(&filters.Options{IgnoreOwnerReferences: true}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedDaemonSetsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"DaemonSet": {
				"test-ds-without-owner",
			},
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

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
