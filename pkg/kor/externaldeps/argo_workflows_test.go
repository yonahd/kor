package externaldeps

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestWorkflowTemplateScanner_GetName(t *testing.T) {
	scanner := NewWorkflowTemplateScanner()
	expectedName := "Argo Workflows WorkflowTemplate"
	if scanner.GetName() != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, scanner.GetName())
	}
}

func TestWorkflowTemplateScanner_GetSupportedResources(t *testing.T) {
	scanner := NewWorkflowTemplateScanner()
	supportedResources := scanner.GetSupportedResources()
	expected := []string{"ConfigMap", "Secret", "PVC"}
	
	if len(supportedResources) != len(expected) {
		t.Errorf("expected %d supported resources, got %d", len(expected), len(supportedResources))
		return
	}
	
	for i, resource := range expected {
		if supportedResources[i] != resource {
			t.Errorf("expected resource %q at index %d, got %q", resource, i, supportedResources[i])
		}
	}
}

func TestWorkflowTemplateScanner_extractResourceReferences(t *testing.T) {
	scanner := NewWorkflowTemplateScanner()
	
	// Create a mock WorkflowTemplate with various resource references
	wt := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "WorkflowTemplate",
			"metadata": map[string]interface{}{
				"name":      "test-workflow-template",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"synchronization": map[string]interface{}{
					"semaphore": map[string]interface{}{
						"configMapKeyRef": map[string]interface{}{
							"name": "workflow-semaphore-config",
							"key":  "workflow",
						},
					},
				},
				"templates": []interface{}{
					map[string]interface{}{
						"name": "test-template",
						"script": map[string]interface{}{
							"env": []interface{}{
								map[string]interface{}{
									"name": "CONFIG_VALUE",
									"valueFrom": map[string]interface{}{
										"configMapKeyRef": map[string]interface{}{
											"name": "app-config",
											"key":  "config-key",
										},
									},
								},
								map[string]interface{}{
									"name": "SECRET_VALUE",
									"valueFrom": map[string]interface{}{
										"secretKeyRef": map[string]interface{}{
											"name": "app-secret",
											"key":  "secret-key",
										},
									},
								},
							},
						},
						"volumes": []interface{}{
							map[string]interface{}{
								"name": "secret-volume",
								"secret": map[string]interface{}{
									"secretName": "volume-secret",
								},
							},
							map[string]interface{}{
								"name": "config-volume",
								"configMap": map[string]interface{}{
									"name": "volume-config",
								},
							},
							map[string]interface{}{
								"name": "pvc-volume",
								"persistentVolumeClaim": map[string]interface{}{
									"claimName": "test-pvc",
								},
							},
							map[string]interface{}{
								"name": "projected-volume",
								"projected": map[string]interface{}{
									"sources": []interface{}{
										map[string]interface{}{
											"configMap": map[string]interface{}{
												"name": "projected-config",
											},
										},
										map[string]interface{}{
											"secret": map[string]interface{}{
												"name": "projected-secret",
											},
										},
									},
								},
							},
						},
					},
				},
				"volumes": []interface{}{
					map[string]interface{}{
						"name": "global-secret-volume",
						"secret": map[string]interface{}{
							"secretName": "global-secret",
						},
					},
				},
			},
		},
	}
	
	refs := &ResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}
	
	scanner.extractResourceReferences(wt, refs)
	
	expectedConfigMaps := []string{
		"workflow-semaphore-config",
		"app-config",
		"volume-config",
		"projected-config",
	}
	
	expectedSecrets := []string{
		"app-secret",
		"volume-secret",
		"projected-secret",
		"global-secret",
	}
	
	expectedPVCs := []string{
		"test-pvc",
	}
	
	// Check ConfigMaps
	if len(refs.ConfigMaps) != len(expectedConfigMaps) {
		t.Errorf("expected %d ConfigMaps, got %d: %v", len(expectedConfigMaps), len(refs.ConfigMaps), refs.ConfigMaps)
	}
	for _, expected := range expectedConfigMaps {
		found := false
		for _, actual := range refs.ConfigMaps {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected ConfigMap %q not found in refs: %v", expected, refs.ConfigMaps)
		}
	}
	
	// Check Secrets
	if len(refs.Secrets) != len(expectedSecrets) {
		t.Errorf("expected %d Secrets, got %d: %v", len(expectedSecrets), len(refs.Secrets), refs.Secrets)
	}
	for _, expected := range expectedSecrets {
		found := false
		for _, actual := range refs.Secrets {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected Secret %q not found in refs: %v", expected, refs.Secrets)
		}
	}
	
	// Check PVCs
	if len(refs.PVCs) != len(expectedPVCs) {
		t.Errorf("expected %d PVCs, got %d: %v", len(expectedPVCs), len(refs.PVCs), refs.PVCs)
	}
	for _, expected := range expectedPVCs {
		found := false
		for _, actual := range refs.PVCs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected PVC %q not found in refs: %v", expected, refs.PVCs)
		}
	}
}

func TestWorkflowTemplateScanner_extractResourceReferences_EmptySpec(t *testing.T) {
	scanner := NewWorkflowTemplateScanner()
	
	// Test with empty WorkflowTemplate
	wt := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "WorkflowTemplate",
			"metadata": map[string]interface{}{
				"name":      "empty-workflow-template",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{},
		},
	}
	
	refs := &ResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}
	
	scanner.extractResourceReferences(wt, refs)
	
	// Should be no references in an empty spec
	if len(refs.ConfigMaps) != 0 {
		t.Errorf("expected 0 ConfigMaps, got %d", len(refs.ConfigMaps))
	}
	if len(refs.Secrets) != 0 {
		t.Errorf("expected 0 Secrets, got %d", len(refs.Secrets))
	}
	if len(refs.PVCs) != 0 {
		t.Errorf("expected 0 PVCs, got %d", len(refs.PVCs))
	}
}

func TestWorkflowTemplateScanner_extractResourceReferences_MissingSpec(t *testing.T) {
	scanner := NewWorkflowTemplateScanner()
	
	// Test with WorkflowTemplate missing spec
	wt := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "WorkflowTemplate",
			"metadata": map[string]interface{}{
				"name":      "no-spec-workflow-template",
				"namespace": "test-namespace",
			},
		},
	}
	
	refs := &ResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}
	
	// Should not panic and should return no references
	scanner.extractResourceReferences(wt, refs)
	
	if len(refs.ConfigMaps) != 0 {
		t.Errorf("expected 0 ConfigMaps, got %d", len(refs.ConfigMaps))
	}
	if len(refs.Secrets) != 0 {
		t.Errorf("expected 0 Secrets, got %d", len(refs.Secrets))
	}
	if len(refs.PVCs) != 0 {
		t.Errorf("expected 0 PVCs, got %d", len(refs.PVCs))
	}
}
