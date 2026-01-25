package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/yonahd/kor/pkg/common"
)

var (
	channel      = "test"
	outputBuffer = "Test!"
	token        = "xoxb-..."
)

func TestSendToSlack_EmptyBuffer(t *testing.T) {
	opts := common.Opts{
		WebhookURL: "https://hooks.slack.com/services/test",
	}

	err := SendToSlack(SlackMessage{}, opts, "")
	if err != nil {
		t.Errorf("Expected nil for empty buffer, got %v", err)
	}
}

func TestSendToSlack_BadOpts(t *testing.T) {
	testCases := []struct {
		Name string
		Opts common.Opts
	}{
		{
			Name: "Empty options",
			Opts: common.Opts{},
		},
		{
			Name: "No channel",
			Opts: common.Opts{Token: "xoxb-..."},
		},
		{
			Name: "No token",
			Opts: common.Opts{Channel: channel},
		},
	}

	for _, tc := range testCases {
		err := SendToSlack(SlackMessage{}, tc.Opts, outputBuffer)
		if err == nil {
			t.Errorf("Expected error for empty Opts, got nil")
		}
	}
}

func TestSendToSlack_Webhook_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Got unexpected method: %s", r.Method)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Got unexpected content type: %s", contentType)
		}

		body, _ := io.ReadAll(r.Body)
		var payload SlackPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("Failed to unmarshal payload: %v", err)
		}
		if payload.Channel != "" {
			t.Errorf("Expected payload channel to be nil, got '%s'", payload.Channel)
		}
		if payload.Text != outputBuffer {
			t.Errorf("Expected payload text to be '%s', got '%s'", outputBuffer, payload.Text)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := common.Opts{
		WebhookURL: server.URL,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestSendToSlack_Webhook_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	opts := common.Opts{
		WebhookURL: server.URL,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestSendToSlack_API_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Got unexpected method: %s", r.Method)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Got unexpected content type: %s", contentType)
		}
		authorization := r.Header.Get("Authorization")
		if !strings.HasSuffix(authorization, token) {
			t.Errorf("Bad authorization header format: %s", authorization)
		}

		body, _ := io.ReadAll(r.Body)
		var payload SlackPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("Failed to unmarshal payload: %v", err)
		}

		if payload.Channel != channel {
			t.Errorf("Expected payload channel to be '%s', got '%s'", channel, payload.Channel)
		}
		if payload.Text != outputBuffer {
			t.Errorf("Expected payload text to be '%s', got '%s'", outputBuffer, payload.Text)
		}

		w.WriteHeader(http.StatusOK)
		response := SlackAPIResponse{Ok: true}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	originalURL := SlackAPIURL
	SlackAPIURL = server.URL
	defer func() { SlackAPIURL = originalURL }()

	opts := common.Opts{
		Token:   token,
		Channel: channel,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestSendToSlack_API_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := SlackAPIResponse{
			Ok:    false,
			Error: "invalid_auth",
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	originalURL := SlackAPIURL
	SlackAPIURL = server.URL
	defer func() { SlackAPIURL = originalURL }()

	opts := common.Opts{
		Token:   token,
		Channel: channel,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestWriteOutputToFile(t *testing.T) {
	outputBuffer := bytes.Buffer{}
	outputBuffer.WriteString("This is a test output.\n")
	expectedOutput := outputBuffer.String()

	outputFilePath, err := writeOutputToFile(expectedOutput)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if _, err := os.Stat(outputFilePath); os.IsNotExist(err) {
		t.Errorf("Expected output file to exist, got error: %v", err)
	}

	fileContent, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Errorf("Failed to read output file: %v", err)
	}

	if string(fileContent) != expectedOutput {
		t.Errorf("Expected file content:\n%s\nGot:\n%s", expectedOutput, string(fileContent))
	}
}
