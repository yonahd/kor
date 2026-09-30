package kor

import (
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func TestSetOrphanedResourceMetricsGroupByNamespace(t *testing.T) {
	orphanedResourcesGauge.Reset()

	data := map[string]map[string][]string{
		"default": {
			"ConfigMap": {"script"},
		},
	}

	setOrphanedResourceMetrics(data, "namespace")

	metric := &dto.Metric{}
	if err := orphanedResourcesGauge.WithLabelValues("ConfigMap", "default", "script").Write(metric); err != nil {
		t.Fatalf("failed writing metric: %v", err)
	}
	value := metric.GetGauge().GetValue()
	if value != 1 {
		t.Fatalf("expected metric value 1, got %v", value)
	}
}

func TestSetOrphanedResourceMetricsGroupByResource(t *testing.T) {
	orphanedResourcesGauge.Reset()

	data := map[string]map[string][]string{
		"ConfigMap": {
			"default": {"script"},
		},
	}

	setOrphanedResourceMetrics(data, "resource")

	metric := &dto.Metric{}
	if err := orphanedResourcesGauge.WithLabelValues("ConfigMap", "default", "script").Write(metric); err != nil {
		t.Fatalf("failed writing metric: %v", err)
	}
	value := metric.GetGauge().GetValue()
	if value != 1 {
		t.Fatalf("expected metric value 1, got %v", value)
	}
}

func TestGetUnusedResourcesAll(t *testing.T) {
	clientset := createTestMultiResources(t)

	filterOpts := &filters.Options{
		IncludeNamespaces: []string{testNamespace},
	}
	opts := common.Opts{GroupBy: "namespace"}
	output, err := getUnusedResources(filterOpts, clientset, nil, nil, "json", opts, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "configmap-1") {
		t.Errorf("Expected configmap-1 in output, got %s", output)
	}
}

func TestGetUnusedResourcesAllExplicit(t *testing.T) {
	clientset := createTestMultiResources(t)

	filterOpts := &filters.Options{
		IncludeNamespaces: []string{testNamespace},
	}
	opts := common.Opts{GroupBy: "namespace"}
	output, err := getUnusedResources(filterOpts, clientset, nil, nil, "json", opts, []string{"all"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "configmap-1") {
		t.Errorf("Expected configmap-1 in output, got %s", output)
	}
}

func TestGetUnusedResourcesSpecific(t *testing.T) {
	clientset := createTestMultiResources(t)

	opts := common.Opts{GroupBy: "namespace"}
	output, err := getUnusedResources(&filters.Options{}, clientset, nil, nil, "json", opts, []string{"configmap"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "configmap-1") {
		t.Errorf("Expected configmap-1 in output, got %s", output)
	}
}
