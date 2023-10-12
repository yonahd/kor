package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

type initClientFn func(t *testing.T) *fake.Clientset

func createTestNamespace(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}, metav1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	return clientset
}

func TestRetrieveUsedNS(t *testing.T) {
	tests := []struct {
		description string
		initClient  initClientFn
		expectUsed  bool
	}{
		{
			description: "unused",
			initClient:  func(t *testing.T) *fake.Clientset { return fake.NewSimpleClientset() },
			expectUsed:  false,
		},
		{
			description: "used-with-ConfigMap",
			initClient:  createTestConfigmaps,
			expectUsed:  true,
		},
		{
			description: "used-with-HorizontalPodAutoscalers",
			initClient:  createTestHpas,
			expectUsed:  true,
		},
		{
			description: "used-with-Ingress",
			initClient:  createTestIngresses,
			expectUsed:  true,
		},
		{
			description: "used-with-PodDisruptionBudget",
			initClient:  createTestPdbs,
			expectUsed:  true,
		},
		{
			description: "used-with-Secret",
			initClient:  createTestSecrets,
			expectUsed:  true,
		},
		{
			description: "used-with-Service",
			initClient:  createTestServices,
			expectUsed:  true,
		},
		{
			description: "used-with-ServiceAccount",
			initClient:  createTestServiceAccounts,
			expectUsed:  true,
		},
		{
			description: "used-with-Deployment",
			initClient:  createTestDeployments,
			expectUsed:  true,
		},
		{
			description: "used-with-PersistentVolumeClaim",
			initClient:  createTestPvcs,
			expectUsed:  true,
		},
		{
			description: "used-with-Role",
			initClient:  createTestRoles,
			expectUsed:  true,
		},
		{
			description: "used-with-StatefulSet",
			initClient:  createTestStatefulSets,
			expectUsed:  true,
		},
	}

	for _, test := range tests {
		clientset := test.initClient(t)

		usedNS, err := retrieveUsedNS(clientset, testNamespace)
		if err != nil {
			t.Errorf("test %s failed: %v", test.description, err)
		}
		if test.expectUsed && len(usedNS) == 0 {
			t.Errorf("test %s failed, expected used namespace, got unused", test.description)
		}
		if !test.expectUsed && len(usedNS) != 0 {
			t.Errorf("test %s failed, expected unused namespace, got used", test.description)
		}
	}
}

func TestGetUnusedNamespaceStructured(t *testing.T) {
	clientset := createTestNamespace(t)

	includeExcludeLists := IncludeExcludeLists{
		IncludeListStr: "",
		ExcludeListStr: "",
	}

	output, err := GetUnusedNamespacesStructured(includeExcludeLists, clientset, "json")
	if err != nil {
		t.Fatalf("Error calling GetUnusedNamespacesStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Namespaces": {testNamespace},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output: %v", actualOutput)
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
