package kor

import (
	"context"
	"encoding/json"
	"github.com/yonahd/kor/pkg/filters"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func createTestClusterRoles(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	clusterRole1 := CreateTestClusterRole("test-clusterRole1")
	clusterRole2 := CreateTestClusterRole("test-clusterRole2")
	clusterRole3 := CreateTestClusterRole("test-clusterRole3")
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	testRoleRef2 := CreateTestRoleRef("test-clusterRole2")
	testClusterRoleBinding := CreateTestClusterRoleBindingRoleRef(testNamespace, "test-rb2", "test-sa", testRoleRef2)
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), testClusterRoleBinding, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	testRoleRef3 := CreateTestRoleRef("test-clusterRole3")
	testRoleBinding := CreateTestRoleBinding(testNamespace, "test-rb", "test-sa", testRoleRef3)
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), testRoleBinding, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	return clientset
}
func TestRetrieveUsedClusterRoles(t *testing.T) {
	clientset := createTestClusterRoles(t)

	usedClusterRoles, err := retrieveUsedClusterRoles(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedClusterRoles) != 2 {
		t.Errorf("Expected 1 used cluster role, got %d", len(usedClusterRoles))
	}

	expectedRoles := []string{"test-clusterRole1", "test-clusterRole3"}
	if reflect.DeepEqual(usedClusterRoles, expectedRoles) {
		t.Errorf("Expected 'test-role1', got %s", usedClusterRoles[0])
	}
}

func TestRetrieveClusterRoleNames(t *testing.T) {
	clientset := createTestClusterRoles(t)
	allRoles, err := retrieveClusterRoleNames(clientset)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(allRoles) != 3 {
		t.Errorf("Expected 2 roles, got %d", len(allRoles))
	}
}

func TestProcessNamespaceClusterRoles(t *testing.T) {
	clientset := createTestClusterRoles(t)

	unusedClusterRoles, err := processNamespaceClusterRoles(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedClusterRoles) != 1 {
		t.Errorf("Expected 1 unused role, got %d", len(unusedClusterRoles))
	}

	if unusedClusterRoles[0] != "test-clusterRole1" {
		t.Errorf("Expected 'test-clusterRole1', got %s", unusedClusterRoles[0])
	}
}

func TestGetUnusedClusterRolesStructured(t *testing.T) {
	clientset := createTestClusterRoles(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
	}

	output, err := GetUnusedClusterRoles(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedRolesStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ClusterRoles": {"test-clusterRole1"},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match \n actualOutput:\n %s \n expectedOutput:\n %s", actualOutput, expectedOutput)
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
