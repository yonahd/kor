package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestVolumeAttachments(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	// Create a valid node
	_, err := clientset.CoreV1().Nodes().Create(context.TODO(), CreateTestNode("node-1"), v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating node: %v", err)
	}

	// Create a valid PV
	_, err = clientset.CoreV1().PersistentVolumes().Create(context.TODO(), CreateTestPv("pv-1", "", map[string]string{}, ""), v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating PV: %v", err)
	}

	// Create a valid CSIDriver
	_, err = clientset.StorageV1().CSIDrivers().Create(context.TODO(), CreateTestCSIDriver("csi-driver-1"), v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating CSIDriver: %v", err)
	}

	// Create VolumeAttachments
	_, err = clientset.StorageV1().VolumeAttachments().Create(context.TODO(), CreateTestVolumeAttachment("va-1", "csi-driver-1", "node-1", "pv-1"), v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating VolumeAttachment %s: %v", "va-1", err)
	}

	_, err = clientset.StorageV1().VolumeAttachments().Create(context.TODO(), CreateTestVolumeAttachment("va-2", "csi-driver-1", "node-1", "pv-unknown"), v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating VolumeAttachment %s: %v", "va-2", err)
	}

	_, err = clientset.StorageV1().VolumeAttachments().Create(context.TODO(), CreateTestVolumeAttachment("va-3", "csi-driver-1", "node-unknown", "pv-1"), v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating VolumeAttachment %s: %v", "va-3", err)
	}

	_, err = clientset.StorageV1().VolumeAttachments().Create(context.TODO(), CreateTestVolumeAttachment("va-4", "csi-driver-unknown", "node-1", "pv-1"), v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating VolumeAttachment %s: %v", "va-4", err)
	}

	return clientset
}

func TestGetUnusedVolumeAttachments(t *testing.T) {
	clientset := createTestVolumeAttachments(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedVolumeAttachments(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedVolumeAttachments: %v", err)
	}

	// Expected unused resources:
	// - va-2: No PV
	// - va-3: No Node
	// - va-4: No Driver
	expectedUnused := []string{"va-2", "va-3", "va-4"}
	expectedOutput := map[string]map[string][]string{
		"": {
			"VolumeAttachment": expectedUnused,
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	// Sort before comparing since order is not guaranteed
	sort.Strings(expectedOutput[""]["VolumeAttachment"])
	sort.Strings(actualOutput[""]["VolumeAttachment"])

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output %+v, but got %+v", expectedOutput, actualOutput)
	}
}
