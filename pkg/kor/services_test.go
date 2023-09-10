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

func TestGetEndpointsWithoutSubsets(t *testing.T) {
	// Create a fake Kubernetes client for testing
	clientset := fake.NewSimpleClientset()

	// Create a Deployment without replicas for testing
	endpoint1 := CreateTestEndpoint("test-namespace", "test-endpoint1", 0)
	endpoint2 := CreateTestEndpoint("test-namespace", "test-endpoint2", 1)
	_, err := clientset.CoreV1().Endpoints("test-namespace").Create(context.TODO(), endpoint1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	_, err = clientset.CoreV1().Endpoints("test-namespace").Create(context.TODO(), endpoint2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	servicesWithoutEndpoints, err := ProcessNamespaceServices(clientset, "test-namespace")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(servicesWithoutEndpoints) != 1 {
		t.Errorf("Expected 1 service without endpoint, got %d", len(servicesWithoutEndpoints))
	}

	if servicesWithoutEndpoints[0] != "test-endpoint1" {
		t.Errorf("Expected 'test-endpoint1', got %s", servicesWithoutEndpoints[0])
	}
}

// Initialize the Kubernetes API scheme
func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
