package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/filters"
)

func createTestPvs(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	pv1 := CreateTestPv("test-pv1", "Bound", AppLabels, "test-sc1")
	_, err := clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "PV", err)
	}

	pv2 := CreateTestPv("test-pv2", "Available", AppLabels, "test-sc1")
	_, err = clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "PV", err)
	}

	pv3 := CreateTestPv("test-pv3", "Bound", UsedLabels, "test-sc1")
	_, err = clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "PV", err)
	}

	pv4 := CreateTestPv("test-pv4", "Available", UnusedLabels, "test-sc1")
	_, err = clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "PV", err)
	}

	return clientset
}

func TestProcessPvs(t *testing.T) {
	clientset := createTestPvs(t)
	usedPvs, err := processPvs(clientset, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(usedPvs) != 2 {
		t.Errorf("Expected 2 used pv, got %d", len(usedPvs))
	}

	if usedPvs[0].Name != "test-pv2" && usedPvs[1].Name != "test-pv3" {
		t.Errorf("Expected 'test-pv2', got %s", usedPvs[0])
	}
}

func TestGetUnusedPvs(t *testing.T) {
	clientset := createTestPvs(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedPvs(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedPvs: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"": {
			"Pv": {
				"test-pv2",
				"test-pv4",
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
