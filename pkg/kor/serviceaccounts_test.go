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

	"github.com/yonahd/kor/pkg/filters"
)

func createTestServiceAccounts(t *testing.T) *fake.Clientset {

	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	sa1 := CreateTestServiceAccount(testNamespace, "test-sa1", AppLabels)
	_, err = clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ServiceAccount", err)
	}

	sa2 := CreateTestServiceAccount(testNamespace, "test-sa2", AppLabels)
	_, err = clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ServiceAccount", err)
	}

	sa3 := CreateTestServiceAccount(testNamespace, "test-sa3", UsedLabels)
	_, err = clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ServiceAccount", err)
	}

	sa4 := CreateTestServiceAccount(testNamespace, "test-sa4", UnusedLabels)
	_, err = clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ServiceAccount", err)
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

	testRoleRef := CreateTestRoleRef("test-role")
	roleBinding1 := CreateTestRoleBinding(testNamespace, "test-crb1", "test-sa1", testRoleRef)
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

	testVolume := CreateTestVolume("test-volume1", "test-pvc")
	volumeList = append(volumeList, *testVolume)

	podWithSA := CreateTestPod(testNamespace, "test-pod1", "test-sa1", volumeList, AppLabels)
	_, err := clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), podWithSA, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pod", err)
	}
	serviceAccountUsedByPod, _, _, err := retrieveUsedSA(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(serviceAccountUsedByPod) != 1 {
		t.Errorf("Expected 2 serviceAccount Used by pod, got %d", len(serviceAccountUsedByPod))
	}

	if serviceAccountUsedByPod[0] != "test-sa1" {
		t.Errorf("Expected 'test-sa1', got %s", serviceAccountUsedByPod[0])
	}

}

func TestRetrieveServiceAccountNames(t *testing.T) {
	clientset := createTestServiceAccounts(t)
	serviceAccountNames, _, err := retrieveServiceAccountNames(clientset, testNamespace, &filters.Options{})
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

	testVolume := CreateTestVolume("test-volume1", "test-pvc")
	volumeList = append(volumeList, *testVolume)

	testRoleRef := CreateTestRoleRef("test-role")

	roleBinding1 := CreateTestRoleBinding(testNamespace, "test-crb1", "test-sa1", testRoleRef)
	_, err := clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), roleBinding1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "roleBinding", err)
	}

	podWithSA := CreateTestPod(testNamespace, "test-pod1", "test-sa1", volumeList, AppLabels)
	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), podWithSA, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Pod", err)
	}

	unusedServiceAccounts, err := processNamespaceSA(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(unusedServiceAccounts) != 2 {
		t.Errorf("Expected 2 serviceAccount Used by pod, got %d", len(unusedServiceAccounts))
	}

	if unusedServiceAccounts[0].Name != "test-sa2" {
		t.Errorf("Expected 'test-sa2', got %s", unusedServiceAccounts[0])
	}
}

func TestGetUnusedServiceAccountsStructured(t *testing.T) {
	clientset := createTestServiceAccounts(t)

	clusterRoleBinding1 := CreateTestClusterRoleBinding(testNamespace, "test-crb1", "test-sa1")
	_, err := clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "clusterRoleBinding", err)
	}

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedServiceAccounts(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedServiceAccountsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ServiceAccount": {
				"test-sa2",
				"test-sa4",
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
