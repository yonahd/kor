package externaldeps

import (
	"context"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ResourceReferences holds references to different types of Kubernetes resources
type ResourceReferences struct {
	ConfigMaps []string
	Secrets    []string
	PVCs       []string
	// We can extend this for other resource types in the future
}

// ExternalResourceScanner defines the interface for scanning external CRDs
// that may reference standard Kubernetes resources
type ExternalResourceScanner interface {
	// GetName returns a human-readable name for this scanner
	GetName() string

	// IsEnabled checks if this scanner should be activated
	// (e.g., by checking if the required CRD exists in the cluster)
	IsEnabled(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface) (bool, error)

	// ScanNamespace scans a specific namespace for resource references
	ScanNamespace(ctx context.Context, namespace string, clientset kubernetes.Interface, dynamicClient dynamic.Interface) (*ResourceReferences, error)

	// GetSupportedResources returns the list of resource types this scanner can find references to
	GetSupportedResources() []string
}

// ScannerRegistry manages multiple external resource scanners
type ScannerRegistry struct {
	scanners []ExternalResourceScanner
}

// NewScannerRegistry creates a new scanner registry
func NewScannerRegistry() *ScannerRegistry {
	return &ScannerRegistry{
		scanners: make([]ExternalResourceScanner, 0),
	}
}

// RegisterScanner registers a new external resource scanner
func (r *ScannerRegistry) RegisterScanner(scanner ExternalResourceScanner) {
	r.scanners = append(r.scanners, scanner)
}

// ScanNamespace scans a namespace using all registered and enabled scanners
func (r *ScannerRegistry) ScanNamespace(ctx context.Context, namespace string, clientset kubernetes.Interface, dynamicClient dynamic.Interface) (*ResourceReferences, error) {
	aggregatedRefs := &ResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}

	for _, scanner := range r.scanners {
		enabled, err := scanner.IsEnabled(ctx, clientset, dynamicClient)
		if err != nil {
			return nil, err
		}

		if !enabled {
			continue
		}

		refs, err := scanner.ScanNamespace(ctx, namespace, clientset, dynamicClient)
		if err != nil {
			return nil, err
		}

		// Aggregate the references
		aggregatedRefs.ConfigMaps = append(aggregatedRefs.ConfigMaps, refs.ConfigMaps...)
		aggregatedRefs.Secrets = append(aggregatedRefs.Secrets, refs.Secrets...)
		aggregatedRefs.PVCs = append(aggregatedRefs.PVCs, refs.PVCs...)
	}

	return aggregatedRefs, nil
}

// GetEnabledScanners returns a list of currently enabled scanners
func (r *ScannerRegistry) GetEnabledScanners(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface) ([]ExternalResourceScanner, error) {
	enabled := make([]ExternalResourceScanner, 0)

	for _, scanner := range r.scanners {
		isEnabled, err := scanner.IsEnabled(ctx, clientset, dynamicClient)
		if err != nil {
			return nil, err
		}

		if isEnabled {
			enabled = append(enabled, scanner)
		}
	}

	return enabled, nil
}
