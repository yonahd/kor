package kor

import (
	"context"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestProcessNamespacePdbs(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	appLabels1 := map[string]string{
		"app": "my-app",
	}
	appLabels2 := map[string]string{}

	pdb1 := CreateTestPdb(testNamespace, "test-pdb1", appLabels1)
	pdb2 := CreateTestPdb(testNamespace, "test-pdb2", appLabels1)
	pdb3 := CreateTestPdb(testNamespace, "test-pdb3", appLabels2)
	_, err := clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb1, v1.CreateOptions{})
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

	deployment1 := CreateTestDeployment("test-namespace", "test-deployment2", 1, appLabels1)
	_, err = clientset.AppsV1().Deployments("test-namespace").Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	sts1 := CreateTestStatefulSet(testNamespace, "test-sts2", 1, appLabels1)
	_, err = clientset.AppsV1().StatefulSets(testNamespace).Create(context.TODO(), sts1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "StatefulSet", err)
	}

	unusedPdbs, err := processNamespacePdbs(clientset, testNamespace)
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
