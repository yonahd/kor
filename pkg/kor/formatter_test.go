package kor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"

	"github.com/yonahd/kor/pkg/common"
)

func TestGetTableRow(t *testing.T) {
	row := getTableRow(0, "ConfigMap", "configmap-1")
	expected := []string{"1", "ConfigMap", "configmap-1"}
	if !slices.Equal(row, expected) {
		t.Errorf("Expected %v, got %v", expected, row)
	}
}

func TestGetTableRowWithReason(t *testing.T) {
	row := getTableRowResourceInfo(0, "ConfigMap", ResourceInfo{Name: "configmap-1", Reason: "unused"}, true)
	expected := []string{"1", "ConfigMap", "configmap-1", "unused"}
	if !slices.Equal(row, expected) {
		t.Errorf("Expected %v, got %v", expected, row)
	}
}

func TestGetTableHeader(t *testing.T) {
	if header := getTableHeader("namespace", false); len(header) != 3 {
		t.Errorf("Expected 3 columns for namespace group by, got %d", len(header))
	}
	if header := getTableHeader("namespace", true); len(header) != 4 {
		t.Errorf("Expected 4 columns for namespace group by with reason, got %d", len(header))
	}
	if header := getTableHeader("resource", false); len(header) != 3 {
		t.Errorf("Expected 3 columns for resource group by, got %d", len(header))
	}
	if header := getTableHeader("resource", true); len(header) != 4 {
		t.Errorf("Expected 4 columns for resource group by with reason, got %d", len(header))
	}
	if header := getTableHeader("invalid", false); header != nil {
		t.Errorf("Expected nil header for invalid group by, got %v", header)
	}
}

