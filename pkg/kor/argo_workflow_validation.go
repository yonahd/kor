package kor

import (
	"context"
	"sync"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	// Global state to track if Argo WorkflowTemplate CRD is available
	argoWorkflowTemplateEnabled bool
	argoWorkflowTemplateOnce    sync.Once
	workflowTemplateGVR         = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "workflowtemplates",
	}
)

// ArgoWorkflowTemplateResourceReferences holds references to resources used by WorkflowTemplates
type ArgoWorkflowTemplateResourceReferences struct {
	ConfigMaps []string
	Secrets    []string
	PVCs       []string
}

// ValidateArgoWorkflowTemplateAvailability checks once if the WorkflowTemplate CRD exists in the cluster
func ValidateArgoWorkflowTemplateAvailability(dynamicClient dynamic.Interface) bool {
	argoWorkflowTemplateOnce.Do(func() {
		// Try to list WorkflowTemplates to check if the CRD exists
		// Use limit=1 to minimize overhead - we just need to know if it exists
		_, err := dynamicClient.Resource(workflowTemplateGVR).Namespace("").List(context.TODO(), metav1.ListOptions{Limit: 1})
		argoWorkflowTemplateEnabled = (err == nil)
	})
	return argoWorkflowTemplateEnabled
}

// ValidateResourceReferencesFromArgoWorkflowTemplates scans WorkflowTemplates for resource references
func ValidateResourceReferencesFromArgoWorkflowTemplates(clientset kubernetes.Interface, dynamicClient dynamic.Interface, namespace string) (*ArgoWorkflowTemplateResourceReferences, error) {
	refs := &ArgoWorkflowTemplateResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}

	// If WorkflowTemplate CRD is not available, return empty references
	if !ValidateArgoWorkflowTemplateAvailability(dynamicClient) {
		return refs, nil
	}

	// List all WorkflowTemplates in the namespace
	workflowTemplateList, err := dynamicClient.Resource(workflowTemplateGVR).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Process each WorkflowTemplate
	for _, item := range workflowTemplateList.Items {
		// Convert unstructured to typed WorkflowTemplate
		var workflowTemplate wfv1.WorkflowTemplate
		if err := convertUnstructuredToWorkflowTemplate(&item, &workflowTemplate); err != nil {
			// Skip invalid WorkflowTemplates but continue processing others
			continue
		}

		extractResourceReferencesFromTypedWorkflowTemplate(&workflowTemplate, refs)
	}

	// Remove duplicates and sort
	refs.ConfigMaps = RemoveDuplicatesAndSort(refs.ConfigMaps)
	refs.Secrets = RemoveDuplicatesAndSort(refs.Secrets)
	refs.PVCs = RemoveDuplicatesAndSort(refs.PVCs)

	return refs, nil
}

// convertUnstructuredToWorkflowTemplate converts unstructured data to a typed WorkflowTemplate
func convertUnstructuredToWorkflowTemplate(unstructuredItem *unstructured.Unstructured, workflowTemplate *wfv1.WorkflowTemplate) error {
	// Convert unstructured to typed WorkflowTemplate using runtime converter
	return runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredItem.Object, workflowTemplate)
}

// extractResourceReferencesFromTypedWorkflowTemplate extracts resource references from a typed WorkflowTemplate
func extractResourceReferencesFromTypedWorkflowTemplate(wt *wfv1.WorkflowTemplate, refs *ArgoWorkflowTemplateResourceReferences) {
	// Extract from synchronization semaphore
	if wt.Spec.Synchronization != nil && wt.Spec.Synchronization.Semaphore != nil {
		if wt.Spec.Synchronization.Semaphore.ConfigMapKeyRef != nil {
			refs.ConfigMaps = append(refs.ConfigMaps, wt.Spec.Synchronization.Semaphore.ConfigMapKeyRef.Name)
		}
	}

	// Extract from templates
	for _, template := range wt.Spec.Templates {
		extractResourceReferencesFromTemplate(&template, refs)
	}

	// Extract from global volumes
	extractResourceReferencesFromVolumes(wt.Spec.Volumes, refs)
}

// extractResourceReferencesFromTemplate extracts resource references from a single template
func extractResourceReferencesFromTemplate(template *wfv1.Template, refs *ArgoWorkflowTemplateResourceReferences) {
	// Extract from container environment variables
	if template.Container != nil {
		extractResourceReferencesFromContainer(template.Container, refs)
	}

	// Extract from script environment variables
	if template.Script != nil {
		extractResourceReferencesFromContainer(&template.Script.Container, refs)
	}

	// Extract from volumes
	extractResourceReferencesFromVolumes(template.Volumes, refs)
}

// extractResourceReferencesFromContainer extracts resource references from container environment variables
func extractResourceReferencesFromContainer(container *corev1.Container, refs *ArgoWorkflowTemplateResourceReferences) {
	// Extract from environment variables
	for _, env := range container.Env {
		if env.ValueFrom != nil {
			if env.ValueFrom.ConfigMapKeyRef != nil {
				refs.ConfigMaps = append(refs.ConfigMaps, env.ValueFrom.ConfigMapKeyRef.Name)
			}
			if env.ValueFrom.SecretKeyRef != nil {
				refs.Secrets = append(refs.Secrets, env.ValueFrom.SecretKeyRef.Name)
			}
		}
	}

	// Extract from envFrom
	for _, envFrom := range container.EnvFrom {
		if envFrom.ConfigMapRef != nil {
			refs.ConfigMaps = append(refs.ConfigMaps, envFrom.ConfigMapRef.Name)
		}
		if envFrom.SecretRef != nil {
			refs.Secrets = append(refs.Secrets, envFrom.SecretRef.Name)
		}
	}
}

// extractResourceReferencesFromVolumes extracts resource references from volumes
func extractResourceReferencesFromVolumes(volumes []corev1.Volume, refs *ArgoWorkflowTemplateResourceReferences) {
	for _, volume := range volumes {
		// ConfigMap volumes
		if volume.ConfigMap != nil {
			refs.ConfigMaps = append(refs.ConfigMaps, volume.ConfigMap.Name)
		}

		// Secret volumes
		if volume.Secret != nil {
			refs.Secrets = append(refs.Secrets, volume.Secret.SecretName)
		}

		// PVC volumes
		if volume.PersistentVolumeClaim != nil {
			refs.PVCs = append(refs.PVCs, volume.PersistentVolumeClaim.ClaimName)
		}

		// Projected volumes
		if volume.Projected != nil {
			for _, source := range volume.Projected.Sources {
				if source.ConfigMap != nil {
					refs.ConfigMaps = append(refs.ConfigMaps, source.ConfigMap.Name)
				}
				if source.Secret != nil {
					refs.Secrets = append(refs.Secrets, source.Secret.Name)
				}
			}
		}
	}
}
