package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/yonahd/kor/pkg/common"
)

type SendMessageToSlack interface {
	SendToSlack(opts common.Opts, outputBuffer string) error
}

type SlackPayload struct {
	Text string `json:"text"`
}

type SlackMessage struct {
}

func SendToSlack(sm SendMessageToSlack, opts common.Opts, outputBuffer string) error {
	return sm.SendToSlack(opts, outputBuffer)
}

func (sm SlackMessage) SendToSlack(opts common.Opts, outputBuffer string) error {
	if opts.WebhookURL != "" {

		// Prepare payload safely
		slackPayload := SlackPayload{Text: outputBuffer}
		payload, err := json.Marshal(slackPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal Slack payload: %w", err)
		}

		_, err = http.Post(opts.WebhookURL, "application/json", bytes.NewBuffer(payload))

		if err != nil {
			return err
		}
		return nil
	} else if opts.Channel != "" && opts.Token != "" {
		fmt.Printf("Sending message to Slack channel %s...\n", opts.Channel)
		outputFilePath, _ := writeOutputToFile(outputBuffer)

		var formData bytes.Buffer
		writer := multipart.NewWriter(&formData)

		fileWriter, err := writer.CreateFormFile("file", outputFilePath)
		if err != nil {
			return err
		}
		file, err := os.Open(outputFilePath)
		if err != nil {
			return err
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Printf("failed to close file: %v\n", err)
			}
		}()
		_, err = io.Copy(fileWriter, file)
		if err != nil {
			return err
		}

		if err := writer.WriteField("channels", opts.Channel); err != nil {
			return err
		}

		if err := writer.Close(); err != nil {
			return err
		}

		req, err := http.NewRequest("POST", "https://slack.com/api/files.upload", &formData)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+opts.Token)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("failed to close response body: %v\n", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("slack API returned non-OK status code: %d", resp.StatusCode)
		}

		return nil
	} else {
		return errors.New("SlackOpts must contain either WebhookURL or Channel and Token")
	}
}

func writeOutputToFile(outputBuffer string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user's home directory: %v", err)
	}

	outputFileName := "kor-scan-results.txt"
	outputFilePath := filepath.Join(homeDir, outputFileName)

	file, err := os.Create(outputFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("failed to close file: %v\n", err)
		}
	}()

	_, err = file.WriteString(outputBuffer)
	if err != nil {
		return "", fmt.Errorf("failed to write output to file: %v", err)
	}

	return outputFilePath, nil
}
