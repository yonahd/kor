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
)

func createTestHpas(clientset *fake.Clientset, t *testing.T) *fake.Clientset {

	deploymentName := "test-deployment"
	appLabels := map[string]string{}

	deployment1 := CreateTestDeployment(testNamespace, deploymentName, 1, appLabels)
	hpa1 := CreateTestHpa(testNamespace, "test-hpa1", deploymentName, 1, 1)

	hpa2 := CreateTestHpa(testNamespace, "test-hpa2", "non-existing-deployment", 1, 1)
	_, err := clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(context.TODO(), hpa1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Hpa: %v", err)
	}

	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(context.TODO(), hpa2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Hpa: %v", err)
	}

	return clientset
}

func createTestHpaClient(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	createTestHpas(clientset, t)

	return clientset
}

func TestExtractUnusedHpas(t *testing.T) {
	clientset := createTestHpaClient(t)

	unusedHpas, err := extractUnusedHpas(clientset, testNamespace, &FilterOptions{})
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

func TestGetUnusedHpasStructured(t *testing.T) {
	clientset := createTestHpaClient(t)

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

	output, err := GetUnusedHpas(includeExcludeLists, &FilterOptions{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedHpasStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Hpa": {"test-hpa2"},
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
