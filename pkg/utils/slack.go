package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type SlackAPIResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type SlackMessage struct {
}

func SendToSlack(sm SendMessageToSlack, opts common.Opts, outputBuffer string) error {
	return sm.SendToSlack(opts, outputBuffer)
}

func (sm SlackMessage) SendToSlack(opts common.Opts, outputBuffer string) error {
	if outputBuffer == "" {
		return nil
	}

	if opts.WebhookURL != "" {

		// Prepare payload safely
		slackPayload := SlackPayload{Text: outputBuffer}
		payload, err := json.Marshal(slackPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal Slack payload: %w", err)
		}

		resp, err := http.Post(opts.WebhookURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("failed to close response body: %v\n", err)
			}
		}()

		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("slack webhook returned non-OK status code: %d", resp.StatusCode)
		}

		return nil
	} else if opts.Channel != "" && opts.Token != "" {
		fmt.Printf("Sending message to Slack channel %s...\n", opts.Channel)
		messagePayload := map[string]interface{}{
			"channel": opts.Channel,
			"text":    outputBuffer,
		}

		payload, err := json.Marshal(messagePayload)
		if err != nil {
			return fmt.Errorf("failed to marshal Slack message payload: %w", err)
		}

		req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+opts.Token)
		req.Header.Set("Content-Type", "application/json")

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

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("slack API returned non-OK status code: %d, body: %s", resp.StatusCode, string(body))
		}

		var slackResp SlackAPIResponse
		if err := json.Unmarshal(body, &slackResp); err != nil {
			return fmt.Errorf("failed to parse Slack API response: %w", err)
		}

		if !slackResp.Ok {
			return fmt.Errorf("slack API error: %s", slackResp.Error)
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
