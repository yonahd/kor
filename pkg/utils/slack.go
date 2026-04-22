package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yonahd/kor/pkg/common"
)

const slackEndpointURL = "https://slack.com/api/chat.postMessage"

var client = &http.Client{
	Timeout: 5 * time.Minute,
}

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
		slackPayload := SlackPayload{
			Text: outputBuffer,
		}
		payload, err := json.Marshal(slackPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		response, err := client.Post(opts.WebhookURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				fmt.Printf("failed to close response body: %v\n", err)
			}
		}()

		_, err = io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("non-OK status code: %d", response.StatusCode)
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

		request, err := http.NewRequest(http.MethodPost, slackEndpointURL, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		request.Header.Set("Authorization", "Bearer "+opts.Token)
		request.Header.Set("Content-Type", "application/json")

		response, err := client.Do(request)
		if err != nil {
			return err
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				fmt.Printf("failed to close response body: %v\n", err)
			}
		}()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("non-OK status code: %d, body: %s", response.StatusCode, string(body))
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
