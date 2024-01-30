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

func createTestPvcs(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()
	var volumeList []corev1.Volume

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	pvc1 := CreateTestPvc(testNamespace, "test-pvc1")
	pvc2 := CreateTestPvc(testNamespace, "test-pvc2")
	_, err = clientset.CoreV1().PersistentVolumeClaims(testNamespace).Create(context.TODO(), pvc1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pvc", err)
	}

	_, err = clientset.CoreV1().PersistentVolumeClaims(testNamespace).Create(context.TODO(), pvc2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pvc", err)
	}

	testVolume := CreateTestVolume("test-volume", "test-pvc1")
	volumeList = append(volumeList, *testVolume)
	testPod := CreateTestPod(testNamespace, "test-pod", "test-sa", volumeList)

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), testPod, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pvc", err)
	}

	return clientset
}

func TestRetrieveUsedPvcs(t *testing.T) {
	clientset := createTestPvcs(t)
	usedPvcs, err := retrieveUsedPvcs(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedPvcs) != 1 {
		t.Errorf("Expected 1 used pvc, got %d", len(usedPvcs))
	}

	if usedPvcs[0] != "test-pvc1" {
		t.Errorf("Expected 'test-pvc1', got %s", usedPvcs[0])
	}
}

func TestProcessNamespacePvcs(t *testing.T) {
	clientset := createTestPvcs(t)
	usedPvcs, err := processNamespacePvcs(clientset, testNamespace, &FilterOptions{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedPvcs) != 1 {
		t.Errorf("Expected 1 unused pvc, got %d", len(usedPvcs))
	}

	if usedPvcs[0] != "test-pvc2" {
		t.Errorf("Expected 'test-pvc2', got %s", usedPvcs[0])
	}
}

func TestGetUnusedPvcsStructured(t *testing.T) {
	clientset := createTestPvcs(t)

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

	output, err := GetUnusedPvcs(includeExcludeLists, &FilterOptions{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedPvcsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Pvc": {"test-pvc2"},
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
