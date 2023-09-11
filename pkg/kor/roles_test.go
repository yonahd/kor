package kor

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func createTestRoles(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()
	role1 := CreateTestRole(testNamespace, "test-role1")
	role2 := CreateTestRole(testNamespace, "test-role2")
	_, err := clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), role1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	_, err = clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), role2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	testRoleRef := CreateTestRoleRef("test-role1")
	testRoleBinding := CreateTestRoleBinding(testNamespace, "test-rb", "test-sa", testRoleRef)
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), testRoleBinding, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	return clientset
}
func TestRetrieveUsedRoles(t *testing.T) {
	clientset := createTestRoles(t)

	usedRoles, err := retrieveUsedRoles(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedRoles) != 1 {
		t.Errorf("Expected 1 used role, got %d", len(usedRoles))
	}

	if usedRoles[0] != "test-role1" {
		t.Errorf("Expected 'test-role1', got %s", usedRoles[0])
	}
}

func TestRetrieveRoleNames(t *testing.T) {
	clientset := createTestRoles(t)
	allRoles, err := retrieveRoleNames(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(allRoles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(allRoles))
	}
}

func TestProcessNamespaceRoles(t *testing.T) {
	clientset := createTestRoles(t)
	
	usedRoles, err := processNamespaceRoles(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedRoles) != 1 {
		t.Errorf("Expected 1 unused role, got %d", len(usedRoles))
	}

	if usedRoles[0] != "test-role2" {
		t.Errorf("Expected 'test-role2', got %s", usedRoles[0])
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
