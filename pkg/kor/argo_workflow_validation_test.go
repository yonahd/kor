package kor

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestValidateArgoWorkflowTemplateAvailability(t *testing.T) {
	scheme := runtime.NewScheme()

	// Test case: WorkflowTemplate CRD doesn't exist (simulated by fake client)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	enabled := ValidateArgoWorkflowTemplateAvailability(dynamicClient)

	// Since we can't easily fake a CRD's existence in tests, we expect this to be false
	if enabled {
		t.Error("Expected ValidateArgoWorkflowTemplateAvailability to return false when CRD doesn't exist")
	}
}

func TestValidateResourceReferencesFromArgoWorkflowTemplates_NoCRD(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	refs, err := ValidateResourceReferencesFromArgoWorkflowTemplates(clientset, dynamicClient, "test-namespace")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// When CRD doesn't exist, should return empty references
	if len(refs.ConfigMaps) != 0 {
		t.Errorf("Expected 0 ConfigMaps, got %d", len(refs.ConfigMaps))
	}
	if len(refs.Secrets) != 0 {
		t.Errorf("Expected 0 Secrets, got %d", len(refs.Secrets))
	}
	if len(refs.PVCs) != 0 {
		t.Errorf("Expected 0 PVCs, got %d", len(refs.PVCs))
	}
}

func TestExtractResourceReferencesFromTypedWorkflowTemplate(t *testing.T) {
	// Create a typed WorkflowTemplate with various resource references
	wt := &wfv1.WorkflowTemplate{
		Spec: wfv1.WorkflowSpec{
			// Synchronization semaphore
			Synchronization: &wfv1.Synchronization{
				Semaphore: &wfv1.SemaphoreRef{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "workflow-semaphore-config",
						},
						Key: "workflow",
					},
				},
			},
			// Global volumes
			Volumes: []corev1.Volume{
				{
					Name: "global-secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "global-secret",
						},
					},
				},
				{
					Name: "global-config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "global-config",
							},
						},
					},
				},
			},
			// Templates
			Templates: []wfv1.Template{
				{
					Name: "test-template",
					Container: &corev1.Container{
						Name:  "test-container",
						Image: "busybox",
						Env: []corev1.EnvVar{
							{
								Name: "CONFIG_VALUE",
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "app-config",
										},
										Key: "config-key",
									},
								},
							},
							{
								Name: "SECRET_VALUE",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "app-secret",
										},
										Key: "secret-key",
									},
								},
							},
						},
						EnvFrom: []corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "env-config",
									},
								},
							},
							{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "env-secret",
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "secret-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "volume-secret",
								},
							},
						},
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "volume-config",
									},
								},
							},
						},
						{
							Name: "pvc-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "test-pvc",
								},
							},
						},
						{
							Name: "projected-volume",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "projected-config",
												},
											},
										},
										{
											Secret: &corev1.SecretProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "projected-secret",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	refs := &ArgoWorkflowTemplateResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}

	extractResourceReferencesFromTypedWorkflowTemplate(wt, refs)

	expectedConfigMaps := []string{
		"workflow-semaphore-config",
		"global-config",
		"app-config",
		"env-config",
		"volume-config",
		"projected-config",
	}

	expectedSecrets := []string{
		"global-secret",
		"app-secret",
		"env-secret",
		"volume-secret",
		"projected-secret",
	}

	expectedPVCs := []string{
		"test-pvc",
	}

	// Check ConfigMaps
	if len(refs.ConfigMaps) != len(expectedConfigMaps) {
		t.Errorf("Expected %d ConfigMaps, got %d: %v", len(expectedConfigMaps), len(refs.ConfigMaps), refs.ConfigMaps)
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
			t.Errorf("Expected ConfigMap %q not found in refs: %v", expected, refs.ConfigMaps)
		}
	}

	// Check Secrets
	if len(refs.Secrets) != len(expectedSecrets) {
		t.Errorf("Expected %d Secrets, got %d: %v", len(expectedSecrets), len(refs.Secrets), refs.Secrets)
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
			t.Errorf("Expected Secret %q not found in refs: %v", expected, refs.Secrets)
		}
	}

	// Check PVCs
	if len(refs.PVCs) != len(expectedPVCs) {
		t.Errorf("Expected %d PVCs, got %d: %v", len(expectedPVCs), len(refs.PVCs), refs.PVCs)
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
			t.Errorf("Expected PVC %q not found in refs: %v", expected, refs.PVCs)
		}
	}
}

func TestExtractResourceReferencesFromTypedWorkflowTemplate_Empty(t *testing.T) {
	// Test with empty WorkflowTemplate
	wt := &wfv1.WorkflowTemplate{
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{}, // Empty templates
		},
	}

	refs := &ArgoWorkflowTemplateResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}

	extractResourceReferencesFromTypedWorkflowTemplate(wt, refs)

	// Should be no references in an empty spec
	if len(refs.ConfigMaps) != 0 {
		t.Errorf("Expected 0 ConfigMaps, got %d", len(refs.ConfigMaps))
	}
	if len(refs.Secrets) != 0 {
		t.Errorf("Expected 0 Secrets, got %d", len(refs.Secrets))
	}
	if len(refs.PVCs) != 0 {
		t.Errorf("Expected 0 PVCs, got %d", len(refs.PVCs))
	}
}
