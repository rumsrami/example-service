package logger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// =========================================================================
// SlackLogging

// SlackOutput wraps the app logger
type SlackOutput struct {
	Webhook string
	Client  *http.Client
}

// NewSlackOutput returns a new slack logger
func NewSlackOutput(webhookURL string) SlackOutput {
	client := &http.Client{Timeout: 10 * time.Second}
	return SlackOutput{
		Webhook: webhookURL,
		Client:  client,
	}
}

func (sl SlackOutput) Write(p []byte) (n int, err error) {
	var reqBody = struct {
		Text string `json:"text"`
	}{
		Text: string(p),
	}

	slackBody, err := json.Marshal(reqBody)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, sl.Webhook, bytes.NewBuffer(slackBody))
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := sl.Client.Do(req)
	if err != nil {
		return 0, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return
	}

	return len(p), nil
}