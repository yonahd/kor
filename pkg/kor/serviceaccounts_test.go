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

var resourceType = "ServiceAccount"

func createTestServiceAccounts(t *testing.T) *fake.Clientset {

	clientset := fake.NewSimpleClientset()

	// Create a Deployment without replicas for testing
	sa1 := CreateTestServiceAccount(testNamespace, "test-sa1")
	sa2 := CreateTestServiceAccount(testNamespace, "test-sa2")
	_, err := clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", resourceType, err)
	}

	_, err = clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", resourceType, err)
	}
	return clientset
}
func TestGetServiceAccountsFromClusterRoleBindings(t *testing.T) {
	clientset := createTestServiceAccounts(t)

	clusterRoleBinding1 := CreateTestClusterRoleBinding(testNamespace, "test-crb1", "test-sa1")
	_, err := clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "clusterRoleBinding", err)
	}

	serviceAccountWithCRB, err := getServiceAccountsFromClusterRoleBindings(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(serviceAccountWithCRB) != 1 {
		t.Errorf("Expected 1 serviceAccount without CRB, got %d", len(serviceAccountWithCRB))
	}

	if serviceAccountWithCRB[0] != "test-sa1" {
		t.Errorf("Expected 'test-sa1', got %s", serviceAccountWithCRB[0])
	}

}

func TestGetServiceAccountsFromRoleBindings(t *testing.T) {
	clientset := createTestServiceAccounts(t)

	roleBinding1 := CreateTestRoleBinding(testNamespace, "test-crb1", "test-sa1")
	_, err := clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), roleBinding1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "roleBinding", err)
	}

	serviceAccountWithRB, err := getServiceAccountsFromRoleBindings(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(serviceAccountWithRB) != 1 {
		t.Errorf("Expected 1 serviceAccount without CRB, got %d", len(serviceAccountWithRB))
	}

	if serviceAccountWithRB[0] != "test-sa1" {
		t.Errorf("Expected 'test-sa1', got %s", serviceAccountWithRB[0])
	}
}

func TestRetrieveUsedSA(t *testing.T) {
	var volumeList []corev1.Volume
	clientset := createTestServiceAccounts(t)

	testVolume := CreateTestVolume("test-volume1")
	volumeList = append(volumeList, *testVolume)

	podWithSA := CreateTestPod(testNamespace, "test-pod1", "test-sa1", volumeList)
	_, err := clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), podWithSA, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pod", err)
	}
	serviceAccountUsedByPod, _, _, err := retrieveUsedSA(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(serviceAccountUsedByPod) != 2 {
		t.Errorf("Expected 2 serviceAccount Used by pod, got %d", len(serviceAccountUsedByPod))
	}

	if serviceAccountUsedByPod[0] != "test-sa1" || serviceAccountUsedByPod[1] != "default" {
		t.Errorf("Expected 'test-sa1' and 'default', got %s", serviceAccountUsedByPod[0])
	}

}

func TestRetrieveServiceAccountNames(t *testing.T) {
	clientset := createTestServiceAccounts(t)
	serviceAccountNames, err := retrieveServiceAccountNames(clientset, testNamespace)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(serviceAccountNames) != 2 {
		t.Errorf("Expected 2 serviceAccounts , got %d", len(serviceAccountNames))
	}
}

func TestProcessNamespaceSA(t *testing.T) {
	clientset := createTestServiceAccounts(t)
	var volumeList []corev1.Volume

	testVolume := CreateTestVolume("test-volume1")
	volumeList = append(volumeList, *testVolume)

	roleBinding1 := CreateTestRoleBinding(testNamespace, "test-crb1", "test-sa1")
	_, err := clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), roleBinding1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "roleBinding", err)
	}

	podWithSA := CreateTestPod(testNamespace, "test-pod1", "test-sa1", volumeList)
	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), podWithSA, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pod", err)
	}

	unusedServiceAccounts, err := processNamespaceSA(clientset, testNamespace)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(unusedServiceAccounts) != 1 {
		t.Errorf("Expected 2 serviceAccount Used by pod, got %d", len(unusedServiceAccounts))
	}

	if unusedServiceAccounts[0] != "test-sa2" {
		t.Errorf("Expected 'test-sa2', got %s", unusedServiceAccounts[0])
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
