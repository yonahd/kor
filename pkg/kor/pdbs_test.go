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

var testNamespace2 = "test-namespace2"

func createTestPdbs(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()
	namespaces := []string{testNamespace, testNamespace2}
	var err error

	for _, ns := range namespaces {
		_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: ns},
		}, v1.CreateOptions{})

		if err != nil {
			t.Fatalf("Error creating namespace %s: %v", ns, err)
		}
	}

	appLabels1 := map[string]string{
		"app": "my-app",
	}

	appLabels2 := map[string]string{
		"unused-app": "my-unused-app",
	}

	// Empty selector
	appLabels3 := map[string]string{}

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

	// Unused PDB - no matching templates / workloads
	pdb3 := CreateTestPdb(testNamespace, "test-pdb3", appLabels2, AppLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	pdb4 := CreateTestPdb(testNamespace, "test-pdb4", AppLabels, UsedLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	// Unused PDB - kor/used: false
	pdb5 := CreateTestPdb(testNamespace, "test-pdb5", AppLabels, UnusedLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb5, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	pdb6 := CreateTestPdb(testNamespace, "test-pdb6", appLabels3, AppLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb6, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pdb", err)
	}

	// Unused PDB - empty selector with 0 pods running
	pdb7 := CreateTestPdb(testNamespace2, "test-pdb7", appLabels3, AppLabels)
	_, err = clientset.PolicyV1().PodDisruptionBudgets(testNamespace2).Create(context.TODO(), pdb7, v1.CreateOptions{})
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

	pod1 := CreateTestPod(testNamespace, "test-arbitrary-pod", "", nil, appLabels1)
	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pod", err)
	}

	return clientset
}

func TestProcessNamespacePdbs(t *testing.T) {
	clientset := createTestPdbs(t)
	namespaces := []string{testNamespace, testNamespace2}
	expectedUnusedPdbs := []string{"test-pdb3", "test-pdb5", "test-pdb7"}
	totalUnusedPdbs := []ResourceInfo{}

	for _, ns := range namespaces {
		unusedPdbs, err := processNamespacePdbs(clientset, ns, &filters.Options{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		totalUnusedPdbs = append(totalUnusedPdbs, unusedPdbs...)
	}

	if len(totalUnusedPdbs) != len(expectedUnusedPdbs) {
		t.Errorf("Expected %d unused pdbs, got %d", len(expectedUnusedPdbs), len(totalUnusedPdbs))
	}

	for i, expected := range expectedUnusedPdbs {
		if totalUnusedPdbs[i].Name != expected {
			t.Errorf("Expected '%s' in unused pdbs, got '%s'", expected, totalUnusedPdbs[i].Name)
		}
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
		testNamespace2: {
			"Pdb": {
				"test-pdb7",
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
