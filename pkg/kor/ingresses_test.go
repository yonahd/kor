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

func createTestIngresses(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	service1 := CreateTestService(testNamespace, "my-service-1")

	_, err = clientset.CoreV1().Services(testNamespace).Create(context.TODO(), service1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Service", err)
	}

	ingress1 := CreateTestIngress(testNamespace, "test-ingress-1", "my-service-1", "test-secret", AppLabels)
	_, err = clientset.NetworkingV1().Ingresses(testNamespace).Create(context.TODO(), ingress1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Ingress", err)
	}

	ingress2 := CreateTestIngress(testNamespace, "test-ingress-2", "my-service-2", "test-secret", AppLabels)
	_, err = clientset.NetworkingV1().Ingresses(testNamespace).Create(context.TODO(), ingress2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Ingress", err)
	}

	ingress3 := CreateTestIngress(testNamespace, "test-ingress-3", "my-service-2", "test-secret", UsedLabels)
	_, err = clientset.NetworkingV1().Ingresses(testNamespace).Create(context.TODO(), ingress3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Ingress", err)
	}

	ingress4 := CreateTestIngress(testNamespace, "test-ingress-4", "my-service-1", "test-secret", UnusedLabels)
	_, err = clientset.NetworkingV1().Ingresses(testNamespace).Create(context.TODO(), ingress4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Ingress", err)
	}

	return clientset
}

func TestRetrieveUsedIngress(t *testing.T) {
	clientset := createTestIngresses(t)

	usedIngresses, err := retrieveUsedIngress(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedIngresses) != 2 {
		t.Errorf("Expected 2 used Ingress objects, got %d", len(usedIngresses))
	}

	if !contains(usedIngresses, "test-ingress-1") {
		t.Error("Expected specific Ingress objects in the list")
	}
}

func TestGetUnusedIngressesStructured(t *testing.T) {
	clientset := createTestIngresses(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedIngresses(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedIngressesStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Ingress": {
				"test-ingress-2",
				"test-ingress-4",
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
