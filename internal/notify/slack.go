package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
)

type SlackClient struct {
	http      *http.Client
	webhook   string

	// Optional: for attachments (Slack incoming webhooks can't upload files).
	botToken string
	channel  string
}

func NewSlackClient(httpc *http.Client, webhookURL, botToken, channel string) (*SlackClient, error) {
	if webhookURL == "" && botToken == "" {
		return nil, errors.New("slack webhook url or bot token required")
	}
	if webhookURL != "" {
		if _, err := url.Parse(webhookURL); err != nil {
			return nil, err
		}
	}
	return &SlackClient{http: httpc, webhook: webhookURL, botToken: botToken, channel: channel}, nil
}

func (s *SlackClient) Name() string { return "slack" }
func (s *SlackClient) MaxTextChars() int { return 3500 }
func (s *SlackClient) MaxAttachBytes() int { return 20000000 } // only used when file upload token is set

func (s *SlackClient) SendText(text string) error {
	if s.webhook == "" {
		return errors.New("slack incoming webhook url not set")
	}
	body := map[string]any{
		"text": text,
		"mrkdwn": true,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.webhook, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}

func (s *SlackClient) SendFile(filename string, contentType string, data []byte, caption string) error {
	// Slack file upload requires Web API token + channel.
	if s.botToken == "" || s.channel == "" {
		// Fallback: send as text
		return s.SendText(caption + "\n" + WrapCodeBlockMarkdown(string(data)))
	}

	// Slack Web API: files.upload
	api := "https://slack.com/api/files.upload"

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	_ = w.WriteField("channels", s.channel)
	_ = w.WriteField("initial_comment", caption)

	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	_, _ = fw.Write(data)
	_ = w.Close()

	req, _ := http.NewRequest("POST", api, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.botToken)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}
