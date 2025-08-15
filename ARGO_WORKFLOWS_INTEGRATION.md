# Argo Workflows Integration

This document describes the integration of Argo Workflows WorkflowTemplate CRD support into Kor, which prevents false positives when detecting unused ConfigMaps, Secrets, and PVCs that are referenced by WorkflowTemplates.

## Overview

Argo Workflows uses WorkflowTemplate CRDs to define reusable workflow specifications. These templates often reference Kubernetes resources like ConfigMaps, Secrets, and PersistentVolumeClaims (PVCs). Without understanding these references, Kor might incorrectly mark these resources as unused, leading to false positives.

This integration adds intelligent scanning of WorkflowTemplate CRDs to identify resource dependencies and prevent false positives.

## Features

### ✅ Supported Resource References

The integration detects references to the following resources in WorkflowTemplates:

#### ConfigMaps
- Global synchronization semaphores: `spec.synchronization.semaphore.configMapKeyRef`
- Environment variables: `env[].valueFrom.configMapKeyRef`  
- Volume mounts: `volumes[].configMap`
- Projected volumes: `volumes[].projected.sources[].configMap`

#### Secrets
- Environment variables: `env[].valueFrom.secretKeyRef`
- Volume mounts: `volumes[].secret`
- Projected volumes: `volumes[].projected.sources[].secret`

#### PersistentVolumeClaims
- Volume mounts: `volumes[].persistentVolumeClaim`

### 🔄 Automatic Detection

- The integration automatically detects if the WorkflowTemplate CRD is available in the cluster
- Only activates when Argo Workflows is installed and WorkflowTemplate CRD exists
- Gracefully handles clusters without Argo Workflows (no impact on performance)

### 🔧 Modular Architecture

- Built with a pluggable scanner architecture
- Easy to extend for other Argo Workflows CRDs (ClusterWorkflowTemplate, etc.)
- Can be extended for other workflow engines (Tekton, etc.)

## How It Works

### Architecture

```
┌─────────────────────────┐
│   Kor Main Scanner      │
│   (ConfigMap/Secret/    │
│    PVC Detection)       │
└─────────┬───────────────┘
          │
          v
┌─────────────────────────┐
│  External Dependencies  │
│       Registry          │
└─────────┬───────────────┘
          │
          v
┌─────────────────────────┐
│  WorkflowTemplate       │
│       Scanner           │
└─────────────────────────┘
```

### Integration Points

The integration hooks into the existing resource scanning logic at these points:

1. **ConfigMap Scanning** (`pkg/kor/configmaps.go`)
   - `retrieveUsedCMFromExternalCRDs()` function
   - Integrated into `processNamespaceCM()`

2. **Secret Scanning** (`pkg/kor/secrets.go`)
   - `retrieveUsedSecretsFromExternalCRDs()` function  
   - Integrated into `processNamespaceSecret()`

3. **PVC Scanning** (`pkg/kor/pvc.go`)
   - `retrieveUsedPvcsFromExternalCRDs()` function
   - Integrated into `processNamespacePvcs()`

## Usage

The integration works transparently - no changes to existing Kor commands are needed.

### Examples

#### Before Integration (False Positives)
```bash
$ kor configmap --include-namespaces production
Unused ConfigMaps in namespace "production":
- workflow-config     # Actually used by WorkflowTemplate!
- app-settings        # Actually used by WorkflowTemplate!
```

#### After Integration (Accurate Results)
```bash
$ kor configmap --include-namespaces production
No unused ConfigMaps found in namespace "production"
```

#### Test WorkflowTemplate Detection
```bash
$ kor all --include-namespaces argo-workflows-namespace
# Will automatically detect and scan WorkflowTemplates if CRD exists
```

## Supported WorkflowTemplate Patterns

### Synchronization Semaphores
```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
spec:
  synchronization:
    semaphore:
      configMapKeyRef:
        name: workflow-semaphore-config  # ✅ Detected
        key: workflow
```

### Environment Variables
```yaml
spec:
  templates:
  - name: my-template
    script:
      env:
      - name: CONFIG_VALUE
        valueFrom:
          configMapKeyRef:
            name: app-config  # ✅ Detected
            key: config-key
      - name: SECRET_VALUE
        valueFrom:
          secretKeyRef:
            name: app-secret  # ✅ Detected
            key: secret-key
```

### Volume Mounts
```yaml
spec:
  templates:
  - name: my-template
    volumes:
    - name: config-volume
      configMap:
        name: volume-config  # ✅ Detected
    - name: secret-volume
      secret:
        secretName: volume-secret  # ✅ Detected
    - name: data-volume
      persistentVolumeClaim:
        claimName: my-pvc  # ✅ Detected
```

