package externaldeps

import (
	"sync"
)

var (
	globalRegistry *ScannerRegistry
	once           sync.Once
)

// GetGlobalRegistry returns the global scanner registry instance
// This follows the singleton pattern to ensure we have one registry across the application
func GetGlobalRegistry() *ScannerRegistry {
	once.Do(func() {
		globalRegistry = NewScannerRegistry()
		// Register all available scanners
		registerDefaultScanners()
	})
	return globalRegistry
}

// registerDefaultScanners registers all the default external resource scanners
func registerDefaultScanners() {
	// Register Argo Workflows WorkflowTemplate scanner
	globalRegistry.RegisterScanner(NewWorkflowTemplateScanner())

	// Future scanners can be registered here:
	// globalRegistry.RegisterScanner(NewArgoRolloutsScanner())
	// globalRegistry.RegisterScanner(NewTektonPipelineScanner())
	// etc.
}
