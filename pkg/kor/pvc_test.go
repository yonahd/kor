package kor

import (
	"context"
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

	pvc1 := CreateTestPvc(testNamespace, "test-pvc1")
	pvc2 := CreateTestPvc(testNamespace, "test-pvc2")
	_, err := clientset.CoreV1().PersistentVolumeClaims(testNamespace).Create(context.TODO(), pvc1, v1.CreateOptions{})
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

func TestRetreiveUsedPvcs(t *testing.T) {
	clientset := createTestPvcs(t)
	usedPvcs, err := retreiveUsedPvcs(clientset, testNamespace)
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
	usedPvcs, err := processNamespacePvcs(clientset, testNamespace)
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

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
