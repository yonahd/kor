package externaldeps

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	argoWorkflowsAPIVersion = "argoproj.io/v1alpha1"
	workflowTemplateKind    = "WorkflowTemplate"
)

// WorkflowTemplateScanner scans Argo Workflows WorkflowTemplate CRDs
// for references to ConfigMaps, Secrets, and PVCs
type WorkflowTemplateScanner struct {
	gvr schema.GroupVersionResource
}

// NewWorkflowTemplateScanner creates a new WorkflowTemplate scanner
func NewWorkflowTemplateScanner() *WorkflowTemplateScanner {
	return &WorkflowTemplateScanner{
		gvr: schema.GroupVersionResource{
			Group:    "argoproj.io",
			Version:  "v1alpha1",
			Resource: "workflowtemplates",
		},
	}
}

// GetName returns the name of this scanner
func (s *WorkflowTemplateScanner) GetName() string {
	return "Argo Workflows WorkflowTemplate"
}

// IsEnabled checks if the WorkflowTemplate CRD is available in the cluster
func (s *WorkflowTemplateScanner) IsEnabled(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface) (bool, error) {
	// Try to list WorkflowTemplates to check if the CRD exists
	// We use a limit of 1 to avoid loading all resources, just to check availability
	_, err := dynamicClient.Resource(s.gvr).Namespace("").List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		// If we get an error, it likely means the CRD doesn't exist or we don't have permissions
		return false, nil
	}
	return true, nil
}

// ScanNamespace scans WorkflowTemplates in a specific namespace for resource references
func (s *WorkflowTemplateScanner) ScanNamespace(ctx context.Context, namespace string, clientset kubernetes.Interface, dynamicClient dynamic.Interface) (*ResourceReferences, error) {
	refs := &ResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}

	// List all WorkflowTemplates in the namespace
	workflowTemplates, err := dynamicClient.Resource(s.gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list WorkflowTemplates in namespace %s: %v", namespace, err)
	}

	// Scan each WorkflowTemplate for resource references
	for _, wt := range workflowTemplates.Items {
		s.extractResourceReferences(&wt, refs)
	}

	return refs, nil
}

// GetSupportedResources returns the resource types this scanner can find references to
func (s *WorkflowTemplateScanner) GetSupportedResources() []string {
	return []string{"ConfigMap", "Secret", "PVC"}
}

// extractResourceReferences extracts resource references from a WorkflowTemplate
func (s *WorkflowTemplateScanner) extractResourceReferences(wt *unstructured.Unstructured, refs *ResourceReferences) {
	spec, found, err := unstructured.NestedMap(wt.Object, "spec")
	if !found || err != nil {
		return
	}

	// Extract global level references
	s.extractFromSpec(spec, refs)

	// Extract references from templates
	templates, found, err := unstructured.NestedSlice(spec, "templates")
	if found && err == nil {
		for _, template := range templates {
			if templateMap, ok := template.(map[string]interface{}); ok {
				s.extractFromTemplate(templateMap, refs)
			}
		}
	}

	// Extract references from volumes at the global level
	volumes, found, err := unstructured.NestedSlice(spec, "volumes")
	if found && err == nil {
		s.extractFromVolumes(volumes, refs)
	}
}

// extractFromSpec extracts resource references from the WorkflowTemplate spec
func (s *WorkflowTemplateScanner) extractFromSpec(spec map[string]interface{}, refs *ResourceReferences) {
	// Extract from synchronization.semaphore.configMapKeyRef
	if syncMap, found := spec["synchronization"]; found {
		if sync, ok := syncMap.(map[string]interface{}); ok {
			if semaphoreMap, found := sync["semaphore"]; found {
				if semaphore, ok := semaphoreMap.(map[string]interface{}); ok {
					if configMapKeyRef, found := semaphore["configMapKeyRef"]; found {
						if cmRef, ok := configMapKeyRef.(map[string]interface{}); ok {
							if name, found := cmRef["name"]; found {
								if nameStr, ok := name.(string); ok {
									refs.ConfigMaps = append(refs.ConfigMaps, nameStr)
								}
							}
						}
					}
				}
			}
		}
	}
}

