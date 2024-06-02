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

	"github.com/yonahd/kor/pkg/filters"
)

func createTestDeployments(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	deployment1 := CreateTestDeployment(testNamespace, "test-deployment1", 0, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	deployment2 := CreateTestDeployment(testNamespace, "test-deployment2", 1, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	deployment3 := CreateTestDeployment(testNamespace, "test-deployment3", 0, UsedLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	deployment4 := CreateTestDeployment(testNamespace, "test-deployment4", 1, UnusedLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	return clientset
}

func TestProcessNamespaceDeployments(t *testing.T) {
	clientset := createTestDeployments(t)

	deploymentsWithoutReplicas, err := processNamespaceDeployments(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(deploymentsWithoutReplicas) != 2 {
		t.Errorf("Expected 1 deployment without replicas, got %d", len(deploymentsWithoutReplicas))
	}

	if deploymentsWithoutReplicas[0].Name != "test-deployment1" && deploymentsWithoutReplicas[1].Name != "test-deployment4" {
		t.Errorf("Expected 'test-deployment1', 'test-deployment4',got %s, %s", deploymentsWithoutReplicas[0], deploymentsWithoutReplicas[1])
	}
}

func TestGetUnusedDeploymentsStructured(t *testing.T) {
	clientset := createTestDeployments(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedDeployments(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedDeploymentsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Deployment": {
				"test-deployment1",
				"test-deployment4",
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
