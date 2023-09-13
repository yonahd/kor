package kor

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func SendToSlackWebhook(webhookURL string, message string) error {
	payload := []byte(`{"text": "` + message + `"}`)

	_, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
	return err
}

func SendFileToSlack(filePath string, initialComment string, channels string, token string) error {
	var formData bytes.Buffer
	writer := multipart.NewWriter(&formData)

	fileWriter, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return err
	}

	if err := writer.WriteField("initial_comment", initialComment); err != nil {
		return err
	}

	if err := writer.WriteField("channels", channels); err != nil {
		return err
	}

	writer.Close()

	req, err := http.NewRequest("POST", "https://slack.com/api/files.upload", &formData)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned non-OK status code: %d", resp.StatusCode)
	}

	return nil
}

func writeOutputToFile(outputBuffer bytes.Buffer) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Failed to get user's home directory: %v", err)
	}

	outputFileName := "output.txt"
	outputFilePath := filepath.Join(homeDir, outputFileName)

	file, err := os.Create(outputFilePath)
	if err != nil {
		return "", fmt.Errorf("Failed to create output file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(outputBuffer.String())
	if err != nil {
		return "", fmt.Errorf("Failed to write output to file: %v", err)
	}

	return outputFilePath, nil
}
