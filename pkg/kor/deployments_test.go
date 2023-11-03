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

func createTestDeployments(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	appLabels := map[string]string{}

	deployment1 := CreateTestDeployment(testNamespace, "test-deployment1", 0, appLabels)
	deployment2 := CreateTestDeployment(testNamespace, "test-deployment2", 1, appLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	return clientset
}

func TestProcessNamespaceDeployments(t *testing.T) {
	clientset := createTestDeployments(t)

	deploymentsWithoutReplicas, err := ProcessNamespaceDeployments(clientset, testNamespace, &FilterOptions{})
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

func TestGetUnusedDeploymentsStructured(t *testing.T) {
	clientset := createTestDeployments(t)

	includeExcludeLists := IncludeExcludeLists{
		IncludeListStr: "",
		ExcludeListStr: "",
	}

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
	}

	output, err := GetUnusedDeployments(includeExcludeLists, &FilterOptions{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedDeploymentsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Deployments": {"test-deployment1"},
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
