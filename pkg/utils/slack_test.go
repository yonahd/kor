package utils

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/yonahd/kor/pkg/common"
)

type SendToSlackTestCase struct {
	Name         string
	Opts         common.Opts
	OutputBuffer string
}

var testCases = []SendToSlackTestCase{
	{
		Name: "Test using WebhookURL",
		Opts: common.Opts{
			WebhookURL: "slack.webhookurl.com",
		},
		OutputBuffer: "Test message",
	},
	{
		Name: "Test using Channel and Token",
		Opts: common.Opts{
			Channel: "your_channel",
			Token:   "your_token",
		},
		OutputBuffer: "Test message",
	},
	{
		Name:         "Test with empty Opts",
		Opts:         common.Opts{},
		OutputBuffer: "Test message",
	},
}

func TestSendToSlack(t *testing.T) {
	for _, tc := range testCases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := SendToSlack(SlackMessage{}, tc.Opts, tc.OutputBuffer); err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		}))

		defer server.Close()
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
