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

func TestProcessNamespaceDeployments(t *testing.T) {
	// Create a fake Kubernetes client for testing
	clientset := fake.NewSimpleClientset()
	appLabels := map[string]string{}
	// Create a Deployment without replicas for testing
	deployment1 := CreateTestDeployment("test-namespace", "test-deployment1", 0, appLabels)
	deployment2 := CreateTestDeployment("test-namespace", "test-deployment2", 1, appLabels)
	_, err := clientset.AppsV1().Deployments("test-namespace").Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	_, err = clientset.AppsV1().Deployments("test-namespace").Create(context.TODO(), deployment2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	// Test the getDeploymentsWithoutReplicas function
	deploymentsWithoutReplicas, err := ProcessNamespaceDeployments(clientset, "test-namespace")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(deploymentsWithoutReplicas) != 1 {
		t.Errorf("Expected 1 deployment without replicas, got %d", len(deploymentsWithoutReplicas))
	}

	if deploymentsWithoutReplicas[0] != "test-deployment1" {
		t.Errorf("Expected 'test-deployment1', got %s", deploymentsWithoutReplicas[0])
	}
}

// Initialize the Kubernetes API scheme
func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
