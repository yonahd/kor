package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestMultiResources(t *testing.T) (kubernetes.Interface, clusterconfig.ClientInterface) {
	clientsetinterface, _ := NewFakeClientSet(t)
	clientset := clientsetinterface.GetKubernetesClient()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	deployment1 := CreateTestDeployment(testNamespace, "test-deployment1", 0, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	configmap1 := CreateTestConfigmap(testNamespace, "configmap-1", AppLabels)
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	return clientset, clientsetinterface

}

func TestRetrieveNamespaceDiff(t *testing.T) {
	clientset, _ := createTestMultiResources(t)
	resourceList := []string{"cm", "pdb", "deployment"}
	filterOpts := &filters.Options{}

	namespaceDiff := retrieveNamespaceDiffs(clientset, testNamespace, resourceList, filterOpts)

	if len(namespaceDiff) != 3 {
		t.Fatalf("Expected 3 diffs, got %d", len(namespaceDiff))
	}

	if namespaceDiff[0].resourceType != "ConfigMap" && namespaceDiff[0].diff[0].Name != "configmap-1" {
		t.Fatalf("Expected configmap-1, got %s", namespaceDiff[0].diff[0].Name)
	}

	if namespaceDiff[1].resourceType != "Pdb" && namespaceDiff[1].diff != nil {
		t.Fatalf("Expected nil, got %s", namespaceDiff[1].diff[0].Name)
	}

	if namespaceDiff[2].resourceType != "Deployment" && namespaceDiff[2].diff[0].Name != "test-deployment1" {
		t.Fatalf("Expected test-deployment1, got %s", namespaceDiff[2].diff[0].Name)
	}

}

func TestGetUnusedMulti(t *testing.T) {
	clientset, clientsetinterface := createTestMultiResources(t)
	resourceList := "cm,pdb,deployment"

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedMulti(resourceList, &filters.Options{}, clientset, nil, nil, clientsetinterface, "json", opts)

	if err != nil {
		t.Fatalf("Error calling GetUnusedMulti: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"test-namespace": {
			"ConfigMap": {
				"configmap-1",
			},
			"Deployment": {
				"test-deployment1",
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
