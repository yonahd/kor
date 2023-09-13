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

func TestExtractUnusedHpas(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	deploymentName := "test-deployment"
	appLabels := map[string]string{}

	// Create a Deployment without replicas for testing
	deployment1 := CreateTestDeployment("test-namespace", deploymentName, 1, appLabels)
	hpa1 := CreateTestHpa("test-namespace", "test-hpa1", deploymentName, 1, 1)

	hpa2 := CreateTestHpa("test-namespace", "test-hpa2", "non-existing-deployment", 1, 1)
	_, err := clientset.AppsV1().Deployments("test-namespace").Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers("test-namespace").Create(context.TODO(), hpa1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Hpa: %v", err)
	}

	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers("test-namespace").Create(context.TODO(), hpa2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Hpa: %v", err)
	}

	// Test the getDeploymentsWithoutReplicas function
	unusedHpas, err := extractUnusedHpas(clientset, "test-namespace")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedHpas) != 1 {
		t.Errorf("Expected 1 unused HPA, got %d", len(unusedHpas))
	}

	if unusedHpas[0] != "test-hpa2" {
		t.Errorf("Expected 'test-hpa2', got %s", unusedHpas[0])
	}
}

// Initialize the Kubernetes API scheme
func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
