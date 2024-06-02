package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/filters"
)

func createTestClusterRoles(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	var AggregatedLabels = map[string]string{"rbac.authorization.k8s.io/aggregate-to-test-clusterRole1": "true"}
	var matchLabels = v1.LabelSelector{
		MatchLabels: AggregatedLabels,
	}
	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	clusterRole1 := CreateTestClusterRole("test-clusterRole1", AppLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "clusterRole", err)
	}

	clusterRole2 := CreateTestClusterRole("test-clusterRole2", AppLabels, matchLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "clusterRole", err)
	}

	clusterRole3 := CreateTestClusterRole("test-clusterRole3", AppLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	clusterRole4 := CreateTestClusterRole("test-clusterRole4", UsedLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "clusterRole", err)
	}

	clusterRole5 := CreateTestClusterRole("test-clusterRole5", UnusedLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole5, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	clusterRole6 := CreateTestClusterRole("test-clusterRole6", AggregatedLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole6, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	testRoleRef2 := CreateTestRoleRefForClusterRole("test-clusterRole2")
	testClusterRoleBinding := CreateTestClusterRoleBindingRoleRef(testNamespace, "test-rb2", "test-sa", testRoleRef2)
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), testClusterRoleBinding, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	testRoleRef3 := CreateTestRoleRefForClusterRole("test-clusterRole3")
	testRoleBinding := CreateTestRoleBinding(testNamespace, "test-rb", "test-sa", testRoleRef3)
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), testRoleBinding, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "clusterRole", err)
	}

	return clientset
}
func TestRetrieveUsedClusterRoles(t *testing.T) {
	clientset := createTestClusterRoles(t)

	usedClusterRoles, err := retrieveUsedClusterRoles(clientset, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(usedClusterRoles) != 3 {
		t.Errorf("Expected 3 used cluster role, got %d", len(usedClusterRoles))
	}

	expectedRoles := []string{
		"test-clusterRole2",
		"test-clusterRole3",
		"test-clusterRole6",
	}
	sort.Strings(usedClusterRoles)
	t.Log(usedClusterRoles)
	if !reflect.DeepEqual(usedClusterRoles, expectedRoles) {
		t.Errorf("Expected 'test-role3', 'test-role2', 'test-role6', got %s, %s, %s", usedClusterRoles[0], usedClusterRoles[1], usedClusterRoles[2])
	}
}

func TestRetrieveClusterRoleNames(t *testing.T) {
	clientset := createTestClusterRoles(t)
	allRoles, _, err := retrieveClusterRoleNames(clientset, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(allRoles) != 4 {
		t.Errorf("Expected 4 roles, got %d", len(allRoles))
	}
}

func TestProcessClusterRoles(t *testing.T) {
	clientset := createTestClusterRoles(t)

	unusedClusterRoles, err := processClusterRoles(clientset, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedClusterRoles) != 2 {
		t.Errorf("Expected 2 unused role, got %d", len(unusedClusterRoles))
	}

	if unusedClusterRoles[0].Name != "test-clusterRole1" && unusedClusterRoles[1].Name != "test-clusterRole5" {
		t.Errorf("Expected 'test-clusterRole1', 'test-clusterRole5', got %s, %s", unusedClusterRoles[0], unusedClusterRoles[1])
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
		GroupBy:       "namespace",
	}

	output, err := GetUnusedClusterRoles(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedRolesStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"": {
			"ClusterRole": {
				"test-clusterRole1",
				"test-clusterRole5",
			},
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
