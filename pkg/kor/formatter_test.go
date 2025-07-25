package kor

import (
	"bytes"
	"testing"

	"github.com/yonahd/kor/pkg/common"
)

func TestUnusedResourceFormatterSlackFix(t *testing.T) {
	tests := []struct {
		name         string
		outputFormat string
		outputBuffer bytes.Buffer
		opts         common.Opts
		wantErr      bool
		description  string
	}{
		{
			name:         "table format without slack config",
			outputFormat: "table",
			outputBuffer: *bytes.NewBufferString("Test output"),
			opts:         common.Opts{},
			wantErr:      false,
			description:  "Should return table output without error when no Slack config",
		},
		{
			name:         "table format with token and channel - should fail but not with unsupported format error",
			outputFormat: "table",
			outputBuffer: *bytes.NewBufferString("Test output"),
			opts: common.Opts{
				Token:   "xoxb-test-token",
				Channel: "C08MW33MP16",
			},
			wantErr:     true,  // Will fail because of network call, but not with "unsupported format" error
			description: "Should fail at Slack API level, not with unsupported format error",
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
			result, err := unusedResourceFormatter(tt.outputFormat, tt.outputBuffer, tt.opts, []byte("{}"))
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				// For the token/channel case, make sure it's NOT the "unsupported format" error
				if tt.outputFormat == "table" && err != nil && err.Error() == "unsupported output format: table" {
					t.Errorf("Got 'unsupported output format' error, which should be fixed. Error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result == "" {
					t.Errorf("Expected non-empty result")
				}
			}
		})
	}
}