### Projected Volumes
```yaml
spec:
  volumes:
  - name: projected-volume
    projected:
      sources:
      - configMap:
          name: projected-config  # ✅ Detected
      - secret:
          name: projected-secret  # ✅ Detected
```

## Technical Implementation

### Key Components

#### 1. External Dependencies Framework (`pkg/kor/externaldeps/`)

**Interface** (`interface.go`)
- `ExternalResourceScanner` interface for pluggable scanners
- `ResourceReferences` struct for holding resource references
- `ScannerRegistry` for managing multiple scanners

**WorkflowTemplate Scanner** (`argo_workflows.go`)
- Implements `ExternalResourceScanner` interface
- Scans WorkflowTemplate CRDs for resource references
- Handles various referencing patterns

**Registry** (`registry.go`)
- Global singleton registry for scanners
- Automatic registration of default scanners

#### 2. Integration with Existing Scanners

Each resource type's scanner has been extended with:
- `retrieveUsed<ResourceType>FromExternalCRDs()` function
- Integration into the main processing function
- Proper deduplication and sorting of results

### Error Handling

- Graceful handling when WorkflowTemplate CRD doesn't exist
- Non-intrusive - if external scanning fails, traditional scanning continues
- Comprehensive error logging for debugging

### Performance Considerations

- CRD existence check is cached by the scanner registry
- Only active when WorkflowTemplate CRD is present
- Minimal overhead on clusters without Argo Workflows
- Efficient unstructured JSON parsing for CRD content

## Testing

Comprehensive test coverage includes:

- **Unit Tests** (`*_test.go`)
  - Scanner interface implementations
  - Resource reference extraction logic
  - Registry functionality
  - Error handling scenarios

- **Integration Tests**
  - End-to-end resource scanning with mock WorkflowTemplates
  - CRD detection and activation logic
  - False positive prevention verification

### Running Tests

```bash
# Run all external dependencies tests
go test ./pkg/kor/externaldeps/... -v

# Run full test suite
go test ./... -v
```

## Future Extensions

The modular architecture makes it easy to add support for:

### Additional Argo Workflows CRDs
- ClusterWorkflowTemplate
- CronWorkflow  
- WorkflowEventBinding

### Other Workflow Engines
- Tekton Pipelines
- GitHub Actions
- Jenkins X

### Example: Adding ClusterWorkflowTemplate Support

```go
// In pkg/kor/externaldeps/argo_workflows.go
type ClusterWorkflowTemplateScanner struct {
    gvr schema.GroupVersionResource
}

func NewClusterWorkflowTemplateScanner() *ClusterWorkflowTemplateScanner {
    return &ClusterWorkflowTemplateScanner{
        gvr: schema.GroupVersionResource{
            Group:    "argoproj.io",
            Version:  "v1alpha1", 
            Resource: "clusterworkflowtemplates",
        },
    }
}

// Implement ExternalResourceScanner interface...
```

```go
// In pkg/kor/externaldeps/registry.go
func registerDefaultScanners() {
    globalRegistry.RegisterScanner(NewWorkflowTemplateScanner())
    globalRegistry.RegisterScanner(NewClusterWorkflowTemplateScanner()) // Add this
}
```

## Troubleshooting

### Common Issues

**1. WorkflowTemplate CRD not detected**
- Verify Argo Workflows is installed: `kubectl get crd workflowtemplates.argoproj.io`
- Check Kor has permissions to list WorkflowTemplates

**2. Resources still marked as unused**
- Verify WorkflowTemplate references use correct field names
- Check if resources are in the same namespace as WorkflowTemplate
- Enable verbose logging to see scanning activity

**3. Performance impact**
- Integration only runs when WorkflowTemplate CRD exists
- Check cluster resource usage if scanning is slow
- Consider namespace-specific scanning for large clusters

### Debug Information

Add debug logging to see external scanner activity:
```go
// In your code
registry := externaldeps.GetGlobalRegistry()
enabled, err := registry.GetEnabledScanners(ctx, clientset, dynamicClient)
fmt.Printf("Enabled external scanners: %v\n", enabled)
```

## Contributing

To contribute to the Argo Workflows integration:

1. Follow the existing code patterns in `pkg/kor/externaldeps/`
2. Add comprehensive unit tests for new scanners
3. Update this documentation for new features
4. Ensure backward compatibility with existing Kor functionality

## Security Considerations

- The integration uses the same RBAC permissions as the main Kor application
- No additional cluster permissions required beyond standard Kor requirements
- CRD content is parsed as unstructured data - no direct deserialization
- Follows Kubernetes security best practices for CRD access
