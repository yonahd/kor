package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func createTestPvs(clientset *fake.Clientset, t *testing.T) *fake.Clientset {

	pv1 := CreateTestPv("test-pv1", "Bound")
	pv2 := CreateTestPv("test-pv2", "Available")
	_, err := clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "PV", err)
	}

	_, err = clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "PV", err)
	}

	return clientset
}

func createTestPvsClient(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	createTestPvs(clientset, t)

	return clientset
}

func TestProcessPvs(t *testing.T) {
	clientset := createTestPvsClient(t)
	usedPvs, err := processPvs(clientset, &FilterOptions{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedPvs) != 1 {
		t.Errorf("Expected 1 used pv, got %d", len(usedPvs))
	}

	if usedPvs[0] != "test-pv2" {
		t.Errorf("Expected 'test-pv2', got %s", usedPvs[0])
	}
}

func TestGetUnusedPvs(t *testing.T) {
	clientset := createTestPvsClient(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
	}

	output, err := GetUnusedPvs(&FilterOptions{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedPvs: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"": {
			"Pv": {"test-pv2"},
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
