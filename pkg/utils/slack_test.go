package utils

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"

	"github.com/yonahd/kor/pkg/common"
)

const (
	channel      = "test"
	outputBuffer = "Test!"
	token        = "xoxb-..."
	webhookURL   = "https://hooks.slack.com/services/test"
	endpointURL  = slackEndpointURL
)

func TestSendToSlack_EmptyBuffer(t *testing.T) {
	opts := common.Opts{
		WebhookURL: webhookURL,
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
			Opts: common.Opts{
				Token: token,
			},
		},
		{
			Name: "No token",
			Opts: common.Opts{
				Channel: channel,
			},
		},
	}

	for _, tc := range testCases {
		err := SendToSlack(SlackMessage{}, tc.Opts, outputBuffer)
		if err == nil {
			t.Errorf("Test %s: expected error, got nil", tc.Name)
		}
	}
}

func TestSendToSlack_Webhook_Success(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	httpmock.RegisterResponder(http.MethodPost, webhookURL,
		func(r *http.Request) (*http.Response, error) {
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Got unexpected content type: %s", contentType)
			}

			var payload SlackPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("Failed to unmarshal payload: %v", err)
			}
			if payload.Channel != "" {
				t.Errorf("Expected payload channel to be empty, got '%s'", payload.Channel)
			}
			if payload.Text != outputBuffer {
				t.Errorf("Expected payload text to be '%s', got '%s'", outputBuffer, payload.Text)
			}

			return httpmock.NewStringResponse(http.StatusOK, "ok"), nil
		},
	)

	opts := common.Opts{
		WebhookURL: webhookURL,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	calls := httpmock.GetTotalCallCount()
	if calls != 1 {
		t.Errorf("Expected 1 HTTP call, got %d", calls)
	}
}

func TestSendToSlack_Webhook_Error(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	httpmock.RegisterResponder(
		http.MethodPost,
		webhookURL,
		httpmock.NewBytesResponder(
			http.StatusInternalServerError,
			[]byte{},
		),
	)

	opts := common.Opts{
		WebhookURL: webhookURL,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	calls := httpmock.GetTotalCallCount()
	if calls != 1 {
		t.Errorf("Expected 1 HTTP call, got %d", calls)
	}
}

func TestSendToSlack_API_Success(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	httpmock.RegisterResponder(http.MethodPost, endpointURL,
		func(r *http.Request) (*http.Response, error) {
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Got unexpected content type: %s", contentType)
			}
			authorization := r.Header.Get("Authorization")
			if !strings.HasSuffix(authorization, token) {
				t.Errorf("Bad authorization header format: %s", authorization)
			}

			var payload SlackPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("Failed to unmarshal payload: %v", err)
			}

			if payload.Channel != channel {
				t.Errorf("Expected payload channel to be '%s', got '%s'", channel, payload.Channel)
			}
			if payload.Text != outputBuffer {
				t.Errorf("Expected payload text to be '%s', got '%s'", outputBuffer, payload.Text)
			}

			response, _ := httpmock.NewJsonResponse(
				http.StatusOK,
				SlackAPIResponse{
					Ok: true,
				},
			)
			return response, nil
		},
	)

	opts := common.Opts{
		Token:   token,
		Channel: channel,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	calls := httpmock.GetTotalCallCount()
	if calls != 1 {
		t.Errorf("Expected 1 HTTP call, got %d", calls)
	}
}

func TestSendToSlack_API_Error(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	invalidAuth := "invalid_auth"
	httpmock.RegisterResponder(http.MethodPost, endpointURL,
		func(req *http.Request) (*http.Response, error) {
			response, _ := httpmock.NewJsonResponse(
				http.StatusOK,
				SlackAPIResponse{
					Ok:    false,
					Error: invalidAuth,
				},
			)
			return response, nil
		},
	)

	opts := common.Opts{
		Token:   token,
		Channel: channel,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err != nil && !strings.HasSuffix(err.Error(), invalidAuth) {
		t.Errorf("Expected error to be '%s', got '%s'", invalidAuth, err.Error())
	}
	calls := httpmock.GetTotalCallCount()
	if calls != 1 {
		t.Errorf("Expected 1 HTTP call, got %d", calls)
	}
}

func TestSendToSlack_API_NonOKStatus(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	httpmock.RegisterResponder(
		http.MethodPost,
		endpointURL,
		httpmock.NewBytesResponder(
			http.StatusInternalServerError,
			[]byte{},
		),
	)

	opts := common.Opts{
		Token:   token,
		Channel: channel,
	}

	err := SendToSlack(SlackMessage{}, opts, outputBuffer)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	calls := httpmock.GetTotalCallCount()
	if calls != 1 {
		t.Errorf("Expected 1 HTTP call, got %d", calls)
	}
}
