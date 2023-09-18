package kor

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSendToSlackWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()

	webhookURL := server.URL
	message := "Test message"
	err := SendToSlackWebhook(webhookURL, message)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func createTestFile(t *testing.T) string {
	content := "This is a test file content."
	filePath := "testfile.txt"
	err := os.WriteFile(filePath, []byte(content), 0666)
	if err != nil {
		t.Fatal(err)
	}
	return filePath
}

func TestSendFileToSlack(t *testing.T) {
	testFile := createTestFile(t)
	defer os.Remove(testFile)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "multipart/form-data; boundary=" {
			t.Errorf("Expected Content-Type 'multipart/form-data', got '%s'", r.Header.Get("Content-Type"))
		}

		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			t.Errorf("Error parsing multipart form data: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("Error getting file from form data: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		initialComment := r.FormValue("initial_comment")
		channels := r.FormValue("channels")

		if initialComment != "Test comment" {
			t.Errorf("Expected initial_comment 'Test comment', got '%s'", initialComment)
		}
		if channels != "test_channel" {
			t.Errorf("Expected channels 'test_channel', got '%s'", channels)
		}

		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()

	initialComment := "Test comment"
	channels := "test_channel"
	token := "test_token"

	err := SendFileToSlack(testFile, initialComment, channels, token)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestWriteOutputToFile(t *testing.T) {
	outputBuffer := bytes.Buffer{}
	outputBuffer.WriteString("This is a test output.\n")
	expectedOutput := outputBuffer.String()

	outputFilePath, err := writeOutputToFile(outputBuffer)
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
