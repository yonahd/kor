package kor

import (
	"bytes"
	"net/http"
)

func sendToSlack(webhookURL string, message string) error {
	payload := []byte(`{"text": "` + message + `"}`)

	_, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
	return err
}
