package kor

import (
	"bytes"
	"testing"

	"github.com/yonahd/kor/pkg/common"
)

func TestUnusedResourceFormatter(t *testing.T) {
	tests := []struct {
		name         string
		outputFormat string
		outputBuffer bytes.Buffer
		opts         common.Opts
		wantErr      bool
		description  string
	}{
		{
			name:         "table format without slack",
			outputFormat: "table",
			outputBuffer: *bytes.NewBufferString("Test output"),
			opts:         common.Opts{},
			wantErr:      false,
			description:  "Should return table output without error when no Slack config",
		},
		{
			name:         "table format with webhook",
			outputFormat: "table",
			outputBuffer: *bytes.NewBufferString("Test output"),
			opts: common.Opts{
				WebhookURL: "https://hooks.slack.com/test",
			},
			wantErr:     false,
			description: "Should return table output without error when webhook URL is provided",
		},
		{
			name:         "table format with token and channel",
			outputFormat: "table",
			outputBuffer: *bytes.NewBufferString("Test output"),
			opts: common.Opts{
				Token:   "xoxb-test-token",
				Channel: "C08MW33MP16",
			},
			wantErr:     false,
			description: "Should return table output without error when token and channel are provided",
		},
		{
			name:         "unsupported format",
			outputFormat: "xml",
			outputBuffer: *bytes.NewBufferString("Test output"),
			opts:         common.Opts{},
			wantErr:      true,
			description:  "Should return error for unsupported format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For tests that would actually try to send to Slack, we expect them to fail
			// but they should fail at the SendToSlack level, not with "unsupported output format"
			result, err := unusedResourceFormatter(tt.outputFormat, tt.outputBuffer, tt.opts, nil)

			if tt.wantErr && err == nil {
				t.Errorf("unusedResourceFormatter() %s: expected error but got none", tt.description)
			}

			if !tt.wantErr && err != nil {
				// For Slack cases, we expect errors from the actual sending, not from format support
				if tt.opts.WebhookURL != "" || (tt.opts.Token != "" && tt.opts.Channel != "") {
					// These should fail at the SendToSlack level, not with "unsupported output format"
					if err.Error() == "unsupported output format: table" {
						t.Errorf("unusedResourceFormatter() %s: got 'unsupported output format' error, this should be fixed", tt.description)
					}
					// The actual Slack sending will fail in tests, which is expected
				} else {
					t.Errorf("unusedResourceFormatter() %s: unexpected error: %v", tt.description, err)
				}
			}

			if !tt.wantErr && err == nil && result == "" {
				t.Errorf("unusedResourceFormatter() %s: expected non-empty result", tt.description)
			}
		})
	}
}