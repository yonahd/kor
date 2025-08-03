package kor

import (
	"context"
	"encoding/json"
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

func createTestAllResourcesClient(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// Create test resources using the refactored functions
	createTestDeployments(clientset, t)
	createTestServices(clientset, t)
	createTestConfigmaps(clientset, t)

	return clientset
}

func TestGetUnusedAllNamespaced(t *testing.T) {
	clientset := createTestAllResourcesClient(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	// Verify that the output contains data for the test namespace
	if _, exists := actualOutput[testNamespace]; !exists {
		t.Errorf("Expected output to contain namespace %s", testNamespace)
	}

	// Verify that deployments are included (we know from deployment tests that test-deployment1 should be unused)
	if deployments, exists := actualOutput[testNamespace]["Deployment"]; exists {
		found := false
		for _, deployment := range deployments {
			if deployment == "test-deployment1" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find unused deployment 'test-deployment1' in output")
		}
	} else {
		t.Errorf("Expected output to contain Deployment resources")
	}

	// Verify that services are included (we know from service tests that test-endpoint1 should be unused)
	if services, exists := actualOutput[testNamespace]["Service"]; exists {
		found := false
		for _, service := range services {
			if service == "test-endpoint1" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find unused service 'test-endpoint1' in output")
		}
	} else {
		t.Errorf("Expected output to contain Service resources")
	}

	// Verify that configmaps are included (we know from configmap tests that some should be unused)
	if configmaps, exists := actualOutput[testNamespace]["ConfigMap"]; exists {
		if len(configmaps) == 0 {
			t.Errorf("Expected to find some unused configmaps in output")
		}
	} else {
		t.Errorf("Expected output to contain ConfigMap resources")
	}
}

func TestGetUnusedAllStructured(t *testing.T) {
	clientset := createTestAllResourcesClient(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "resource",
	}

	output, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	// When grouping by resource, the structure should be resource -> namespace -> list
	// Verify that we have different resource types
	resourceTypes := []string{"Deployment", "Service", "ConfigMap"}
	for _, resourceType := range resourceTypes {
		if _, exists := actualOutput[resourceType]; !exists {
			t.Errorf("Expected output to contain resource type %s", resourceType)
		}
	}

	// Verify that deployments contain our test namespace
	if deployments, exists := actualOutput["Deployment"]; exists {
		if _, exists := deployments[testNamespace]; !exists {
			t.Errorf("Expected Deployment resources to contain namespace %s", testNamespace)
		}
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
}