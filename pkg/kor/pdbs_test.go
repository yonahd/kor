package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
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

	pdb1 := CreateTestPdb(testNamespace, "test-pdb1", appLabels1, AppLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	pdb2 := CreateTestPdb(testNamespace, "test-pdb2", appLabels1, AppLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	pdb3 := CreateTestPdb(testNamespace, "test-pdb3", AppLabels, AppLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	pdb4 := CreateTestPdb(testNamespace, "test-pdb4", AppLabels, UsedLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	pdb5 := CreateTestPdb(testNamespace, "test-pdb5", AppLabels, UnusedLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb5, v1.CreateOptions{})
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

	unusedPdbs, err := processNamespacePdbs(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedPdbs) != 2 {
		t.Errorf("Expected 2 unused pdb, got %d", len(unusedPdbs))
	}

	if unusedPdbs[0].Name != "test-pdb3" && unusedPdbs[1].Name != "test-pdb5" {
		t.Errorf("Expected 'test-pdb3', got %s", unusedPdbs[0])
	}
}

func TestGetUnusedPdbsStructured(t *testing.T) {
	clientset := createTestPdbs(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedPdbs(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedPdbsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Pdb": {
				"test-pdb3",
				"test-pdb5",
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