// extractFromTemplate extracts resource references from a single template
func (s *WorkflowTemplateScanner) extractFromTemplate(template map[string]interface{}, refs *ResourceReferences) {
	// Extract from script.env
	if script, found := template["script"]; found {
		if scriptMap, ok := script.(map[string]interface{}); ok {
			s.extractFromEnv(scriptMap, refs)
		}
	}

	// Extract from container.env
	if container, found := template["container"]; found {
		if containerMap, ok := container.(map[string]interface{}); ok {
			s.extractFromEnv(containerMap, refs)
		}
	}

	// Extract from volumes
	if volumes, found := template["volumes"]; found {
		if volumeSlice, ok := volumes.([]interface{}); ok {
			s.extractFromVolumes(volumeSlice, refs)
		}
	}
}

// extractFromEnv extracts resource references from environment variables
func (s *WorkflowTemplateScanner) extractFromEnv(container map[string]interface{}, refs *ResourceReferences) {
	if env, found := container["env"]; found {
		if envSlice, ok := env.([]interface{}); ok {
			for _, envVar := range envSlice {
				if envVarMap, ok := envVar.(map[string]interface{}); ok {
					if valueFrom, found := envVarMap["valueFrom"]; found {
						if valueFromMap, ok := valueFrom.(map[string]interface{}); ok {
							// Check for configMapKeyRef
							if configMapKeyRef, found := valueFromMap["configMapKeyRef"]; found {
								if cmRef, ok := configMapKeyRef.(map[string]interface{}); ok {
									if name, found := cmRef["name"]; found {
										if nameStr, ok := name.(string); ok {
											refs.ConfigMaps = append(refs.ConfigMaps, nameStr)
										}
									}
								}
							}
							// Check for secretKeyRef
							if secretKeyRef, found := valueFromMap["secretKeyRef"]; found {
								if secretRef, ok := secretKeyRef.(map[string]interface{}); ok {
									if name, found := secretRef["name"]; found {
										if nameStr, ok := name.(string); ok {
											refs.Secrets = append(refs.Secrets, nameStr)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// extractFromVolumes extracts resource references from volumes
func (s *WorkflowTemplateScanner) extractFromVolumes(volumes []interface{}, refs *ResourceReferences) {
	for _, volume := range volumes {
		if volumeMap, ok := volume.(map[string]interface{}); ok {
			// Check for secret volumes
			if secret, found := volumeMap["secret"]; found {
				if secretMap, ok := secret.(map[string]interface{}); ok {
					if secretName, found := secretMap["secretName"]; found {
						if nameStr, ok := secretName.(string); ok {
							refs.Secrets = append(refs.Secrets, nameStr)
						}
					}
				}
			}

			// Check for configMap volumes
			if configMap, found := volumeMap["configMap"]; found {
				if cmMap, ok := configMap.(map[string]interface{}); ok {
					if name, found := cmMap["name"]; found {
						if nameStr, ok := name.(string); ok {
							refs.ConfigMaps = append(refs.ConfigMaps, nameStr)
						}
					}
				}
			}

			// Check for PVC volumes
			if pvc, found := volumeMap["persistentVolumeClaim"]; found {
				if pvcMap, ok := pvc.(map[string]interface{}); ok {
					if claimName, found := pvcMap["claimName"]; found {
						if nameStr, ok := claimName.(string); ok {
							refs.PVCs = append(refs.PVCs, nameStr)
						}
					}
				}
			}

			// Check for projected volumes (can contain configMaps and secrets)
			if projected, found := volumeMap["projected"]; found {
				if projectedMap, ok := projected.(map[string]interface{}); ok {
					if sources, found := projectedMap["sources"]; found {
						if sourcesSlice, ok := sources.([]interface{}); ok {
							for _, source := range sourcesSlice {
								if sourceMap, ok := source.(map[string]interface{}); ok {
									// ConfigMap in projected volume
									if configMap, found := sourceMap["configMap"]; found {
										if cmMap, ok := configMap.(map[string]interface{}); ok {
											if name, found := cmMap["name"]; found {
												if nameStr, ok := name.(string); ok {
													refs.ConfigMaps = append(refs.ConfigMaps, nameStr)
												}
											}
										}
									}
									// Secret in projected volume
									if secret, found := sourceMap["secret"]; found {
										if secretMap, ok := secret.(map[string]interface{}); ok {
											if name, found := secretMap["name"]; found {
												if nameStr, ok := name.(string); ok {
													refs.Secrets = append(refs.Secrets, nameStr)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
