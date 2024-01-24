package kor

/*
import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func createTestAllClient(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}, metav1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	createTestConfigmaps(clientset, t)
	createTestDeployments(clientset, t)
	createTestHpas(clientset, t)
	createTestIngresses(clientset, t)
	createTestJobs(clientset, t)
	createTestPdbs(clientset, t)
	createTestPods(clientset, t)
	createTestPvs(clientset, t)
	createTestPvcs(clientset, t)
	createTestReplicaSets(clientset, t)
	createTestRoles(clientset, t)
	createTestSecrets(clientset, t)
	createTestServiceAccounts(clientset, t)
	createTestServices(clientset, t)
	createTestStatefulSets(clientset, t)

	return clientset
}

func TestGetUnusedAll(t *testing.T) {
	clientset := createTestAllClient(t)

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

	output, err := GetUnusedAll(includeExcludeLists, &FilterOptions{}, clientset, "", "", "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedAll: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ConfigMap": {"configmap-3"},
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

*/
