package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/yonahd/kor/pkg/common"
)

var SlackAPIURL = "https://slack.com/api"

type SendMessageToSlack interface {
	SendToSlack(opts common.Opts, outputBuffer string) error
}

type SlackPayload struct {
	Text    string `json:"text"`
	Channel string `json:"channel,omitempty"`
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
			return fmt.Errorf("failed to marshal payload: %w", err)
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

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non-OK status code: %d, body: %s", resp.StatusCode, string(body))
		}

		return nil
	} else if opts.Channel != "" && opts.Token != "" {
		fmt.Printf("Sending message to Slack channel %s...\n", opts.Channel)
		slackPayload := SlackPayload{
			Text:    outputBuffer,
			Channel: opts.Channel,
		}

		payload, err := json.Marshal(slackPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		req, err := http.NewRequest("POST", SlackAPIURL+"/chat.postMessage", bytes.NewBuffer(payload))
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
			return fmt.Errorf("non-OK status code: %d, body: %s", resp.StatusCode, string(body))
		}

		var slackResp SlackAPIResponse
		if err := json.Unmarshal(body, &slackResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !slackResp.Ok {
			return fmt.Errorf("API error: %s", slackResp.Error)
		}

		return nil
	} else {
		return errors.New("SlackOpts must contain either WebhookURL or Channel and Token")
	}
}
