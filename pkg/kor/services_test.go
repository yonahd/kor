package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func createTestServices(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	appLabels := map[string]string{}
	usedLabels := map[string]string{"kor/used": "true"}
	unusedLabels := map[string]string{"kor/used": "false"}

	endpoint1 := CreateTestEndpoint(testNamespace, "test-endpoint1", 0, appLabels)
	_, err = clientset.CoreV1().Endpoints(testNamespace).Create(context.TODO(), endpoint1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	endpoint2 := CreateTestEndpoint(testNamespace, "test-endpoint2", 1, appLabels)
	_, err = clientset.CoreV1().Endpoints(testNamespace).Create(context.TODO(), endpoint2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	endpoint3 := CreateTestEndpoint(testNamespace, "test-endpoint3", 1, usedLabels)
	_, err = clientset.CoreV1().Endpoints(testNamespace).Create(context.TODO(), endpoint3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	endpoint4 := CreateTestEndpoint(testNamespace, "test-endpoint4", 1, unusedLabels)
	_, err = clientset.CoreV1().Endpoints(testNamespace).Create(context.TODO(), endpoint4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	return clientset
}

func TestGetEndpointsWithoutSubsets(t *testing.T) {
	clientset := createTestServices(t)

	servicesWithoutEndpoints, err := ProcessNamespaceServices(clientset, testNamespace, &FilterOptions{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(servicesWithoutEndpoints) != 2 {
		t.Errorf("Expected 2 service without endpoint, got %d", len(servicesWithoutEndpoints))
	}

	if servicesWithoutEndpoints[0] != "test-endpoint1" || servicesWithoutEndpoints[1] != "test-endpoint4" {
		t.Errorf("Expected 'test-endpoint1', got %s", servicesWithoutEndpoints[0])
	}
}

func TestGetUnusedServicesStructured(t *testing.T) {
	clientset := createTestServices(t)

	includeExcludeLists := IncludeExcludeLists{
		IncludeListStr: testNamespace,
		ExcludeListStr: "",
	}

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
	}

	output, err := GetUnusedServices(includeExcludeLists, &FilterOptions{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedServicesStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Services": {"test-endpoint1", "test-endpoint4"},
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
