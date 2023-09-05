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

func TestRetrieveUsedIngress(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Create a fake Ingress with multiple rules and backends for testing
	service1 := CreateTestService("test-namespace", "my-service-1")
	ingress1 := CreateTestIngress("test-namespace", "test-ingress-1", "my-service-1")
	ingress2 := CreateTestIngress("test-namespace", "test-ingress-2", "my-service-2")

	// Create the Ingresses in the fake clientset
	_, _ = clientset.CoreV1().Services("test-namespace").Create(context.TODO(), service1, v1.CreateOptions{})
	_, _ = clientset.NetworkingV1().Ingresses("test-namespace").Create(context.TODO(), ingress1, v1.CreateOptions{})
	_, _ = clientset.NetworkingV1().Ingresses("test-namespace").Create(context.TODO(), ingress2, v1.CreateOptions{})

	// Test the retrieveUsedIngress function
	usedIngresses, err := retrieveUsedIngress(clientset, "test-namespace")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedIngresses) != 1 {
		t.Errorf("Expected 1 used Ingress objects, got %d", len(usedIngresses))
	}

	if !contains(usedIngresses, "test-ingress-1") {
		t.Error("Expected specific Ingress objects in the list")
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Initialize the Kubernetes API scheme
func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
