package kor

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestProcessNamespaceStatefulSets(t *testing.T) {
	resourceType := "statefulSet"
	clientset := fake.NewSimpleClientset()

	// Create a Deployment without replicas for testing
	sts1 := CreateTestStatefulSet(testNamespace, "test-sts1", 0)
	sts2 := CreateTestStatefulSet(testNamespace, "test-sts2", 1)
	_, err := clientset.AppsV1().StatefulSets("test-namespace").Create(context.TODO(), sts1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", resourceType, err)
	}

	_, err = clientset.AppsV1().StatefulSets("test-namespace").Create(context.TODO(), sts2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", resourceType, err)
	}

	// Test the getDeploymentsWithoutReplicas function
	statefulSetsWithoutReplicas, err := ProcessNamespaceStatefulSets(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(statefulSetsWithoutReplicas) != 1 {
		t.Errorf("Expected 1 deployment without replicas, got %d", len(statefulSetsWithoutReplicas))
	}

	if statefulSetsWithoutReplicas[0] != "test-sts1" {
		t.Errorf("Expected 'test-sts1', got %s", statefulSetsWithoutReplicas[0])
	}
}

// Initialize the Kubernetes API scheme
func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
