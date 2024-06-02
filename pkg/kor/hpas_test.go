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

func createTestHpas(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	deploymentName := "test-deployment"

	deployment1 := CreateTestDeployment(testNamespace, deploymentName, 1, AppLabels)

	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	hpa1 := CreateTestHpa(testNamespace, "test-hpa1", deploymentName, 1, 1, AppLabels)
	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(context.TODO(), hpa1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Hpa: %v", err)
	}

	hpa2 := CreateTestHpa(testNamespace, "test-hpa2", "non-existing-deployment", 1, 1, AppLabels)
	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(context.TODO(), hpa2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Hpa: %v", err)
	}

	hpa3 := CreateTestHpa(testNamespace, "test-hpa3", deploymentName, 1, 1, UsedLabels)
	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(context.TODO(), hpa3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Hpa: %v", err)
	}

	hpa4 := CreateTestHpa(testNamespace, "test-hpa4", "non-existing-deployment", 1, 1, UnusedLabels)
	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(context.TODO(), hpa4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Hpa: %v", err)
	}

	return clientset
}

func TestExtractUnusedHpas(t *testing.T) {
	clientset := createTestHpas(t)

	unusedHpas, err := processNamespaceHpas(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedHpas) != 2 {
		t.Errorf("Expected 1 unused HPA, got %d", len(unusedHpas))
	}

	if unusedHpas[0].Name != "test-hpa2" && unusedHpas[1].Name != "test-hpa4" {
		t.Errorf("Expected 'test-hpa2', 'test-hpa4', got %s, %s", unusedHpas[0], unusedHpas[1])
	}
}

func TestGetUnusedHpasStructured(t *testing.T) {
	clientset := createTestHpas(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedHpas(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedHpasStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Hpa": {
				"test-hpa2",
				"test-hpa4",
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
