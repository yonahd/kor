package kor

import (
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestDeleteResource(t *testing.T) {
	clientset := kubernetes.NewForConfigOrDie(&rest.Config{
		Host: "https://localhost:6443",
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})

	tests := []struct {
		name          string
		diff          []string
		resourceType  string
		expectedDiff  []string
		expectedError bool
	}{
		{
			name:          "Test deletion confirmation",
			diff:          []string{"resource1", "resource2"},
			resourceType:  "ConfigMap",
			expectedDiff:  []string{"resource1-DELETED", "resource2"},
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deletedDiff := DeleteResource(test.diff, clientset, "namespace", test.resourceType)

			for i, deleted := range deletedDiff {
				if deleted != test.expectedDiff[i] {
					t.Errorf("Expected: %s, Got: %s", test.expectedDiff[i], deleted)
				}
			}
		})
	}
}
