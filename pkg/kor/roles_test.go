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

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestRoles(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	role1 := CreateTestRole(testNamespace, "test-role1", AppLabels)
	_, err = clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), role1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	role2 := CreateTestRole(testNamespace, "test-role2", AppLabels)
	_, err = clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), role2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	role3 := CreateTestRole(testNamespace, "test-role3", UsedLabels)
	_, err = clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), role3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	role4 := CreateTestRole(testNamespace, "test-role4", UnusedLabels)
	_, err = clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), role4, v1.CreateOptions{})
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
	allRoles, _, err := retrieveRoleNames(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(allRoles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(allRoles))
	}
}

func TestProcessNamespaceRoles(t *testing.T) {
	clientset := createTestRoles(t)

	unusedRoles, err := processNamespaceRoles(clientset, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedRoles) != 2 {
		t.Errorf("Expected 2 unused roles, got %d", len(unusedRoles))
	}

	if unusedRoles[0].Name != "test-role2" || unusedRoles[1].Name != "test-role4" {
		t.Errorf("Expected 'test-role2', 'test-role4', got %s %s", unusedRoles[0], unusedRoles[1])
	}
}

func TestGetUnusedRolesStructured(t *testing.T) {
	clientset := createTestRoles(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedRoles(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedRolesStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Role": {
				"test-role2",
				"test-role4",
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

func TestFilterOwnerReferencedRoles(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// Create two roles - one owned by another resource, one standalone
	// Role owned by another resource
	ownedRole := CreateTestRole(testNamespace, "owned-role", AppLabels)
	// Add owner reference to another resource
	ownedRole.OwnerReferences = []v1.OwnerReference{
		{
			Kind: "Application",
			Name: "test-application",
		},
	}

	// Standalone Role
	standaloneRole := CreateTestRole(testNamespace, "standalone-role", AppLabels)

	_, err = clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), ownedRole, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Role: %v", err)
	}

	_, err = clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), standaloneRole, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake Role: %v", err)
	}

	// Test without filter - should return both
	filterOptsNoSkip := &filters.Options{IgnoreOwnerReferences: false}
	unusedWithoutFilter, err := processNamespaceRoles(clientset, testNamespace, filterOptsNoSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused Roles: %v", err)
	}

	if len(unusedWithoutFilter) != 2 {
		t.Errorf("Expected 2 unused Role objects without filter, got %d", len(unusedWithoutFilter))
	}

	// Test with filter - should return only standalone
	filterOptsWithSkip := &filters.Options{IgnoreOwnerReferences: true}
	unusedWithFilter, err := processNamespaceRoles(clientset, testNamespace, filterOptsWithSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused Roles: %v", err)
	}

	if len(unusedWithFilter) != 1 {
		t.Errorf("Expected 1 unused Role object with filter, got %d", len(unusedWithFilter))
	}

	if unusedWithFilter[0].Name != "standalone-role" {
		t.Errorf("Expected standalone-role to be unused, got %s", unusedWithFilter[0].Name)
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