func TestFormatOutput(t *testing.T) {
	resources := map[string]map[string][]ResourceInfo{
		"test-namespace": {
			"ConfigMap": {{Name: "configmap-1", Reason: "unused"}},
		},
	}

	opts := common.Opts{GroupBy: "namespace"}
	buffer := FormatOutput(resources, opts)
	output := buffer.String()
	for _, expected := range []string{"test-namespace", "RESOURCE TYPE", "RESOURCE NAME", "configmap-1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestFormatOutputGroupByResource(t *testing.T) {
	ResourceKindList = map[string]ResourceKind{
		"configmap": {Plural: "configmaps"},
	}
	resources := map[string]map[string][]ResourceInfo{
		"configmap": {
			"test-namespace": {{Name: "configmap-1"}},
		},
	}

	opts := common.Opts{GroupBy: "resource"}
	buffer := FormatOutput(resources, opts)
	output := buffer.String()
	for _, expected := range []string{"Unused configmaps", "NAMESPACE", "configmap-1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestFormatOutputEmptyNamespace(t *testing.T) {
	resources := map[string]map[string][]ResourceInfo{
		"empty-namespace": {},
	}

	opts := common.Opts{GroupBy: "namespace", Verbose: true}
	buffer := FormatOutput(resources, opts)
	output := buffer.String()
	if !strings.Contains(output, "No unused resources found in the namespace") {
		t.Errorf("Expected verbose empty message, got %s", output)
	}

	opts.Verbose = false
	buffer = FormatOutput(resources, opts)
	if output := buffer.String(); output != "" {
		t.Errorf("Expected empty output when not verbose, got %s", output)
	}
}

func TestFormatOutputForResourceEmpty(t *testing.T) {
	ResourceKindList = map[string]ResourceKind{
		"configmap": {Plural: "configmaps"},
	}

	opts := common.Opts{GroupBy: "resource", Verbose: true}
	output := formatOutputForResource("configmap", map[string][]ResourceInfo{}, opts)
	if !strings.Contains(output, "No unused configmaps found") {
		t.Errorf("Expected verbose empty message, got %s", output)
	}

	opts.Verbose = false
	if output := formatOutputForResource("configmap", map[string][]ResourceInfo{}, opts); output != "" {
		t.Errorf("Expected empty output when not verbose, got %s", output)
	}
}

func TestFormatOutputAll(t *testing.T) {
	allDiffs := []ResourceDiff{
		{
			resourceType: "ConfigMap",
			diff:         []ResourceInfo{{Name: "configmap-1", Reason: "unused"}},
		},
	}

	opts := common.Opts{GroupBy: "namespace"}
	output := FormatOutputAll(testNamespace, allDiffs, opts)
	for _, expected := range []string{"test-namespace", "RESOURCE TYPE", "RESOURCE NAME", "configmap-1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestFormatOutputAllEmpty(t *testing.T) {
	opts := common.Opts{GroupBy: "namespace", Verbose: true}
	output := FormatOutputAll(testNamespace, nil, opts)
	if !strings.Contains(output, "No unused resources found in the namespace") {
		t.Errorf("Expected verbose empty message, got %s", output)
	}

	opts.Verbose = false
	if output := FormatOutputAll(testNamespace, nil, opts); output != "" {
		t.Errorf("Expected empty output when not verbose, got %s", output)
	}
}

func TestUnusedResourceFormatterTable(t *testing.T) {
	var outputBuffer bytes.Buffer
	outputBuffer.WriteString("ConfigMap configmap-1\n")

	output, err := unusedResourceFormatter("table", outputBuffer, common.Opts{}, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if output != "ConfigMap configmap-1\n" {
		t.Errorf("Expected table output, got %s", output)
	}
}

func TestUnusedResourceFormatterTableWithSlackWebhook(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	const webhookURL = "https://hooks.slack.com/services/test"
	httpmock.RegisterResponder(http.MethodPost, webhookURL, httpmock.NewStringResponder(http.StatusOK, "ok"))

	var outputBuffer bytes.Buffer
	outputBuffer.WriteString("ConfigMap configmap-1\n")

	opts := common.Opts{
		WebhookURL:  webhookURL,
		ClusterName: "test-cluster",
	}
	output, err := unusedResourceFormatter("table", outputBuffer, opts, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if output != "ConfigMap configmap-1\n" {
		t.Errorf("Expected table output, got %s", output)
	}
	if calls := httpmock.GetTotalCallCount(); calls != 1 {
		t.Errorf("Expected 1 slack call, got %d", calls)
	}
}

func TestUnusedResourceFormatterTableWithSlackAPIError(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	httpmock.RegisterResponder(
		http.MethodPost,
		"https://slack.com/api/chat.postMessage",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""),
	)

	var outputBuffer bytes.Buffer
	outputBuffer.WriteString("ConfigMap configmap-1\n")

	opts := common.Opts{
		Channel: "test-channel",
		Token:   "test-token",
	}
	_, err := unusedResourceFormatter("table", outputBuffer, opts, nil)
	if err == nil {
		t.Fatalf("Expected error sending to slack, got nil")
	}
	if !strings.Contains(err.Error(), "failed to send message to slack") {
		t.Errorf("Expected slack error, got %v", err)
	}
}

func TestUnusedResourceFormatterJSONNoReason(t *testing.T) {
	jsonResponse := []byte(`{"test-namespace":{"ConfigMap":[{"name":"configmap-1","reason":"unused"}]}}`)

	output, err := unusedResourceFormatter("json", bytes.Buffer{}, common.Opts{}, jsonResponse)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := map[string]map[string][]string{
		"test-namespace": {
			"ConfigMap": {"configmap-1"},
		},
	}
	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestUnusedResourceFormatterJSONWithReason(t *testing.T) {
	jsonResponse := []byte(`{"test-namespace":{"ConfigMap":[{"name":"configmap-1","reason":"unused"}]}}`)

	opts := common.Opts{ShowReason: true}
	output, err := unusedResourceFormatter("json", bytes.Buffer{}, opts, jsonResponse)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := map[string]map[string][]ResourceInfo{
		"test-namespace": {
			"ConfigMap": {{Name: "configmap-1", Reason: "unused"}},
		},
	}
	var actual map[string]map[string][]ResourceInfo
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling output: %v", err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestUnusedResourceFormatterInvalidJSON(t *testing.T) {
	_, err := unusedResourceFormatter("json", bytes.Buffer{}, common.Opts{}, []byte("invalid"))
	if err == nil {
		t.Fatalf("Expected error for invalid json, got nil")
	}
}

func TestUnusedResourceFormatterYAML(t *testing.T) {
	jsonResponse := []byte(`{"test-namespace":{"ConfigMap":[{"name":"configmap-1","reason":"unused"}]}}`)

	output, err := unusedResourceFormatter("yaml", bytes.Buffer{}, common.Opts{}, jsonResponse)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	for _, expected := range []string{"test-namespace:", "- configmap-1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestUnusedResourceFormatterYAMLWithReason(t *testing.T) {
	jsonResponse := []byte(`{"test-namespace":{"ConfigMap":[{"name":"configmap-1","reason":"unused"}]}}`)

	opts := common.Opts{ShowReason: true}
	output, err := unusedResourceFormatter("yaml", bytes.Buffer{}, opts, jsonResponse)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	for _, expected := range []string{"test-namespace:", "name: configmap-1", "reason: unused"} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestUnusedResourceFormatterUnsupportedFormat(t *testing.T) {
	_, err := unusedResourceFormatter("xml", bytes.Buffer{}, common.Opts{}, nil)
	if err == nil {
		t.Fatalf("Expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("Expected unsupported format error, got %v", err)
	}
}
