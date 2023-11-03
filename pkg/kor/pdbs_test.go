package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func createTestPdbs(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	appLabels1 := map[string]string{
		"app": "my-app",
	}
	appLabels2 := map[string]string{}

	pdb1 := CreateTestPdb(testNamespace, "test-pdb1", appLabels1)
	pdb2 := CreateTestPdb(testNamespace, "test-pdb2", appLabels1)
	pdb3 := CreateTestPdb(testNamespace, "test-pdb3", appLabels2)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	deployment1 := CreateTestDeployment(testNamespace, "test-deployment2", 1, appLabels1)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	sts1 := CreateTestStatefulSet(testNamespace, "test-sts2", 1, appLabels1)
	_, err = clientset.AppsV1().StatefulSets(testNamespace).Create(context.TODO(), sts1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "StatefulSet", err)
	}

	return clientset
}

func TestProcessNamespacePdbs(t *testing.T) {
	clientset := createTestPdbs(t)

	unusedPdbs, err := processNamespacePdbs(clientset, testNamespace, &FilterOptions{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedPdbs) != 1 {
		t.Errorf("Expected 1 unused pdb, got %d", len(unusedPdbs))
	}

	if unusedPdbs[0] != "test-pdb3" {
		t.Errorf("Expected 'test-pdb3', got %s", unusedPdbs[0])
	}
}

func TestGetUnusedPdbsStructured(t *testing.T) {
	clientset := createTestPdbs(t)

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

	output, err := GetUnusedPdbs(includeExcludeLists, &FilterOptions{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedPdbsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Pdb": {"test-pdb3"},
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
