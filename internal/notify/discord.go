package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
)

type DiscordClient struct {
	http *http.Client
	hook string
}

func NewDiscordClient(httpc *http.Client, webhookURL string, maxAttachBytes int) (*DiscordClient, error) {
	if webhookURL == "" {
		return nil, errors.New("discord webhook url is empty")
	}
	if _, err := url.Parse(webhookURL); err != nil {
		return nil, err
	}
	return &DiscordClient{http: httpc, hook: webhookURL}, nil
}

func (d *DiscordClient) Name() string { return "discord" }
func (d *DiscordClient) MaxTextChars() int { return 1900 } // keep margin under 2000
func (d *DiscordClient) MaxAttachBytes() int { return 8000000 }

func (d *DiscordClient) SendText(text string) error {
	body := map[string]any{
		"content": text,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", d.hook, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}

func (d *DiscordClient) SendFile(filename string, contentType string, data []byte, caption string) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	payload := map[string]any{
		"content": caption,
	}
	payloadJSON, _ := json.Marshal(payload)

	_ = w.WriteField("payload_json", string(payloadJSON))
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	_, _ = fw.Write(data)
	_ = w.Close()

	req, _ := http.NewRequest("POST", d.hook, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}
