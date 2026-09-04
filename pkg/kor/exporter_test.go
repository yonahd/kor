package kor

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
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
